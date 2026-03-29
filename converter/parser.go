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
	"fmt"
	"os"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

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
// Type helpers used by the generator
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
