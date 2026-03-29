// Package binding provides the Binding struct and supporting infrastructure
// for the Havok physics WASM module.
//
// Binding wraps a wazero api.Module; generated/bindings_gen.go wraps Binding
// via the HP type and adds all HP_* methods. helpers.go provides exported
// memory I/O utilities that the generated code calls via binding.WriteVector3
// etc.
//
// This file is NOT generated — it contains hand-maintained infrastructure.
package binding

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero/api"
)

// Binding wraps a wazero api.Module and exposes the low-level infrastructure
// needed by the generated HP_* methods.
type Binding struct {
	mod    api.Module
	malloc api.Function
	free   api.Function
}

// NewBinding creates a Binding from an already-instantiated wazero Module.
// The module must export "malloc" and "free".
func NewBinding(mod api.Module) (*Binding, error) {
	m := mod.ExportedFunction("malloc")
	f := mod.ExportedFunction("free")
	if m == nil || f == nil {
		return nil, fmt.Errorf("binding: wasm module missing malloc/free exports")
	}
	return &Binding{mod: mod, malloc: m, free: f}, nil
}

// Module returns the underlying wazero api.Module for advanced use.
func (b *Binding) Module() api.Module { return b.mod }

// Mem returns the WASM linear memory for direct read/write.
func (b *Binding) Mem() api.Memory { return b.mod.Memory() }

// Alloc allocates `size` bytes in WASM memory, zero-initialises them, and
// returns the linear-memory pointer.
func (b *Binding) Alloc(ctx context.Context, size uint32) (uint32, error) {
	res, err := b.malloc.Call(ctx, uint64(size))
	if err != nil {
		return 0, fmt.Errorf("binding: malloc(%d): %w", size, err)
	}
	ptr := uint32(res[0])
	if ptr == 0 {
		return 0, fmt.Errorf("binding: malloc returned NULL for size %d", size)
	}
	// Zero the allocated region so result structs start clean.
	buf := make([]byte, size)
	if !b.mod.Memory().Write(ptr, buf) {
		b.free.Call(ctx, uint64(ptr)) //nolint:errcheck
		return 0, fmt.Errorf("binding: zero write failed at ptr=%d size=%d", ptr, size)
	}
	return ptr, nil
}

// FreePtr frees a pointer previously obtained from Alloc.
func (b *Binding) FreePtr(ctx context.Context, ptr uint32) {
	if ptr != 0 {
		b.free.Call(ctx, uint64(ptr)) //nolint:errcheck
	}
}

// CallResultI32 calls a WASM function that returns i32 (Result) directly.
func (b *Binding) CallResultI32(ctx context.Context, fnName string, args ...uint64) (Result, error) {
	fn := b.mod.ExportedFunction(fnName)
	if fn == nil {
		return 0, fmt.Errorf("binding: %s not exported", fnName)
	}
	res, err := fn.Call(ctx, args...)
	if err != nil {
		return 0, fmt.Errorf("binding: %s: %w", fnName, err)
	}
	if len(res) == 0 {
		return Result_OK, nil
	}
	return Result(int32(res[0])), nil
}
