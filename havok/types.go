package havok

// =============================================================================
// Primitive Types (mirroring HavokPhysics.d.ts)
// =============================================================================

// Vector3 is [x, y, z] stored as float64
type Vector3 [3]float64

// Quaternion is [x, y, z, w] stored as float64
type Quaternion [4]float64

// HP_BodyId is a 64-bit body handle
type HP_BodyId [1]uint64

// HP_ShapeId is a 64-bit shape handle
type HP_ShapeId [1]uint64

// HP_WorldId is a 64-bit world handle
type HP_WorldId [1]uint64

// HP_ConstraintId is a 64-bit constraint handle
type HP_ConstraintId [1]uint64

// HP_CollectorId is a 64-bit query collector handle
type HP_CollectorId [1]uint64

// HP_DebugGeometryId is a 64-bit debug geometry handle
type HP_DebugGeometryId [1]uint64

// =============================================================================
// Enum Types
// =============================================================================

// Result is the return status of an HP_ operation
type Result int32

const (
	Result_OK             Result = 0
	Result_FAIL           Result = 1
	Result_INVALIDHANDLE  Result = 2
	Result_INVALIDARGS    Result = 3
	Result_NOTIMPLEMENTED Result = 4
)

func (r Result) IsOK() bool { return r == Result_OK }
func (r Result) Error() string {
	switch r {
	case Result_OK:
		return "OK"
	case Result_FAIL:
		return "FAIL"
	case Result_INVALIDHANDLE:
		return "INVALID_HANDLE"
	case Result_INVALIDARGS:
		return "INVALID_ARGS"
	case Result_NOTIMPLEMENTED:
		return "NOT_IMPLEMENTED"
	default:
		return "UNKNOWN"
	}
}

// MotionType controls how a body moves in the simulation
type MotionType int32

const (
	MotionType_STATIC    MotionType = 0
	MotionType_KINEMATIC MotionType = 1
	MotionType_DYNAMIC   MotionType = 2
)

// ShapeType distinguishes container vs collider shapes
type ShapeType int32

const (
	ShapeType_COLLIDER  ShapeType = 0
	ShapeType_CONTAINER ShapeType = 1
)

// ActivationState reflects whether a body is simulating
type ActivationState int32

const (
	ActivationState_ACTIVE   ActivationState = 0
	ActivationState_INACTIVE ActivationState = 1
)

// ActivationControl determines what controls a body's activation
type ActivationControl int32

const (
	ActivationControl_SIMULATION_CONTROLLED ActivationControl = 0
	ActivationControl_ALWAYS_ACTIVE         ActivationControl = 1
	ActivationControl_ALWAYS_INACTIVE       ActivationControl = 2
)

// ConstraintAxis identifies a constraint axis
type ConstraintAxis int32

const (
	ConstraintAxis_LINEAR_X        ConstraintAxis = 0
	ConstraintAxis_LINEAR_Y        ConstraintAxis = 1
	ConstraintAxis_LINEAR_Z        ConstraintAxis = 2
	ConstraintAxis_ANGULAR_X       ConstraintAxis = 3
	ConstraintAxis_ANGULAR_Y       ConstraintAxis = 4
	ConstraintAxis_ANGULAR_Z       ConstraintAxis = 5
	ConstraintAxis_LINEAR_DISTANCE ConstraintAxis = 6
)

// MaterialCombine controls how material properties are combined on contact
type MaterialCombine int32

const (
	MaterialCombine_GEOMETRIC_MEAN  MaterialCombine = 0
	MaterialCombine_MINIMUM         MaterialCombine = 1
	MaterialCombine_MAXIMUM         MaterialCombine = 2
	MaterialCombine_ARITHMETIC_MEAN MaterialCombine = 3
	MaterialCombine_MULTIPLY        MaterialCombine = 4
)

// =============================================================================
// Composite Types
// =============================================================================

// ObjectStatistics contains counts of allocated native objects
// Memory layout (sret): 7 consecutive i32s (28 bytes total)
//   offset 0:  Result
//   offset 4:  NumBodies
//   offset 8:  NumShapes
//   offset 12: NumConstraints
//   offset 16: NumDebugGeometries
//   offset 20: NumWorlds
//   offset 24: NumQueryCollectors
type ObjectStatistics struct {
	NumBodies          int32
	NumShapes          int32
	NumConstraints     int32
	NumDebugGeometries int32
	NumWorlds          int32
	NumQueryCollectors int32
}

// FilterInfo contains the collision membership and collision masks
// Memory layout: 2 * i32 = 8 bytes
type FilterInfo struct {
	MembershipMask uint32
	CollisionMask  uint32
}

// PhysicsMaterial defines surface interaction properties
// Memory layout: 3 * f32 + 2 * i32 = 20 bytes
type PhysicsMaterial struct {
	StaticFriction     float32
	DynamicFriction    float32
	Restitution        float32
	FrictionCombine    MaterialCombine
	RestitutionCombine MaterialCombine
}

// MassProperties are used to configure dynamic body mass
// Memory layout: Vector3(24) + f64(8) + Vector3(24) + Quaternion(32) = 88 bytes
type MassProperties struct {
	CenterOfMass       Vector3
	Mass               float64
	InertiaTensorDiag  Vector3 // inertia for mass of 1
	InertiaOrientation Quaternion
}

// QTransform is a position + rotation (no scale)
// Memory layout: Vector3(24) + Quaternion(32) = 56 bytes
type QTransform struct {
	Translation Vector3
	Rotation    Quaternion
}
