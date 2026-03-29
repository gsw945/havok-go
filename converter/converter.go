// Package converter parses HavokPhysics.d.ts with tree-sitter TypeScript and
// emits Go source scaffolding for each exported HP_* function via wazero.
//
// The converter understands the following TypeScript constructs:
//   - export type Alias = [T, T, ...]   (tuple type alias)
//   - declare enum Name { A, B, ... }   (enum declaration)
//   - interface HavokPhysicsWithBindings methods
//
// Calling convention derived from emscripten embind:
//   - Functions returning a tuple type use an *sret* first argument (i32 ptr).
//   - Functions returning a scalar (Result / void) return i32 or nothing directly.
//   - Tuple-typed parameters are passed as i32 pointers.
//   - Scalar `number` parameters are passed as f64.
//   - Scalar `boolean` parameters are passed as i32.
package converter

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	sitter "github.com/tree-sitter/go-tree-sitter"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// -----------------------------------------------------------------------------
// Domain model
// -----------------------------------------------------------------------------

// TypeKind classifies a TypeScript type.
type TypeKind int

const (
	KindUnknown    TypeKind = iota
	KindVoid                // void / undefined
	KindNumber              // number → f64
	KindBigInt              // bigint → uint64
	KindBoolean             // boolean → bool / i32
	KindEnum                // a declared enum
	KindTupleAlias          // an export type alias for a tuple
	KindPrimitive           // other recognized primitives
)

// TSType represents a resolved TypeScript type.
type TSType struct {
	Raw      string // original TypeScript spelling
	Kind     TypeKind
	Elements []TSType // non-empty for tuple types
}

// EnumDecl holds an enum name and its values.
type EnumDecl struct {
	Name   string
	Values []string
}

// TypeAlias holds a top-level `export type Alias = [...]` declaration.
type TypeAlias struct {
	Name     string
	Tuple    []TSType // element types
	IsHandle bool     // true for HP_*Id types (single bigint element)
}

// Param represents one method parameter.
type Param struct {
	Name   string
	TSType TSType
}

// Method represents a method signature in HavokPhysicsWithBindings.
type Method struct {
	Name       string
	Params     []Param
	ReturnType TSType
}

// Schema is the full parse result from a d.ts file.
type Schema struct {
	Enums   []EnumDecl
	Types   []TypeAlias
	typeMap map[string]*TypeAlias // name → alias
	enumSet map[string]bool
	Methods []Method
}

// -----------------------------------------------------------------------------
// Parser
// -----------------------------------------------------------------------------

// Parse reads a TypeScript declaration file and returns a Schema.
func Parse(path string) (*Schema, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("converter: reading %s: %w", path, err)
	}

	parser := sitter.NewParser()
	defer parser.Close()

	lang := sitter.NewLanguage(typescript.LanguageTypescript())
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("converter: set language: %w", err)
	}

	tree := parser.Parse(src, nil)
	defer tree.Close()

	s := &Schema{
		typeMap: make(map[string]*TypeAlias),
		enumSet: make(map[string]bool),
	}
	root := tree.RootNode()
	s.walkRoot(root, src)
	return s, nil
}

// -----------------------------------------------------------------------------
// AST walking
// -----------------------------------------------------------------------------

func (s *Schema) walkRoot(n *sitter.Node, src []byte) {
	for i := range n.ChildCount() {
		child := n.Child(i)
		switch child.Kind() {
		case "export_statement":
			s.handleExport(child, src)
		case "ambient_declaration": // declare enum / declare namespace
			s.handleAmbient(child, src)
		case "interface_declaration":
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				name := string(src[nameNode.StartByte():nameNode.EndByte()])
				if name == "HavokPhysicsWithBindings" {
					s.parseInterface(child, src)
				}
			}
		}
	}
}

func (s *Schema) handleExport(n *sitter.Node, src []byte) {
	for i := range n.ChildCount() {
		child := n.Child(i)
		switch child.Kind() {
		case "type_alias_declaration":
			s.parseTypeAlias(child, src)
		case "enum_declaration":
			s.parseEnum(child, src)
		case "ambient_declaration":
			s.handleAmbient(child, src)
		case "interface_declaration":
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				name := string(src[nameNode.StartByte():nameNode.EndByte()])
				if name == "HavokPhysicsWithBindings" {
					s.parseInterface(child, src)
				}
			}
		}
	}
}

