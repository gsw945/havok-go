// Package havok provides Go bindings for the Havok physics engine WASM module.
//
// It wraps the HavokPhysics.wasm binary (from @babylonjs/havok) using wazero,
// a zero-dependency pure-Go WebAssembly runtime.
//
// Usage:
//
// hp, err := havok.New(ctx, "HavokPhysics.wasm")
// if err != nil { log.Fatal(err) }
// defer hp.Close()
//
// res, stats, err := hp.HP_GetStatistics(ctx)
package havok

import (
	"context"
	"fmt"
	"os"

	"github.com/gsw945/havok-go/havok/binding"
	"github.com/gsw945/havok-go/havok/generated"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// HavokPhysics is the top-level handle for the Havok physics engine.
// All HP_* methods are promoted from the embedded *generated.HP.
type HavokPhysics struct {
	*generated.HP
	ctx     context.Context
	runtime wazero.Runtime
}

// New loads the HavokPhysics WASM binary from wasmPath and initialises the engine.
// The returned *HavokPhysics must be closed with Close() to release resources.
func New(ctx context.Context, wasmPath string) (*HavokPhysics, error) {
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("havok: reading wasm file %q: %w", wasmPath, err)
	}
	return NewFromBytes(ctx, wasmBytes)
}

// NewFromBytes initialises the engine from an in-memory WASM binary.
func NewFromBytes(ctx context.Context, wasmBytes []byte) (*HavokPhysics, error) {
	r := wazero.NewRuntime(ctx)

	// Provide WASI (needed for fd_write / stdout in emscripten).
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("havok: wasi instantiation: %w", err)
	}

	// Provide emscripten env stubs.
	if err := registerEnvModule(ctx, r); err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("havok: env module: %w", err)
	}

	// Instantiate the WASM module without auto-start (emscripten uses __wasm_call_ctors).
	cfg := wazero.NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStartFunctions() // do not call _start / _initialize automatically
	mod, err := r.InstantiateWithConfig(ctx, wasmBytes, cfg)
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("havok: wasm instantiation: %w", err)
	}

	// Run emscripten's global constructors (embind registration).
	if ctors := mod.ExportedFunction("__wasm_call_ctors"); ctors != nil {
		if _, err := ctors.Call(ctx); err != nil {
			mod.Close(ctx)
			r.Close(ctx)
			return nil, fmt.Errorf("havok: __wasm_call_ctors: %w", err)
		}
	}

	// Call main(0, 0) to fully initialize internal state (allocators, vtables, etc.).
	if mainFn := mod.ExportedFunction("main"); mainFn != nil {
		if _, err := mainFn.Call(ctx, 0, 0); err != nil {
			mod.Close(ctx)
			r.Close(ctx)
			return nil, fmt.Errorf("havok: main: %w", err)
		}
	}

	b, err := binding.NewBinding(mod)
	if err != nil {
		mod.Close(ctx)
		r.Close(ctx)
		return nil, err
	}

	return &HavokPhysics{HP: generated.NewHP(b), ctx: ctx, runtime: r}, nil
}

// Close releases all resources associated with the engine.
func (h *HavokPhysics) Close() error {
	if err := h.Module().Close(h.ctx); err != nil {
		return err
	}
	return h.runtime.Close(h.ctx)
}
