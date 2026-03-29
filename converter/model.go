package converter

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
	typeMap map[string]TypeAlias // name → alias (value copy, not pointer—prevents stale pointers after slice realloc)
	enumSet map[string]bool
	Methods []Method
}
