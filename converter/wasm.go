package converter

import (
	"fmt"
	"os"
)

// ValueType is a WebAssembly value type as encoded in the binary format.
type ValueType byte

const (
	ValI32 ValueType = 0x7f
	ValI64 ValueType = 0x7e
	ValF32 ValueType = 0x7d
	ValF64 ValueType = 0x7c
)

func (v ValueType) String() string {
	switch v {
	case ValI32:
		return "i32"
	case ValI64:
		return "i64"
	case ValF32:
		return "f32"
	case ValF64:
		return "f64"
	default:
		return fmt.Sprintf("0x%02x", byte(v))
	}
}

// FuncSig is the raw WASM signature of an exported function.
type FuncSig struct {
	Params  []ValueType
	Results []ValueType
}

// ReturnsI32 reports whether the function returns a single i32 (the Havok Result).
func (s FuncSig) ReturnsI32() bool {
	return len(s.Results) == 1 && s.Results[0] == ValI32
}

// WasmSignatures maps an export name to its signature.
type WasmSignatures map[string]FuncSig

type wasmModule struct {
	types   [][]ValueType // type index → params (results tracked separately)
	results [][]ValueType
	imports []uint32 // type index per imported function
	funcs   []uint32 // type index per defined function
}

// ParseWasmSignatures reads a .wasm binary and returns the signature of every
// exported function. This is the authoritative source for the calling
// convention, because the HavokPhysics.d.ts declarations omit the low-level
// pointer/scalar details that emscripten's embind ABI requires.
func ParseWasmSignatures(path string) (WasmSignatures, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("converter: reading wasm %s: %w", path, err)
	}
	if len(raw) < 8 || string(raw[0:4]) != "\x00asm" {
		return nil, fmt.Errorf("converter: %s is not a wasm module", path)
	}

	m := &wasmModule{}
	type exportEntry struct {
		name string
		idx  uint32
		kind byte
	}
	var exports []exportEntry

	i := 8
	for i < len(raw) {
		sectionID := raw[i]
		i++
		size, n, err := readULEB(raw, i)
		if err != nil {
			return nil, err
		}
		i = n
		end := i + int(size)
		if end > len(raw) {
			return nil, fmt.Errorf("converter: truncated wasm section %d", sectionID)
		}

		switch sectionID {
		case 1: // type
			if err := m.parseTypes(raw, i, end); err != nil {
				return nil, err
			}
		case 2: // import
			if err := m.parseImports(raw, i, end); err != nil {
				return nil, err
			}
		case 3: // function
			if err := m.parseFunctions(raw, i, end); err != nil {
				return nil, err
			}
		case 7: // export
			j := i
			count, j, err := readULEB(raw, j)
			if err != nil {
				return nil, err
			}
			for k := uint32(0); k < count; k++ {
				nlen, j2, err := readULEB(raw, j)
				if err != nil {
					return nil, err
				}
				j = j2
				if j+int(nlen) > len(raw) {
					return nil, fmt.Errorf("converter: truncated export name")
				}
				name := string(raw[j : j+int(nlen)])
				j += int(nlen)
				kind := raw[j]
				j++
				idx, j3, err := readULEB(raw, j)
				if err != nil {
					return nil, err
				}
				j = j3
				exports = append(exports, exportEntry{name: name, idx: idx, kind: kind})
			}
		}
		i = end
	}

	out := make(WasmSignatures, len(exports))
	nImports := len(m.imports)
	for _, e := range exports {
		if e.kind != 0 { // function exports only
			continue
		}
		var ti uint32
		if int(e.idx) < nImports {
			ti = m.imports[e.idx]
		} else {
			fi := int(e.idx) - nImports
			if fi >= len(m.funcs) {
				continue
			}
			ti = m.funcs[fi]
		}
		if int(ti) >= len(m.types) {
			continue
		}
		out[e.name] = FuncSig{Params: m.types[ti], Results: m.results[ti]}
	}
	return out, nil
}

func (m *wasmModule) parseTypes(raw []byte, i, end int) error {
	count, j, err := readULEB(raw, i)
	if err != nil {
		return err
	}
	for k := uint32(0); k < count; k++ {
		if j >= end || raw[j] != 0x60 {
			return fmt.Errorf("converter: malformed type section")
		}
		j++
		np, j2, err := readULEB(raw, j)
		if err != nil {
			return err
		}
		j = j2
		params := make([]ValueType, np)
		for p := uint32(0); p < np; p++ {
			params[p] = ValueType(raw[j])
			j++
		}
		nr, j3, err := readULEB(raw, j)
		if err != nil {
			return err
		}
		j = j3
		results := make([]ValueType, nr)
		for r := uint32(0); r < nr; r++ {
			results[r] = ValueType(raw[j])
			j++
		}
		m.types = append(m.types, params)
		m.results = append(m.results, results)
	}
	return nil
}

func (m *wasmModule) parseImports(raw []byte, i, end int) error {
	count, j, err := readULEB(raw, i)
	if err != nil {
		return err
	}
	for k := uint32(0); k < count; k++ {
		mlen, j2, err := readULEB(raw, j)
		if err != nil {
			return err
		}
		j = j2 + int(mlen)
		nlen, j3, err := readULEB(raw, j)
		if err != nil {
			return err
		}
		j = j3 + int(nlen)
		if j >= end {
			return fmt.Errorf("converter: malformed import section")
		}
		kind := raw[j]
		j++
		switch kind {
		case 0: // func
			ti, j4, err := readULEB(raw, j)
			if err != nil {
				return err
			}
			j = j4
			m.imports = append(m.imports, ti)
		case 1: // table
			j++ // elem type
			lim, j4, err := readULEB(raw, j)
			if err != nil {
				return err
			}
			j = j4
			if lim&1 != 0 {
				_, j5, err := readULEB(raw, j)
				if err != nil {
					return err
				}
				j = j5
			}
		case 2: // memory
			lim, j4, err := readULEB(raw, j)
			if err != nil {
				return err
			}
			j = j4
			if lim&1 != 0 {
				_, j5, err := readULEB(raw, j)
				if err != nil {
					return err
				}
				j = j5
			}
		case 3: // global
			j += 2
		default:
			return fmt.Errorf("converter: unknown import kind %d", kind)
		}
	}
	return nil
}

func (m *wasmModule) parseFunctions(raw []byte, i, end int) error {
	count, j, err := readULEB(raw, i)
	if err != nil {
		return err
	}
	for k := uint32(0); k < count; k++ {
		ti, j2, err := readULEB(raw, j)
		if err != nil {
			return err
		}
		j = j2
		m.funcs = append(m.funcs, ti)
	}
	return nil
}

// readULEB decodes an unsigned LEB128 integer starting at i.
func readULEB(b []byte, i int) (uint32, int, error) {
	var result uint32
	var shift uint
	for {
		if i >= len(b) {
			return 0, i, fmt.Errorf("converter: unexpected end of wasm while reading LEB128")
		}
		x := b[i]
		i++
		result |= uint32(x&0x7f) << shift
		if x&0x80 == 0 {
			break
		}
		shift += 7
		if shift > 35 {
			return 0, i, fmt.Errorf("converter: LEB128 too long")
		}
	}
	return result, i, nil
}