func (s *Schema) handleAmbient(n *sitter.Node, src []byte) {
	for i := range n.ChildCount() {
		child := n.Child(i)
		switch child.Kind() {
		case "enum_declaration":
			s.parseEnum(child, src)
		case "interface_declaration":
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				name := string(src[nameNode.StartByte():nameNode.EndByte()])
				if name == "HavokPhysicsWithBindings" {
					s.parseInterface(child, src)
				}
			}
		}
	}
}

func (s *Schema) parseTypeAlias(n *sitter.Node, src []byte) {
	nameNode := n.ChildByFieldName("name")
	valueNode := n.ChildByFieldName("value")
	if nameNode == nil || valueNode == nil {
		return
	}
	name := string(src[nameNode.StartByte():nameNode.EndByte()])
	alias := &TypeAlias{Name: name}

	if valueNode.Kind() == "tuple_type" {
		for i := range valueNode.ChildCount() {
			el := valueNode.Child(i)
			if el.IsNamed() {
				alias.Tuple = append(alias.Tuple, s.resolveType(el, src))
			}
		}
	}

	// HP_*Id types are single-element bigint tuples = native handles
	if strings.HasPrefix(name, "HP_") && strings.HasSuffix(name, "Id") && len(alias.Tuple) == 1 && alias.Tuple[0].Kind == KindBigInt {
		alias.IsHandle = true
	}

	s.Types = append(s.Types, *alias)
	s.typeMap[name] = &s.Types[len(s.Types)-1]
}

func (s *Schema) parseEnum(n *sitter.Node, src []byte) {
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := string(src[nameNode.StartByte():nameNode.EndByte()])
	decl := EnumDecl{Name: name}

	body := n.ChildByFieldName("body")
	if body != nil {
		for i := range body.ChildCount() {
			member := body.Child(i)
			if member.Kind() == "enum_assignment" || member.Kind() == "property_identifier" {
				var memberName string
				if member.Kind() == "enum_assignment" {
					if mn := member.ChildByFieldName("name"); mn != nil {
						memberName = string(src[mn.StartByte():mn.EndByte()])
					}
				} else {
					memberName = string(src[member.StartByte():member.EndByte()])
				}
				if memberName != "" {
					decl.Values = append(decl.Values, memberName)
				}
			}
		}
	}

	s.Enums = append(s.Enums, decl)
	s.enumSet[name] = true
}

func (s *Schema) parseInterface(n *sitter.Node, src []byte) {
	body := n.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := range body.ChildCount() {
		member := body.Child(i)
		if member.Kind() == "method_signature" {
			s.parseMethodSignature(member, src)
		}
	}
}

func (s *Schema) parseMethodSignature(n *sitter.Node, src []byte) {
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	methodName := string(src[nameNode.StartByte():nameNode.EndByte()])

	m := Method{Name: methodName}

	// Parameters
	paramsNode := n.ChildByFieldName("parameters")
	if paramsNode != nil {
		for i := range paramsNode.ChildCount() {
			param := paramsNode.Child(i)
			if param.Kind() == "required_parameter" || param.Kind() == "optional_parameter" {
				var pName string
				if pn := param.ChildByFieldName("pattern"); pn != nil {
					pName = string(src[pn.StartByte():pn.EndByte()])
				}
				var pType TSType
				if tn := param.ChildByFieldName("type"); tn != nil {
					// type annotation node wraps the actual type
					for j := range tn.ChildCount() {
						tc := tn.Child(j)
						if tc.IsNamed() {
							pType = s.resolveType(tc, src)
							break
						}
					}
				}
				m.Params = append(m.Params, Param{Name: pName, TSType: pType})
			}
		}
	}

	// Return type
	if retNode := n.ChildByFieldName("return_type"); retNode != nil {
		for i := range retNode.ChildCount() {
			tc := retNode.Child(i)
			if tc.IsNamed() {
				m.ReturnType = s.resolveType(tc, src)
				break
			}
		}
	}

	s.Methods = append(s.Methods, m)
}

