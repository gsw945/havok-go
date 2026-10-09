package havok

import (
	"context"
	"math"
	"testing"

	havokwasm "github.com/gsw945/havok-go/havok/wasm"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func newEngine(t *testing.T) *HavokPhysics {
	t.Helper()
	if !havokwasm.IsReal() {
		t.Skip("havok/wasm/HavokPhysics.wasm is a placeholder — run `convert` first")
	}
	hp, err := NewFromBytes(context.Background(), havokwasm.WasmBytes)
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	t.Cleanup(func() { hp.Close() }) //nolint:errcheck
	return hp
}

// must returns a checker for one HP_* call so multi-value results can be
// forwarded directly: must(t, "what")(hp.HP_Foo(ctx, ...)).
func must(t *testing.T, what string) func(Result, error) {
	return func(r Result, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		if !r.IsOK() {
			t.Fatalf("%s: %s", what, r.Error())
		}
	}
}

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// -----------------------------------------------------------------------------
// 1. Enum constants must match the wasm's embind metadata.
//
// This is the regression guard for the original divergence from Babylon.js:
// HavokPhysics.d.ts lists MotionType as STATIC, KINEMATIC, DYNAMIC (0,1,2) and
// Result as 0..4, but the wasm actually registers MotionType as 1, 2, 4 and
// Result as HRESULT codes. Getting this wrong leaves every body KINEMATIC, so
// gravity and collision response silently do nothing.
// -----------------------------------------------------------------------------

func TestEnumsMatchWasm(t *testing.T) {
	enums := extractWasmEnums(t)

	checks := []struct {
		enum string
		name string
		want int32
	}{
		{"Result", "RESULT_OK", int32(Result_OK)},
		{"Result", "RESULT_FAIL", int32(Result_FAIL)},
		{"Result", "RESULT_INVALIDHANDLE", int32(Result_INVALIDHANDLE)},
		{"Result", "RESULT_INVALIDARGS", int32(Result_INVALIDARGS)},
		{"Result", "RESULT_NOTIMPLEMENTED", int32(Result_NOTIMPLEMENTED)},
		{"ShapeType", "COLLIDER", int32(ShapeType_COLLIDER)},
		{"ShapeType", "CONTAINER", int32(ShapeType_CONTAINER)},
		{"MotionType", "STATIC", int32(MotionType_STATIC)},
		{"MotionType", "KINEMATIC", int32(MotionType_KINEMATIC)},
		{"MotionType", "DYNAMIC", int32(MotionType_DYNAMIC)},
		{"EventType", "COLLISION_STARTED", int32(EventType_COLLISION_STARTED)},
		{"EventType", "COLLISION_CONTINUED", int32(EventType_COLLISION_CONTINUED)},
		{"EventType", "COLLISION_FINISHED", int32(EventType_COLLISION_FINISHED)},
		{"ActivationState", "ACTIVE", int32(ActivationState_ACTIVE)},
		{"ActivationState", "INACTIVE", int32(ActivationState_INACTIVE)},
		{"ActivationControl", "SIMULATION_CONTROLLED", int32(ActivationControl_SIMULATION_CONTROLLED)},
		{"ActivationControl", "ALWAYS_ACTIVE", int32(ActivationControl_ALWAYS_ACTIVE)},
		{"ActivationControl", "ALWAYS_INACTIVE", int32(ActivationControl_ALWAYS_INACTIVE)},
		{"ConstraintAxis", "LINEAR_X", int32(ConstraintAxis_LINEAR_X)},
		{"ConstraintAxis", "LINEAR_DISTANCE", int32(ConstraintAxis_LINEAR_DISTANCE)},
		{"ConstraintAxisLimitMode", "FREE", int32(ConstraintAxisLimitMode_FREE)},
		{"ConstraintAxisLimitMode", "LIMITED", int32(ConstraintAxisLimitMode_LIMITED)},
		{"ConstraintAxisLimitMode", "LOCKED", int32(ConstraintAxisLimitMode_LOCKED)},
		{"ConstraintMotorType", "NONE", int32(ConstraintMotorType_NONE)},
		{"ConstraintMotorType", "SPRING_ACCELERATION", int32(ConstraintMotorType_SPRING_ACCELERATION)},
		{"MaterialCombine", "GEOMETRIC_MEAN", int32(MaterialCombine_GEOMETRIC_MEAN)},
		{"MaterialCombine", "MULTIPLY", int32(MaterialCombine_MULTIPLY)},
	}

	for _, c := range checks {
		got, ok := enums[c.enum][c.name]
		if !ok {
			t.Errorf("wasm does not register %s.%s", c.enum, c.name)
			continue
		}
		if got != c.want {
			t.Errorf("%s.%s: binding says %d, wasm says %d", c.enum, c.name, c.want, got)
		}
	}

	if got := enums["MotionType"]["DYNAMIC"]; got != 4 {
		t.Errorf("wasm MotionType.DYNAMIC = %d, expected 4", got)
	}
}

// extractWasmEnums instantiates HavokPhysics.wasm with instrumented embind stubs
// and records every enum value the module registers at startup.
func extractWasmEnums(t *testing.T) map[string]map[string]int32 {
	t.Helper()
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	t.Cleanup(func() { r.Close(ctx) }) //nolint:errcheck

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		t.Fatalf("wasi: %v", err)
	}

	enumName := map[uint32]string{}
	result := map[string]map[string]int32{}
	var pending []struct {
		raw  uint32
		name string
		val  int32
	}
	readCStr := func(mod api.Module, ptr uint32) string {
		var out []byte
		for i := uint32(0); i < 256; i++ {
			b, ok := mod.Memory().Read(ptr+i, 1)
			if !ok || b[0] == 0 {
				break
			}
			out = append(out, b[0])
		}
		return string(out)
	}

	b := r.NewHostModuleBuilder("env")
	b.NewFunctionBuilder().WithFunc(func() {}).Export("_abort_js")
	b.NewFunctionBuilder().WithFunc(func(_ uint32) {}).Export("_embind_finalize_value_array")
	b.NewFunctionBuilder().WithFunc(func(_, _, _ uint32, _, _ uint64) {}).Export("_embind_register_bigint")
	b.NewFunctionBuilder().WithFunc(func(_, _, _, _ uint32) {}).Export("_embind_register_bool")
	b.NewFunctionBuilder().WithFunc(func(_ uint32) {}).Export("_embind_register_emval")
	b.NewFunctionBuilder().WithGoModuleFunction(api.GoModuleFunc(func(_ context.Context, mod api.Module, stack []uint64) {
		enumName[uint32(stack[0])] = readCStr(mod, uint32(stack[1]))
	}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, nil).Export("_embind_register_enum")
	b.NewFunctionBuilder().WithGoModuleFunction(api.GoModuleFunc(func(_ context.Context, mod api.Module, stack []uint64) {
		pending = append(pending, struct {
			raw  uint32
			name string
			val  int32
		}{uint32(stack[0]), readCStr(mod, uint32(stack[1])), int32(stack[2])})
	}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, nil).Export("_embind_register_enum_value")
	b.NewFunctionBuilder().WithFunc(func(_, _, _ uint32) {}).Export("_embind_register_float")
	b.NewFunctionBuilder().WithFunc(func(_, _, _, _, _, _, _, _ uint32) {}).Export("_embind_register_function")
	b.NewFunctionBuilder().WithFunc(func(_, _, _, _, _ uint32) {}).Export("_embind_register_integer")
	b.NewFunctionBuilder().WithFunc(func(_, _, _ uint32) {}).Export("_embind_register_memory_view")
	b.NewFunctionBuilder().WithFunc(func(_, _ uint32) {}).Export("_embind_register_std_string")
	b.NewFunctionBuilder().WithFunc(func(_, _, _ uint32) {}).Export("_embind_register_std_wstring")
	b.NewFunctionBuilder().WithFunc(func(_, _, _, _, _, _ uint32) {}).Export("_embind_register_value_array")
	b.NewFunctionBuilder().WithFunc(func(_, _, _, _, _, _, _, _, _ uint32) {}).Export("_embind_register_value_array_element")
	b.NewFunctionBuilder().WithFunc(func(_, _ uint32) {}).Export("_embind_register_void")
	b.NewFunctionBuilder().WithFunc(func(_, _, _ uint32) uint32 { return 0 }).Export("_emval_get_method_caller")
	b.NewFunctionBuilder().WithFunc(func(_, _, _, _, _ uint32) float64 { return 0 }).Export("_emval_call_method")
	b.NewFunctionBuilder().WithFunc(func(_ uint32) {}).Export("_emval_decref")
	b.NewFunctionBuilder().WithFunc(func(_ uint32) {}).Export("_emval_run_destructors")
	b.NewFunctionBuilder().WithFunc(func() uint32 { return 1 }).Export("_emscripten_get_now_is_monotonic")
	b.NewFunctionBuilder().WithFunc(func() float64 { return 0 }).Export("emscripten_get_now")
	b.NewFunctionBuilder().WithFunc(func() float64 { return 0 }).Export("emscripten_date_now")
	b.NewFunctionBuilder().WithFunc(func() uint32 { return 2147483648 }).Export("emscripten_get_heap_max")
	b.NewFunctionBuilder().WithGoModuleFunction(api.GoModuleFunc(func(_ context.Context, mod api.Module, stack []uint64) {
		req := uint32(api.DecodeU32(stack[0]))
		mem := mod.Memory()
		if mem == nil {
			stack[0] = api.EncodeU32(0)
			return
		}
		old := mem.Size()
		if req > old {
			pages := (req - old + 65535) / 65536
			if grown, ok := mem.Grow(pages); !ok || grown == 0 {
				stack[0] = api.EncodeU32(0)
				return
			}
		}
		stack[0] = api.EncodeU32(1)
	}), []api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).Export("emscripten_resize_heap")
	if _, err := b.Instantiate(ctx); err != nil {
		t.Fatalf("env module: %v", err)
	}

	mod, err := r.InstantiateWithConfig(ctx, havokwasm.WasmBytes, wazero.NewModuleConfig().WithStartFunctions())
	if err != nil {
		t.Fatalf("instantiate wasm: %v", err)
	}
	if ctors := mod.ExportedFunction("__wasm_call_ctors"); ctors != nil {
		if _, err := ctors.Call(ctx); err != nil {
			t.Fatalf("__wasm_call_ctors: %v", err)
		}
	}
	if mainFn := mod.ExportedFunction("main"); mainFn != nil {
		mainFn.Call(ctx, 0, 0) //nolint:errcheck
	}

	for _, p := range pending {
		name := enumName[p.raw]
		if result[name] == nil {
			result[name] = map[string]int32{}
		}
		result[name][p.name] = p.val
	}
	if len(result) == 0 {
		t.Fatal("no enums were registered by the wasm")
	}
	return result
}

// -----------------------------------------------------------------------------
// 2. A dynamic cube must fall and rest on a static ground.
// -----------------------------------------------------------------------------

func TestFallingBoxSettlesOnGround(t *testing.T) {
	ctx := context.Background()
	hp := newEngine(t)

	_, world, err := hp.HP_World_Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer hp.HP_World_Release(ctx, world) //nolint:errcheck
	must(t, "SetGravity")(hp.HP_World_SetGravity(ctx, world, Vector3{0, -9.81, 0}))

	const dt = 1.0 / 60.0
	const size = 1.0

	// static ground: top face at y = 0
	_, groundShape, err := hp.HP_Shape_CreateBox(ctx, Vector3{0, -0.5, 0}, Quaternion{0, 0, 0, 1}, Vector3{100, 1, 100})
	if err != nil {
		t.Fatal(err)
	}
	defer hp.HP_Shape_Release(ctx, groundShape) //nolint:errcheck
	_, groundBody, err := hp.HP_Body_Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer hp.HP_Body_Release(ctx, groundBody) //nolint:errcheck
	must(t, "ground SetShape")(hp.HP_Body_SetShape(ctx, groundBody, groundShape))
	must(t, "ground SetMotionType")(hp.HP_Body_SetMotionType(ctx, groundBody, MotionType_STATIC))
	must(t, "ground AddBody")(hp.HP_World_AddBody(ctx, world, groundBody, false))

	// dynamic cube
	_, cubeShape, err := hp.HP_Shape_CreateBox(ctx, Vector3{0, 0, 0}, Quaternion{0, 0, 0, 1}, Vector3{size, size, size})
	if err != nil {
		t.Fatal(err)
	}
	defer hp.HP_Shape_Release(ctx, cubeShape) //nolint:errcheck
	_, cube, err := hp.HP_Body_Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer hp.HP_Body_Release(ctx, cube) //nolint:errcheck
	must(t, "cube SetShape")(hp.HP_Body_SetShape(ctx, cube, cubeShape))
	must(t, "cube SetMotionType")(hp.HP_Body_SetMotionType(ctx, cube, MotionType_DYNAMIC))

	inertia := (size*size + size*size) / 12.0
	must(t, "cube SetMassProperties")(hp.HP_Body_SetMassProperties(ctx, cube, MassProperties{
		Mass: 1, InertiaTensorDiag: Vector3{inertia, inertia, inertia}, InertiaOrientation: Quaternion{0, 0, 0, 1},
	}))
	must(t, "cube SetLinearDamping")(hp.HP_Body_SetLinearDamping(ctx, cube, 5.0))
	must(t, "cube SetAngularDamping")(hp.HP_Body_SetAngularDamping(ctx, cube, 50.0))
	must(t, "cube SetQTransform")(hp.HP_Body_SetQTransform(ctx, cube, QTransform{
		Translation: Vector3{0, 2, 0}, Rotation: Quaternion{0, 0, 0, 1},
	}))
	must(t, "cube AddBody")(hp.HP_World_AddBody(ctx, world, cube, false))

	// A DYNAMIC body must respond to gravity immediately. If the motion type
	// were KINEMATIC (the original bug) the body would stay perfectly still.
	for i := 0; i < 3; i++ {
		must(t, "step")(hp.HP_World_Step(ctx, world, dt))
	}
	_, earlyVel, err := hp.HP_Body_GetLinearVelocity(ctx, cube)
	if err != nil {
		t.Fatal(err)
	}
	if earlyVel[1] >= 0 {
		t.Fatalf("gravity not applied: vy=%.4f after 3 steps (expected < 0)", earlyVel[1])
	}

	for i := 0; i < 240; i++ {
		must(t, "step")(hp.HP_World_Step(ctx, world, dt))
	}

	_, pos, err := hp.HP_Body_GetPosition(ctx, cube)
	if err != nil {
		t.Fatal(err)
	}
	_, vel, err := hp.HP_Body_GetLinearVelocity(ctx, cube)
	if err != nil {
		t.Fatal(err)
	}
	if !near(pos[1], size/2, 0.01) {
		t.Errorf("resting height = %.4f, want ~%.2f", pos[1], size/2)
	}
	if !near(vel[1], 0, 0.05) {
		t.Errorf("resting vy = %.4f, want ~0", vel[1])
	}
}

// -----------------------------------------------------------------------------
// 3. Impulse → velocity (v = J/m).
// -----------------------------------------------------------------------------

func TestImpulseSetsVelocity(t *testing.T) {
	ctx := context.Background()
	hp := newEngine(t)

	_, world, _ := hp.HP_World_Create(ctx)
	defer hp.HP_World_Release(ctx, world) //nolint:errcheck

	_, shape, _ := hp.HP_Shape_CreateSphere(ctx, Vector3{0, 0, 0}, 0.5)
	defer hp.HP_Shape_Release(ctx, shape) //nolint:errcheck
	_, body, _ := hp.HP_Body_Create(ctx)
	defer hp.HP_Body_Release(ctx, body) //nolint:errcheck
	must(t, "SetShape")(hp.HP_Body_SetShape(ctx, body, shape))
	must(t, "SetMotionType")(hp.HP_Body_SetMotionType(ctx, body, MotionType_DYNAMIC))
	must(t, "SetMassProperties")(hp.HP_Body_SetMassProperties(ctx, body, MassProperties{
		Mass: 2, InertiaTensorDiag: Vector3{1, 1, 1}, InertiaOrientation: Quaternion{0, 0, 0, 1},
	}))
	must(t, "AddBody")(hp.HP_World_AddBody(ctx, world, body, false))

	// J = 8 N·s on m = 2 kg  =>  v = 4 m/s
	must(t, "ApplyImpulse")(hp.HP_Body_ApplyImpulse(ctx, body, Vector3{0, 0, 0}, Vector3{8, 0, 0}))
	_, vel, err := hp.HP_Body_GetLinearVelocity(ctx, body)
	if err != nil {
		t.Fatal(err)
	}
	if !near(vel[0], 4, 1e-3) {
		t.Errorf("vx = %.5f, want 4.0 (J/m)", vel[0])
	}

	must(t, "SetLinearVelocity")(hp.HP_Body_SetLinearVelocity(ctx, body, Vector3{1, -2, 3}))
	_, vel2, _ := hp.HP_Body_GetLinearVelocity(ctx, body)
	if !near(vel2[0], 1, 1e-5) || !near(vel2[1], -2, 1e-5) || !near(vel2[2], 3, 1e-5) {
		t.Errorf("SetLinearVelocity round-trip = %v, want (1,-2,3)", vel2)
	}
}

// -----------------------------------------------------------------------------
// 4. Transform round-trip and static-body immobility.
// -----------------------------------------------------------------------------

func TestTransformAndStatic(t *testing.T) {
	ctx := context.Background()
	hp := newEngine(t)

	_, world, _ := hp.HP_World_Create(ctx)
	defer hp.HP_World_Release(ctx, world) //nolint:errcheck
	must(t, "SetGravity")(hp.HP_World_SetGravity(ctx, world, Vector3{0, -9.81, 0}))

	_, shape, _ := hp.HP_Shape_CreateBox(ctx, Vector3{0, 0, 0}, Quaternion{0, 0, 0, 1}, Vector3{1, 1, 1})
	defer hp.HP_Shape_Release(ctx, shape) //nolint:errcheck

	// STATIC body: must not move under gravity.
	_, stat, _ := hp.HP_Body_Create(ctx)
	defer hp.HP_Body_Release(ctx, stat) //nolint:errcheck
	must(t, "static SetShape")(hp.HP_Body_SetShape(ctx, stat, shape))
	must(t, "static SetMotionType")(hp.HP_Body_SetMotionType(ctx, stat, MotionType_STATIC))
	must(t, "static AddBody")(hp.HP_World_AddBody(ctx, world, stat, false))
	must(t, "static SetQTransform")(hp.HP_Body_SetQTransform(ctx, stat, QTransform{
		Translation: Vector3{3, 7, -2}, Rotation: Quaternion{0, 0, 0, 1},
	}))
	for i := 0; i < 60; i++ {
		must(t, "step")(hp.HP_World_Step(ctx, world, 1.0/60.0))
	}
	_, sp, _ := hp.HP_Body_GetPosition(ctx, stat)
	if !near(sp[0], 3, 1e-4) || !near(sp[1], 7, 1e-4) || !near(sp[2], -2, 1e-4) {
		t.Errorf("STATIC body moved to %v, want (3,7,-2)", sp)
	}

	// QTransform round-trip with a non-identity rotation (90° about Y).
	_, dyn, _ := hp.HP_Body_Create(ctx)
	defer hp.HP_Body_Release(ctx, dyn) //nolint:errcheck
	must(t, "dyn SetShape")(hp.HP_Body_SetShape(ctx, dyn, shape))
	must(t, "dyn SetMotionType")(hp.HP_Body_SetMotionType(ctx, dyn, MotionType_DYNAMIC))
	must(t, "dyn AddBody")(hp.HP_World_AddBody(ctx, world, dyn, false))
	s := math.Sqrt2 / 2
	want := QTransform{Translation: Vector3{1.5, 4.25, -0.5}, Rotation: Quaternion{0, s, 0, s}}
	must(t, "dyn SetQTransform")(hp.HP_Body_SetQTransform(ctx, dyn, want))
	_, got, err := hp.HP_Body_GetQTransform(ctx, dyn)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if !near(got.Translation[i], want.Translation[i], 1e-4) {
			t.Errorf("QTransform translation[%d] = %.5f, want %.5f", i, got.Translation[i], want.Translation[i])
		}
	}
	for i := 0; i < 4; i++ {
		if !near(got.Rotation[i], want.Rotation[i], 1e-4) {
			t.Errorf("QTransform rotation[%d] = %.5f, want %.5f", i, got.Rotation[i], want.Rotation[i])
		}
	}
}

// -----------------------------------------------------------------------------
// 5. Mass properties round-trip through the f32 wire layout.
// -----------------------------------------------------------------------------

func TestMassPropertiesRoundTrip(t *testing.T) {
	ctx := context.Background()
	hp := newEngine(t)

	_, shape, _ := hp.HP_Shape_CreateSphere(ctx, Vector3{0, 0, 0}, 1)
	defer hp.HP_Shape_Release(ctx, shape) //nolint:errcheck
	_, body, _ := hp.HP_Body_Create(ctx)
	defer hp.HP_Body_Release(ctx, body) //nolint:errcheck
	must(t, "SetShape")(hp.HP_Body_SetShape(ctx, body, shape))

	want := MassProperties{
		CenterOfMass:       Vector3{0.25, -0.5, 0.75},
		Mass:               2.5,
		InertiaTensorDiag:  Vector3{1.5, 2.5, 3.5},
		InertiaOrientation: Quaternion{0, 0, 0, 1},
	}
	must(t, "SetMassProperties")(hp.HP_Body_SetMassProperties(ctx, body, want))
	_, got, err := hp.HP_Body_GetMassProperties(ctx, body)
	if err != nil {
		t.Fatal(err)
	}
	if !near(got.Mass, want.Mass, 1e-4) {
		t.Errorf("mass = %v, want %v", got.Mass, want.Mass)
	}
	for i := 0; i < 3; i++ {
		if !near(got.CenterOfMass[i], want.CenterOfMass[i], 1e-4) {
			t.Errorf("centerOfMass[%d] = %v, want %v", i, got.CenterOfMass[i], want.CenterOfMass[i])
		}
		if !near(got.InertiaTensorDiag[i], want.InertiaTensorDiag[i], 1e-4) {
			t.Errorf("inertia[%d] = %v, want %v", i, got.InertiaTensorDiag[i], want.InertiaTensorDiag[i])
		}
	}
}

// -----------------------------------------------------------------------------
// 6. Material round-trip and ray cast (exercises the query struct layouts).
// -----------------------------------------------------------------------------

func TestMaterialAndRayCast(t *testing.T) {
	ctx := context.Background()
	hp := newEngine(t)

	_, world, _ := hp.HP_World_Create(ctx)
	defer hp.HP_World_Release(ctx, world) //nolint:errcheck

	_, shape, _ := hp.HP_Shape_CreateSphere(ctx, Vector3{0, 0, 0}, 1)
	defer hp.HP_Shape_Release(ctx, shape) //nolint:errcheck
	wantMat := PhysicsMaterial{StaticFriction: 0.5, DynamicFriction: 0.3, Restitution: 0.2}
	must(t, "SetMaterial")(hp.HP_Shape_SetMaterial(ctx, shape, wantMat))
	_, gotMat, err := hp.HP_Shape_GetMaterial(ctx, shape)
	if err != nil {
		t.Fatal(err)
	}
	if !near(float64(gotMat.StaticFriction), 0.5, 1e-5) ||
		!near(float64(gotMat.DynamicFriction), 0.3, 1e-5) ||
		!near(float64(gotMat.Restitution), 0.2, 1e-5) {
		t.Errorf("material round-trip = %+v, want %+v", gotMat, wantMat)
	}

	_, body, _ := hp.HP_Body_Create(ctx)
	defer hp.HP_Body_Release(ctx, body) //nolint:errcheck
	must(t, "SetShape")(hp.HP_Body_SetShape(ctx, body, shape))
	must(t, "SetMotionType")(hp.HP_Body_SetMotionType(ctx, body, MotionType_STATIC))
	must(t, "AddBody")(hp.HP_World_AddBody(ctx, world, body, false))
	must(t, "step")(hp.HP_World_Step(ctx, world, 1.0/60.0))

	_, collector, err := hp.HP_QueryCollector_Create(ctx, 16)
	if err != nil {
		t.Fatal(err)
	}
	defer hp.HP_QueryCollector_Release(ctx, collector) //nolint:errcheck

	// Ray from z=-5 towards +z through a unit sphere at the origin: the first
	// hit is at z=-1, i.e. 40% along the 10 m ray.
	all := FilterInfo{MembershipMask: 0xFFFFFFFF, CollisionMask: 0xFFFFFFFF}
	must(t, "CastRay")(hp.HP_World_CastRayWithCollector(ctx, world, collector, RayCastInput{
		Start: Vector3{0, 0, -5}, End: Vector3{0, 0, 5}, Filter: all,
	}))
	_, hits, err := hp.HP_QueryCollector_GetNumHits(ctx, collector)
	if err != nil {
		t.Fatal(err)
	}
	if hits < 1 {
		t.Fatalf("ray cast reported %v hits, want >= 1", hits)
	}
	_, res, err := hp.HP_QueryCollector_GetCastRayResult(ctx, collector, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !near(res.Fraction, 0.4, 0.02) {
		t.Errorf("ray fraction = %.4f, want ~0.4", res.Fraction)
	}
	if !near(res.Contact.Position[2], -1, 0.05) {
		t.Errorf("hit z = %.4f, want ~-1", res.Contact.Position[2])
	}
	if !near(res.Contact.Normal[2], -1, 0.05) {
		t.Errorf("hit normal z = %.4f, want ~-1", res.Contact.Normal[2])
	}
	if res.Contact.BodyId[0] != body[0] {
		t.Errorf("hit body = %d, want %d", res.Contact.BodyId[0], body[0])
	}

	// ignoreBody must suppress the hit — this exercises the 8-byte-aligned
	// layout of the query input structs.
	must(t, "CastRay")(hp.HP_World_CastRayWithCollector(ctx, world, collector, RayCastInput{
		Start: Vector3{0, 0, -5}, End: Vector3{0, 0, 5}, Filter: all, IgnoreBody: body,
	}))
	_, hits2, _ := hp.HP_QueryCollector_GetNumHits(ctx, collector)
	if hits2 != 0 {
		t.Errorf("ray cast ignoring the only body reported %v hits, want 0", hits2)
	}
}

// -----------------------------------------------------------------------------
// 7. Determinism: identical worlds must produce identical trajectories.
// -----------------------------------------------------------------------------

func TestDeterministic(t *testing.T) {
	ctx := context.Background()
	hp := newEngine(t)

	run := func() Vector3 {
		_, world, _ := hp.HP_World_Create(ctx)
		defer hp.HP_World_Release(ctx, world) //nolint:errcheck
		must(t, "gravity")(hp.HP_World_SetGravity(ctx, world, Vector3{0, -9.81, 0}))
		_, shape, _ := hp.HP_Shape_CreateBox(ctx, Vector3{0, 0, 0}, Quaternion{0, 0, 0, 1}, Vector3{1, 1, 1})
		defer hp.HP_Shape_Release(ctx, shape) //nolint:errcheck
		_, body, _ := hp.HP_Body_Create(ctx)
		defer hp.HP_Body_Release(ctx, body) //nolint:errcheck
		must(t, "shape")(hp.HP_Body_SetShape(ctx, body, shape))
		must(t, "motion")(hp.HP_Body_SetMotionType(ctx, body, MotionType_DYNAMIC))
		must(t, "mass")(hp.HP_Body_SetMassProperties(ctx, body, MassProperties{
			Mass: 1, InertiaTensorDiag: Vector3{1.0 / 6, 1.0 / 6, 1.0 / 6}, InertiaOrientation: Quaternion{0, 0, 0, 1},
		}))
		must(t, "xform")(hp.HP_Body_SetQTransform(ctx, body, QTransform{
			Translation: Vector3{0, 10, 0}, Rotation: Quaternion{0, 0, 0, 1},
		}))
		must(t, "add")(hp.HP_World_AddBody(ctx, world, body, false))
		for i := 0; i < 120; i++ {
			must(t, "step")(hp.HP_World_Step(ctx, world, 1.0/60.0))
		}
		_, p, _ := hp.HP_Body_GetPosition(ctx, body)
		return p
	}

	a, b := run(), run()
	if a != b {
		t.Errorf("non-deterministic: %v vs %v", a, b)
	}
	if a[1] >= 10 {
		t.Errorf("body did not fall: y=%v", a[1])
	}
}
