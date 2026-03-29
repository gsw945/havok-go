// Package wasm embeds the HavokPhysics.wasm binary for use with wazero.
//
// The wasm file is populated by running:
//
//	havok-go convert --input <path/to/HavokPhysics.d.ts>
//
// which automatically copies HavokPhysics.wasm from the BabylonJS-havok package
// into this directory.  If the wasm file cannot be located automatically, the
// convert command will print the expected destination path so you can copy it
// manually.
package wasm

import _ "embed"

// WasmBytes contains the HavokPhysics.wasm binary, embedded at compile time.
// WasmBytes will be a stub placeholder (8 bytes) until you run "havok-go convert"
// or copy a real HavokPhysics.wasm here.
//
//go:embed HavokPhysics.wasm
var WasmBytes []byte

// IsReal returns true when WasmBytes is a proper WebAssembly module
// (more than the 8-byte magic+version placeholder).
func IsReal() bool {
	return len(WasmBytes) > 8
}