// resolveType maps a tree-sitter type node to a TSType.
func (s *Schema) resolveType(n *sitter.Node, src []byte) TSType {
	text := strings.TrimSpace(string(src[n.StartByte():n.EndByte()]))
	kind := n.Kind()

	switch kind {
	case "predefined_type":
		switch text {
		case "number":
			return TSType{Raw: text, Kind: KindNumber}
		case "boolean":
			return TSType{Raw: text, Kind: KindBoolean}
		case "void":
			return TSType{Raw: text, Kind: KindVoid}
		case "bigint":
			return TSType{Raw: text, Kind: KindBigInt}
		}

	case "type_identifier":
		if s.enumSet[text] {
			return TSType{Raw: text, Kind: KindEnum}
		}
		if alias, ok := s.typeMap[text]; ok {
			_ = alias
			return TSType{Raw: text, Kind: KindTupleAlias}
		}
		// Unknown named type
		return TSType{Raw: text, Kind: KindTupleAlias}

	case "tuple_type":
		t := TSType{Raw: text, Kind: KindTupleAlias}
		for i := range n.ChildCount() {
			el := n.Child(i)
			if el.IsNamed() {
				t.Elements = append(t.Elements, s.resolveType(el, src))
			}
		}
		return t
	}

	return TSType{Raw: text, Kind: KindUnknown}
}

// -----------------------------------------------------------------------------
// Type helpers used by the template
// -----------------------------------------------------------------------------

// GoType maps a TSType to its Go declaration type.
func (s *Schema) GoType(t TSType) string {
	switch t.Kind {
	case KindVoid:
		return ""
	case KindNumber:
		return "float64"
	case KindBigInt:
		return "uint64"
	case KindBoolean:
		return "bool"
	case KindEnum:
		return t.Raw
	case KindTupleAlias:
		if alias, ok := s.typeMap[t.Raw]; ok {
			if alias.IsHandle {
				return t.Raw
			}
		}
		return t.Raw
	}
	return "interface{}" // fallback
}

// WasmParamType returns the WASM value type for a parameter.
// Tuple types are passed as i32 pointers; scalars are their native type.
func (s *Schema) WasmParamType(t TSType) string {
	switch t.Kind {
	case KindNumber:
		return "f64"
	case KindBoolean, KindEnum:
		return "i32"
	case KindBigInt:
		return "i64"
	default:
		return "i32" // tuple/handle/unknown → pointer
	}
}

// IsTuple returns true if the type represents a C++ sret (composite return).
func (s *Schema) IsTuple(t TSType) bool {
	switch t.Kind {
	case KindTupleAlias, KindUnknown:
		return true
	}
	return false
}

// AllocSizeOf returns the estimated WASM memory footprint of t in bytes.
// This is used to allocate sret buffers and parameter buffers.
func (s *Schema) AllocSizeOf(t TSType) uint32 {
	switch t.Kind {
	case KindNumber:
		return 8 // f64
	case KindBigInt:
		return 8 // i64
	case KindBoolean, KindEnum:
		return 4 // i32
	case KindTupleAlias:
		if alias, ok := s.typeMap[t.Raw]; ok {
			return s.tupleSize(alias.Tuple)
		}
		if len(t.Elements) > 0 {
			return s.tupleSize(t.Elements)
		}
		return 16 // fallback: safe for [Result, handle]
	}
	return 8
}

func (s *Schema) tupleSize(elements []TSType) uint32 {
	if len(elements) == 0 {
		return 8
	}
	var total uint32
	for _, el := range elements {
		elemSize := s.AllocSizeOf(el)
		total = alignUpU32(total, elemSize)
		total += elemSize
	}
	// round up to alignment of largest element
	return total
}

func alignUpU32(n, align uint32) uint32 {
	if align == 0 {
		return n
	}
	return (n + align - 1) &^ (align - 1)
}

// -----------------------------------------------------------------------------
// Generator
// -----------------------------------------------------------------------------

// GenerateOptions controls what the generator emits.
type GenerateOptions struct {
	PackageName string // Go package name for the generated file
	OutputDir   string // directory to write generated files into
	TypesFile   string // filename for the types file
	BindingFile string // filename for the bindings file
}

// DefaultOptions returns sensible defaults for the generated output.
func DefaultOptions() GenerateOptions {
	return GenerateOptions{
		PackageName: "generated",
		OutputDir:   ".",
		TypesFile:   "types_gen.go",
		BindingFile: "bindings_gen.go",
	}
}

// Generate runs the generator with the given schema and options.
func Generate(schema *Schema, opts GenerateOptions) error {
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return fmt.Errorf("converter: mkdir %s: %w", opts.OutputDir, err)
	}

	if err := generateTypes(schema, opts); err != nil {
		return err
	}
	if err := generateBindings(schema, opts); err != nil {
		return err
	}
	return nil
}

