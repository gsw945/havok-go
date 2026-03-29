// Package havok provides Go bindings for the Havok physics engine WASM module.
//
// It wraps the HavokPhysics.wasm binary (from @babylonjs/havok) using wazero,
// a zero-dependency pure-Go WebAssembly runtime.
//
// Usage:
//
//	hp, err := havok.New(ctx, "HavokPhysics.wasm")
//	if err != nil { log.Fatal(err) }
//	defer hp.Close()
//
//	res, stats, err := hp.HP_GetStatistics(ctx)
package havok

import (
	"context"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// HavokPhysics is the top-level handle for the Havok physics engine.
type HavokPhysics struct {
	ctx     context.Context
	runtime wazero.Runtime
	mod     api.Module
	malloc  api.Function
	free    api.Function
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

	hp := &HavokPhysics{
		ctx:     ctx,
		runtime: r,
		mod:     mod,
		malloc:  mod.ExportedFunction("malloc"),
		free:    mod.ExportedFunction("free"),
	}

	if hp.malloc == nil || hp.free == nil {
		mod.Close(ctx)
		r.Close(ctx)
		return nil, fmt.Errorf("havok: wasm module missing malloc/free exports")
	}

	return hp, nil
}

// Close releases all resources associated with the engine.
func (h *HavokPhysics) Close() error {
	if err := h.mod.Close(h.ctx); err != nil {
		return err
	}
	return h.runtime.Close(h.ctx)
}

// Module returns the underlying wazero api.Module for advanced use.
func (h *HavokPhysics) Module() api.Module { return h.mod }

// alloc allocates `size` bytes in WASM memory and returns the pointer.
func (h *HavokPhysics) alloc(ctx context.Context, size uint32) (uint32, error) {
	res, err := h.malloc.Call(ctx, uint64(size))
	if err != nil {
		return 0, fmt.Errorf("havok: malloc(%d): %w", size, err)
	}
	ptr := uint32(res[0])
	if ptr == 0 {
		return 0, fmt.Errorf("havok: malloc returned NULL for size %d", size)
	}
	// Zero the allocated region so result structs start clean.
	if zeroErr := h.zero(ctx, ptr, size); zeroErr != nil {
		h.free.Call(ctx, uint64(ptr)) //nolint:errcheck
		return 0, zeroErr
	}
	return ptr, nil
}

// freePtr frees a pointer previously obtained from alloc.
func (h *HavokPhysics) freePtr(ctx context.Context, ptr uint32) {
	if ptr != 0 {
		h.free.Call(ctx, uint64(ptr)) //nolint:errcheck
	}
}

// zero zeros `size` bytes in WASM memory starting at ptr.
func (h *HavokPhysics) zero(ctx context.Context, ptr, size uint32) error {
	mem := h.mod.Memory()
	buf := make([]byte, size)
	if !mem.Write(ptr, buf) {
		return fmt.Errorf("havok: zero: write failed at ptr=%d size=%d", ptr, size)
	}
	return nil
}

// mem returns the WASM linear memory for direct read/write.
func (h *HavokPhysics) mem() api.Memory { return h.mod.Memory() }

// =============================================================================
// HP_* function bindings
//
// Calling convention (emscripten / embind):
//   - Functions returning a tuple type [T1, T2, ...] receive an *sret* pointer
//     as their FIRST argument; the result struct is written there; WASM returns void.
//   - Functions returning a scalar (Result/i32) return directly as WASM i32.
//   - Tuple-typed arguments are passed as i32 pointers to WASM memory.
//   - Scalar number arguments are passed as f64.
//   - Boolean arguments are passed as i32.
// =============================================================================

// HP_GetStatistics returns allocation counts for native Havok objects.
//
// TypeScript: HP_GetStatistics(): [Result, ObjectStatistics]
// WASM: HP_GetStatistics(sretPtr: i32) -> void
// Memory layout of sret (28 bytes, all i32):
//
//	[0] Result, [4] NumBodies, [8] NumShapes, [12] NumConstraints,
//	[16] NumDebugGeometries, [20] NumWorlds, [24] NumQueryCollectors
func (h *HavokPhysics) HP_GetStatistics(ctx context.Context) (Result, ObjectStatistics, error) {
	const sretSize = 28
	ptr, err := h.alloc(ctx, sretSize)
	if err != nil {
		return 0, ObjectStatistics{}, err
	}
	defer h.freePtr(ctx, ptr)

	fn := h.mod.ExportedFunction("HP_GetStatistics")
	if fn == nil {
		return 0, ObjectStatistics{}, fmt.Errorf("havok: HP_GetStatistics not exported")
	}
	if _, err := fn.Call(ctx, uint64(ptr)); err != nil {
		return 0, ObjectStatistics{}, fmt.Errorf("havok: HP_GetStatistics: %w", err)
	}

	mem := h.mem()
	result, err := readI32(mem, ptr)
	if err != nil {
		return 0, ObjectStatistics{}, err
	}
	var stats ObjectStatistics
	for i, fp := range []*int32{
		&stats.NumBodies, &stats.NumShapes, &stats.NumConstraints,
		&stats.NumDebugGeometries, &stats.NumWorlds, &stats.NumQueryCollectors,
	} {
		v, err := readI32(mem, uint32(int(ptr)+4+i*4))
		if err != nil {
			return 0, ObjectStatistics{}, err
		}
		*fp = v
	}

	return Result(result), stats, nil
}

// HP_World_Create creates a new physics world.
//
// TypeScript: HP_World_Create(): [Result, HP_WorldId]
// WASM: HP_World_Create(sretPtr i32) -> (Result i32)
// sretPtr[0..7] = HP_WorldId (i64)
func (h *HavokPhysics) HP_World_Create(ctx context.Context) (Result, HP_WorldId, error) {
	res, id, err := h.createHandle(ctx, "HP_World_Create")
	return res, HP_WorldId{id}, err
}

// HP_World_Release releases a physics world.
//
// TypeScript: HP_World_Release(worldId: HP_WorldId): Result
// WASM: HP_World_Release(worldId i64) -> (Result i32)
func (h *HavokPhysics) HP_World_Release(ctx context.Context, worldId HP_WorldId) (Result, error) {
	return h.releaseHandle(ctx, "HP_World_Release", worldId[0])
}

// HP_World_SetGravity sets the gravity vector on a world.
//
// TypeScript: HP_World_SetGravity(worldId, gravity): Result
// WASM: HP_World_SetGravity(worldId i64, gravityPtr i32) -> (Result i32)
// gravity is 3×f32 = 12 bytes
func (h *HavokPhysics) HP_World_SetGravity(ctx context.Context, worldId HP_WorldId, gravity Vector3) (Result, error) {
	gravPtr, err := h.alloc(ctx, 12) // Vector3 = 3*f32 = 12 bytes
	if err != nil {
		return 0, err
	}
	defer h.freePtr(ctx, gravPtr)

	if err := writeVector3(h.mem(), gravPtr, gravity); err != nil {
		return 0, err
	}

	return h.callResultI32(ctx, "HP_World_SetGravity", worldId[0], uint64(gravPtr))
}

// HP_World_Step advances the simulation by dt seconds.
//
// TypeScript: HP_World_Step(worldId, dt): Result
// WASM: HP_World_Step(worldId i64, dt f32) -> (Result i32)
func (h *HavokPhysics) HP_World_Step(ctx context.Context, worldId HP_WorldId, dt float64) (Result, error) {
	return h.callResultI32(ctx, "HP_World_Step", worldId[0], api.EncodeF32(float32(dt)))
}

// HP_Body_Create allocates a new physics body.
//
// TypeScript: HP_Body_Create(): [Result, HP_BodyId]
// WASM: HP_Body_Create(sretPtr i32) -> (Result i32)
func (h *HavokPhysics) HP_Body_Create(ctx context.Context) (Result, HP_BodyId, error) {
	res, id, err := h.createHandle(ctx, "HP_Body_Create")
	return res, HP_BodyId{id}, err
}

// HP_Body_Release releases a physics body.
//
// WASM: HP_Body_Release(bodyId i64) -> (Result i32)
func (h *HavokPhysics) HP_Body_Release(ctx context.Context, bodyId HP_BodyId) (Result, error) {
	return h.releaseHandle(ctx, "HP_Body_Release", bodyId[0])
}

// HP_Shape_CreateSphere creates a sphere collision shape.
//
// TypeScript: HP_Shape_CreateSphere(center: Vector3, radius: number): [Result, HP_ShapeId]
// WASM: HP_Shape_CreateSphere(sretPtr i32, radius f32, centerPtr i32) -> (Result i32)
// center is 3×f32 = 12 bytes
func (h *HavokPhysics) HP_Shape_CreateSphere(ctx context.Context, center Vector3, radius float64) (Result, HP_ShapeId, error) {
	sretPtr, err := h.alloc(ctx, 8) // 8 bytes for HP_ShapeId (i64)
	if err != nil {
		return 0, HP_ShapeId{}, err
	}
	defer h.freePtr(ctx, sretPtr)

	centerPtr, err := h.alloc(ctx, 12) // 3×f32 = 12 bytes
	if err != nil {
		return 0, HP_ShapeId{}, err
	}
	defer h.freePtr(ctx, centerPtr)

	if err := writeVector3(h.mem(), centerPtr, center); err != nil {
		return 0, HP_ShapeId{}, err
	}

	fn := h.mod.ExportedFunction("HP_Shape_CreateSphere")
	if fn == nil {
		return 0, HP_ShapeId{}, fmt.Errorf("havok: HP_Shape_CreateSphere not exported")
	}
	// HP_Shape_CreateSphere(sretPtr i32, radius f32, centerPtr i32) -> (Result i32)
	// Emscripten places sret LAST when there are input arguments before it.
	// Actual WASM arg order: (centerPtr i32, radius f32, sretPtr i32)
	res, err := fn.Call(ctx, uint64(centerPtr), api.EncodeF32(float32(radius)), uint64(sretPtr))
	if err != nil {
		return 0, HP_ShapeId{}, fmt.Errorf("havok: HP_Shape_CreateSphere: %w", err)
	}

	result := Result_OK
	if len(res) > 0 {
		result = Result(int32(res[0]))
	}
	shapeId, err := readU64(h.mem(), sretPtr)
	if err != nil {
		return 0, HP_ShapeId{}, err
	}
	return result, HP_ShapeId{shapeId}, nil
}

// HP_Shape_Release releases a shape.
//
// WASM: HP_Shape_Release(shapeId i64) -> (Result i32)
func (h *HavokPhysics) HP_Shape_Release(ctx context.Context, shapeId HP_ShapeId) (Result, error) {
	return h.releaseHandle(ctx, "HP_Shape_Release", shapeId[0])
}

// HP_Body_SetShape assigns a shape to a body.
//
// WASM: HP_Body_SetShape(bodyId i64, shapeId i64) -> (Result i32)
func (h *HavokPhysics) HP_Body_SetShape(ctx context.Context, bodyId HP_BodyId, shapeId HP_ShapeId) (Result, error) {
	return h.callResultI32(ctx, "HP_Body_SetShape", bodyId[0], shapeId[0])
}

// HP_Body_SetMotionType sets a body's motion type (STATIC/KINEMATIC/DYNAMIC).
//
// WASM: HP_Body_SetMotionType(bodyId i64, motionType i32) -> (Result i32)
func (h *HavokPhysics) HP_Body_SetMotionType(ctx context.Context, bodyId HP_BodyId, motionType MotionType) (Result, error) {
	return h.callResultI32(ctx, "HP_Body_SetMotionType", bodyId[0], uint64(motionType))
}

// HP_World_AddBody adds a body to a world.
//
// WASM: HP_World_AddBody(worldId i64, bodyId i64, activate i32) -> (Result i32)
func (h *HavokPhysics) HP_World_AddBody(ctx context.Context, worldId HP_WorldId, bodyId HP_BodyId, activate bool) (Result, error) {
	activateI32 := uint64(0)
	if activate {
		activateI32 = 1
	}
	return h.callResultI32(ctx, "HP_World_AddBody", worldId[0], bodyId[0], activateI32)
}

// =============================================================================
// Internal helpers
// =============================================================================

// createHandle calls a WASM function that returns [Result(i32), handle(i64)] where
// Result is the direct return value and handle is written to a 8-byte sret pointer.
//
// Actual WASM convention (verified from binary):
//
//	HP_World_Create(sretPtr i32) -> (i32 result)
//	sretPtr[0..7] = HP_*Id as i64 (handle value)
func (h *HavokPhysics) createHandle(ctx context.Context, fnName string) (Result, uint64, error) {
	sretPtr, err := h.alloc(ctx, 8) // 8 bytes for the i64 handle
	if err != nil {
		return 0, 0, err
	}
	defer h.freePtr(ctx, sretPtr)

	fn := h.mod.ExportedFunction(fnName)
	if fn == nil {
		return 0, 0, fmt.Errorf("havok: %s not exported", fnName)
	}
	res, err := fn.Call(ctx, uint64(sretPtr))
	if err != nil {
		return 0, 0, fmt.Errorf("havok: %s: %w", fnName, err)
	}

	// Result returned directly as i32
	result := Result_OK
	if len(res) > 0 {
		result = Result(int32(res[0]))
	}

	// Handle written to sret[0..7] as i64
	id, err := readU64(h.mem(), sretPtr)
	if err != nil {
		return 0, 0, err
	}
	return result, id, nil
}

// releaseHandle calls a "release" WASM function with the i64 handle as direct argument.
//
// E.g. HP_World_Release(worldId i64) -> (i32 result)
func (h *HavokPhysics) releaseHandle(ctx context.Context, fnName string, id uint64) (Result, error) {
	return h.callResultI32(ctx, fnName, id)
}

// callResultI32 calls a WASM function that returns i32 (Result) directly.
func (h *HavokPhysics) callResultI32(ctx context.Context, fnName string, args ...uint64) (Result, error) {
	fn := h.mod.ExportedFunction(fnName)
	if fn == nil {
		return 0, fmt.Errorf("havok: %s not exported", fnName)
	}
	res, err := fn.Call(ctx, args...)
	if err != nil {
		return 0, fmt.Errorf("havok: %s: %w", fnName, err)
	}
	if len(res) == 0 {
		return Result_OK, nil
	}
	return Result(int32(res[0])), nil
}
