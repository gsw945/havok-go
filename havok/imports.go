package havok

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// registerEnvModule sets up all emscripten env host functions as stubs.
// These are called during __wasm_call_ctors (embind registration) but we
// call the HP_* exports directly, so stubs suffice.
func registerEnvModule(ctx context.Context, r wazero.Runtime) error {
	b := r.NewHostModuleBuilder("env")

	// --- abort ---
	b.NewFunctionBuilder().
		WithFunc(func() {}).
		Export("_abort_js")

	// --- embind value array (tuple type registration) ---
	b.NewFunctionBuilder().
		WithFunc(func(_ uint32) {}).
		Export("_embind_finalize_value_array")

	b.NewFunctionBuilder().
		WithFunc(func(_, _, _ uint32, _, _ uint64) {}).
		Export("_embind_register_bigint")

	b.NewFunctionBuilder().
		WithFunc(func(_, _, _, _ uint32) {}).
		Export("_embind_register_bool")

	b.NewFunctionBuilder().
		WithFunc(func(_ uint32) {}).
		Export("_embind_register_emval")

	b.NewFunctionBuilder().
		WithFunc(func(_, _, _, _ uint32) {}).
		Export("_embind_register_enum")

	b.NewFunctionBuilder().
		WithFunc(func(_, _, _ uint32) {}).
		Export("_embind_register_enum_value")

	b.NewFunctionBuilder().
		WithFunc(func(_, _, _ uint32) {}).
		Export("_embind_register_float")

	// (name, argCount, rawArgTypesAddr, signature, rawInvoker, fn, isAsync, isNonnullReturn)
	b.NewFunctionBuilder().
		WithFunc(func(_, _, _, _, _, _, _, _ uint32) {}).
		Export("_embind_register_function")

	// (primitiveType, name, size, minRange, maxRange)
	b.NewFunctionBuilder().
		WithFunc(func(_, _, _, _, _ uint32) {}).
		Export("_embind_register_integer")

	b.NewFunctionBuilder().
		WithFunc(func(_, _, _ uint32) {}).
		Export("_embind_register_memory_view")

	b.NewFunctionBuilder().
		WithFunc(func(_, _ uint32) {}).
		Export("_embind_register_std_string")

	b.NewFunctionBuilder().
		WithFunc(func(_, _, _ uint32) {}).
		Export("_embind_register_std_wstring")

	// (rawType, name, ctorSig, ctor, dtorSig, dtor)
	b.NewFunctionBuilder().
		WithFunc(func(_, _, _, _, _, _ uint32) {}).
		Export("_embind_register_value_array")

	// 9 params
	b.NewFunctionBuilder().
		WithFunc(func(_, _, _, _, _, _, _, _, _ uint32) {}).
		Export("_embind_register_value_array_element")

	b.NewFunctionBuilder().
		WithFunc(func(_, _ uint32) {}).
		Export("_embind_register_void")

	// --- emval (dynamic value system) ---
	// (argCount, argTypes, kind) -> callerId i32
	b.NewFunctionBuilder().
		WithFunc(func(_, _, _ uint32) uint32 { return 0 }).
		Export("_emval_get_method_caller")

	// (caller, objHandle, methodName, destructorsRef, args) -> f64 (emscripten returns double to cover all JS types)
	b.NewFunctionBuilder().
		WithFunc(func(_, _, _, _, _ uint32) float64 { return 0 }).
		Export("_emval_call_method")

	b.NewFunctionBuilder().
		WithFunc(func(_ uint32) {}).
		Export("_emval_decref")

	b.NewFunctionBuilder().
		WithFunc(func(_ uint32) {}).
		Export("_emval_run_destructors")

	// --- emscripten timing ---
	b.NewFunctionBuilder().
		WithFunc(func() uint32 { return 1 }). // monotonic = true
		Export("_emscripten_get_now_is_monotonic")

	b.NewFunctionBuilder().
		WithFunc(func() float64 { return 0 }).
		Export("emscripten_get_now")

	b.NewFunctionBuilder().
		WithFunc(func() float64 { return 0 }).
		Export("emscripten_date_now")

	// --- emscripten memory ---
	b.NewFunctionBuilder().
		WithFunc(func() uint32 { return 2147483648 }). // 2 GiB max
		Export("emscripten_get_heap_max")

	// resize_heap: called when WASM needs more memory. Return 0 = fail.
	// wazero manages runtime memory growth; this stub is fine for Havok's usage.
	b.NewFunctionBuilder().
		WithGoModuleFunction(
			api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
				requestedSize := uint32(api.DecodeU32(stack[0]))
				mem := mod.Memory()
				if mem != nil {
					oldSize := mem.Size()
					if requestedSize > oldSize {
						pages := (requestedSize - oldSize + 65535) / 65536
						grown, ok := mem.Grow(pages)
						if !ok || grown == 0 {
							stack[0] = api.EncodeU32(0) // fail
							return
						}
					}
					stack[0] = api.EncodeU32(1) // success
					return
				}
				stack[0] = api.EncodeU32(0)
			}),
			[]api.ValueType{api.ValueTypeI32},
			[]api.ValueType{api.ValueTypeI32},
		).
		Export("emscripten_resize_heap")

	_, err := b.Instantiate(ctx)
	return err
}