// -----------------------------------------------------------------------------
// Types template
// -----------------------------------------------------------------------------

var typesTemplate = `// Code generated by havok-go converter. DO NOT EDIT.

package {{.PackageName}}

// =============================================================================
// Enums
// =============================================================================
{{range $enum := .Enums}}
type {{$enum.Name}} int32

const ({{range $i, $v := $enum.Values}}
	{{$enum.Name}}_{{$v}} {{if eq $i 0}}{{$enum.Name}} {{end}}= {{$i}}{{end}}
)
{{end}}

// =============================================================================
// Type aliases (HP handles and composite types)
// =============================================================================
{{range .Types}}
// {{.Name}} mirrors HavokPhysics.d.ts  "type {{.Name}} = [...]"
{{- if .IsHandle}}
type {{.Name}} [1]uint64{{else}}
// Tuple elements: {{range .Tuple}}{{.Raw}}, {{end}}
type {{.Name}} struct {
	// TODO: fill in from AllocSizeOf analysis; placeholder i32 fields:
{{- range $i, $_ := .Tuple}}
	Field{{$i}} uint32{{end}}
}{{end}}
{{end}}
`

func generateTypes(schema *Schema, opts GenerateOptions) error {
	type templateData struct {
		PackageName string
		Enums       []EnumDecl
		Types       []TypeAlias
	}

	// Only emit types that are not already defined in the havok package:
	// (the generator is standalone; it re-emits everything it finds)
	data := templateData{
		PackageName: opts.PackageName,
		Enums:       schema.Enums,
		Types:       schema.Types,
	}

	tmpl, err := template.New("types").Parse(typesTemplate)
	if err != nil {
		return fmt.Errorf("converter: types template: %w", err)
	}

	f, err := os.Create(filepath.Join(opts.OutputDir, opts.TypesFile))
	if err != nil {
		return fmt.Errorf("converter: create %s: %w", opts.TypesFile, err)
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}

// -----------------------------------------------------------------------------
// Bindings template
// -----------------------------------------------------------------------------

var bindingsTemplate = `// Code generated by havok-go converter. DO NOT EDIT.
//
// Each function wraps a Havok WASM export following the emscripten sret
// calling convention:
//   • Tuple return types  → sret pointer as first arg; WASM returns void.
//   • Scalar return types → WASM returns i32 directly.
//   • Tuple parameters    → passed as i32 pointer to WASM memory.
//   • number parameters   → passed as f64.
//   • boolean parameters  → passed as i32.

package {{.PackageName}}

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/tetratelabs/wazero/api"
)

// Binding wraps a wazero api.Module and provides typed HP_* methods.
type Binding struct {
	mod    api.Module
	malloc api.Function
	free   api.Function
}

// NewBinding creates a Binding from an already-instantiated wazero Module.
func NewBinding(mod api.Module) (*Binding, error) {
	m := mod.ExportedFunction("malloc")
	f := mod.ExportedFunction("free")
	if m == nil || f == nil {
		return nil, fmt.Errorf("generated: module missing malloc/free")
	}
	return &Binding{mod: mod, malloc: m, free: f}, nil
}

func (b *Binding) alloc(ctx context.Context, size uint32) (uint32, error) {
	res, err := b.malloc.Call(ctx, uint64(size))
	if err != nil {
		return 0, err
	}
	ptr := uint32(res[0])
	if ptr == 0 {
		return 0, fmt.Errorf("generated: malloc(%d) returned NULL", size)
	}
	buf := make([]byte, size)
	if !b.mod.Memory().Write(ptr, buf) {
		b.free.Call(ctx, uint64(ptr))
		return 0, fmt.Errorf("generated: zero fill failed at %d", ptr)
	}
	return ptr, nil
}

func (b *Binding) freePtr(ctx context.Context, ptr uint32) {
	if ptr != 0 {
		b.free.Call(ctx, uint64(ptr))
	}
}

func readI32(mem api.Memory, offset uint32) (int32, error) {
	bs, ok := mem.Read(offset, 4)
	if !ok {
		return 0, fmt.Errorf("generated: readI32 OOB at %d", offset)
	}
	return int32(binary.LittleEndian.Uint32(bs)), nil
}

func readU64(mem api.Memory, offset uint32) (uint64, error) {
	bs, ok := mem.Read(offset, 8)
	if !ok {
		return 0, fmt.Errorf("generated: readU64 OOB at %d", offset)
	}
	return binary.LittleEndian.Uint64(bs), nil
}

func writeF64(mem api.Memory, offset uint32, v float64) bool {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, math.Float64bits(v))
	return mem.Write(offset, bs)
}

func writeU64(mem api.Memory, offset uint32, v uint64) bool {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, v)
	return mem.Write(offset, bs)
}

// =============================================================================
// Generated HP_* bindings
// =============================================================================
{{range .Methods}}
// {{.Name}} wraps HP_* WASM export.
// TypeScript: {{.Name}}({{range $i, $p := .Params}}{{if $i}}, {{end}}{{$p.Name}}: {{$p.TSType.Raw}}{{end}}): {{.ReturnType.Raw}}
func (b *Binding) {{.Name}}(ctx context.Context{{range .Params}}, {{.Name}} {{call $.GoType .TSType}}{{end}}) {{call $.FuncReturn .ReturnType}} {
	{{- call $.FuncBody . }}
}
{{end}}
`

// templateFuncs provides custom template functions.
type templateFuncs struct {
	schema *Schema
}

func (tf templateFuncs) GoType(t TSType) string    { return tf.schema.GoType(t) }
func (tf templateFuncs) IsTuple(t TSType) bool     { return tf.schema.IsTuple(t) }
func (tf templateFuncs) AllocSize(t TSType) uint32 { return tf.schema.AllocSizeOf(t) }
func (tf templateFuncs) WasmParam(t TSType) string { return tf.schema.WasmParamType(t) }

// FuncReturn builds the Go return type list as a string "(T1, T2, error)".
// For tuple return types (e.g. [Result, HP_BodyId]), it generates multiple return values.
func (tf templateFuncs) FuncReturn(ret TSType) string {
	if ret.Kind == KindVoid || ret.Raw == "" {
		return "error"
	}
	if len(ret.Elements) > 0 {
		// Tuple return: emit each element as a separate return value
		var parts []string
		for _, el := range ret.Elements {
			gt := tf.schema.GoType(el)
			if gt == "" {
				gt = "interface{}"
			}
			parts = append(parts, gt)
		}
		parts = append(parts, "error")
		return "(" + strings.Join(parts, ", ") + ")"
	}
	gt := tf.schema.GoType(ret)
	if gt == "" {
		return "error"
	}
	return fmt.Sprintf("(%s, error)", gt)
}

// FuncBody generates the function body for a given method.
func (tf templateFuncs) FuncBody(m Method) string {
	var b strings.Builder
	isTupleReturn := tf.schema.IsTuple(m.ReturnType)
	retGoType := tf.schema.GoType(m.ReturnType)

	// Determine the zero return expression for the return type
	zeroRetExpr := tf.zeroExpr(m.ReturnType)

	retSize := tf.schema.AllocSizeOf(m.ReturnType)
	var wasmArgs []string

	if isTupleReturn {
		b.WriteString(fmt.Sprintf("\n\tsretPtr, err := b.alloc(ctx, %d)\n\tif err != nil { return %s, err }\n\tdefer b.freePtr(ctx, sretPtr)\n",
			retSize, zeroRetExpr))
		wasmArgs = append(wasmArgs, "uint64(sretPtr)")
	}

	for i, p := range m.Params {
		paramVarName := p.Name
		switch p.TSType.Kind {
		case KindNumber:
			wasmArgs = append(wasmArgs, fmt.Sprintf("api.EncodeF64(%s)", paramVarName))
		case KindBoolean:
			boolVar := fmt.Sprintf("_%sBool", paramVarName)
			b.WriteString(fmt.Sprintf("\n\tvar %s uint64\n\tif %s { %s = 1 }\n", boolVar, paramVarName, boolVar))
			wasmArgs = append(wasmArgs, boolVar)
		case KindEnum:
			wasmArgs = append(wasmArgs, fmt.Sprintf("uint64(%s)", paramVarName))
		default:
			// Tuple / handle → write to WASM memory, pass pointer
			ptrName := fmt.Sprintf("_param%dPtr", i)
			pSize := tf.schema.AllocSizeOf(p.TSType)
			b.WriteString(fmt.Sprintf("\n\t%s, err := b.alloc(ctx, %d)\n\tif err != nil { return %s, err }\n\tdefer b.freePtr(ctx, %s)\n",
				ptrName, pSize, zeroRetExpr, ptrName))
			b.WriteString(fmt.Sprintf("\t// TODO: marshal %s into %s (type: %s)\n", paramVarName, ptrName, p.TSType.Raw))
			wasmArgs = append(wasmArgs, fmt.Sprintf("uint64(%s)", ptrName))
		}
	}

	// Call the WASM function
	b.WriteString(fmt.Sprintf("\n\tfn := b.mod.ExportedFunction(%q)\n\tif fn == nil { return %s, fmt.Errorf(%q) }\n",
		m.Name, zeroRetExpr, "generated: "+m.Name+" not exported"))

	if isTupleReturn || m.ReturnType.Kind == KindVoid || retGoType == "" {
		b.WriteString(fmt.Sprintf("\n\tif _, callErr := fn.Call(ctx, %s); callErr != nil { return %s, callErr }\n",
			strings.Join(wasmArgs, ", "), zeroRetExpr))
	} else {
		b.WriteString(fmt.Sprintf("\n\t_res, callErr := fn.Call(ctx, %s)\n\tif callErr != nil { return %s, callErr }\n",
			strings.Join(wasmArgs, ", "), zeroRetExpr))
	}

	// Return result
	if isTupleReturn {
		b.WriteString(fmt.Sprintf("\n\t// TODO: deserialize result from sretPtr (size=%d bytes)\n", retSize))
		b.WriteString(fmt.Sprintf("\treturn %s, nil // placeholder – implement deserialize\n", zeroRetExpr))
	} else if m.ReturnType.Kind == KindVoid || retGoType == "" {
		b.WriteString("\n\treturn nil\n")
	} else {
		// scalar i32 → direct WASM return
		b.WriteString(fmt.Sprintf("\n\treturn %s(int32(_res[0])), nil\n", retGoType))
	}

	return b.String()
}

// zeroExpr returns the Go source for the zero value of t.
// For tuple types it returns a comma-separated list (one zero per element).
func (tf templateFuncs) zeroExpr(t TSType) string {
	if len(t.Elements) > 0 {
		var parts []string
		for _, el := range t.Elements {
			parts = append(parts, tf.scalarZero(el))
		}
		return strings.Join(parts, ", ")
	}
	return tf.scalarZero(t)
}

// scalarZero returns the zero expression for a single (non-tuple) type.
func (tf templateFuncs) scalarZero(t TSType) string {
	switch t.Kind {
	case KindVoid:
		return ""
	case KindNumber:
		return "0"
	case KindBigInt:
		return "0"
	case KindBoolean:
		return "false"
	case KindEnum:
		return "0"
	case KindTupleAlias:
		gt := tf.schema.GoType(t)
		if gt == "" || gt == "interface{}" {
			return "nil"
		}
		return gt + "{}"
	}
	return "nil"
}

func generateBindings(schema *Schema, opts GenerateOptions) error {
	// Sort methods by name for reproducible output
	methods := make([]Method, len(schema.Methods))
	copy(methods, schema.Methods)
	slices.SortFunc(methods, func(a, b Method) int {
		return cmp.Compare(a.Name, b.Name)
	})

	tf := templateFuncs{schema: schema}

	funcMap := template.FuncMap{
		"GoType":     tf.GoType,
		"FuncReturn": tf.FuncReturn,
		"FuncBody":   tf.FuncBody,
	}

	type templateData struct {
		PackageName string
		Methods     []Method
		GoType      func(TSType) string
		FuncReturn  func(TSType) string
		FuncBody    func(Method) string
	}

	data := templateData{
		PackageName: opts.PackageName,
		Methods:     methods,
		GoType:      tf.GoType,
		FuncReturn:  tf.FuncReturn,
		FuncBody:    tf.FuncBody,
	}

	tmpl, err := template.New("bindings").Funcs(funcMap).Parse(bindingsTemplate)
	if err != nil {
		return fmt.Errorf("converter: bindings template: %w", err)
	}

	f, err := os.Create(filepath.Join(opts.OutputDir, opts.BindingFile))
	if err != nil {
		return fmt.Errorf("converter: create %s: %w", opts.BindingFile, err)
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}
