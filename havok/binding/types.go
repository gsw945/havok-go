// Package binding: hand-crafted type declarations mirroring HavokPhysics.d.ts.
// These are NOT generated — they are manually maintained to ensure correctness.
// bindings_gen.go (generated) depends on types defined here.
package binding

// =============================================================================
// Primitive Types
// =============================================================================

// Vector3 is [x, y, z] stored as float64 (WASM uses f32 on the wire)
type Vector3 [3]float64

// Quaternion is [x, y, z, w] stored as float64 (WASM uses f32 on the wire)
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

// EventType describes the kind of a physics event
type EventType int32

const (
	EventType_COLLISION_STARTED   EventType = 0
	EventType_COLLISION_CONTINUED EventType = 1
	EventType_COLLISION_FINISHED  EventType = 2
	EventType_TRIGGER_ENTERED     EventType = 3
	EventType_TRIGGER_EXITED      EventType = 4
)

// ConstraintAxisLimitMode describes constraint axis limit behaviour
type ConstraintAxisLimitMode int32

const (
	ConstraintAxisLimitMode_FREE    ConstraintAxisLimitMode = 0
	ConstraintAxisLimitMode_LIMITED ConstraintAxisLimitMode = 1
	ConstraintAxisLimitMode_LOCKED  ConstraintAxisLimitMode = 2
)

// ConstraintMotorType describes the motor type on a constraint axis
type ConstraintMotorType int32

const (
	ConstraintMotorType_NONE                ConstraintMotorType = 0
	ConstraintMotorType_VELOCITY            ConstraintMotorType = 1
	ConstraintMotorType_POSITION            ConstraintMotorType = 2
	ConstraintMotorType_SPRING_FORCE        ConstraintMotorType = 3
	ConstraintMotorType_SPRING_ACCELERATION ConstraintMotorType = 4
)

// =============================================================================
// Composite Types
// =============================================================================

// ObjectStatistics contains counts of allocated native objects
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
type MassProperties struct {
	CenterOfMass       Vector3
	Mass               float64
	InertiaTensorDiag  Vector3
	InertiaOrientation Quaternion
}

// QTransform is a position + rotation (no scale)
// Memory layout: Vector3(12) + Quaternion(16) = 28 bytes (WASM f32 layout)
type QTransform struct {
	Translation Vector3
	Rotation    Quaternion
}

// QSTransform is a position + rotation + scale
type QSTransform struct {
	Translation Vector3
	Rotation    Quaternion
	Scale       Vector3
}

// Aabb is an axis-aligned bounding box
type Aabb struct {
	Min Vector3
	Max Vector3
}

// ShapePathIterator is a pair of bigint handles for shape hierarchy traversal
type ShapePathIterator struct {
	ShapeId  uint64
	PathData uint64
}

// ContactPoint contains contact information for a collision
type ContactPoint struct {
	BodyId         HP_BodyId
	ColliderId     HP_ShapeId
	ShapeHierarchy ShapePathIterator
	Position       Vector3
	Normal         Vector3
	TriangleIndex  float64
}

// DebugGeometryInfo contains pointers and counts for debug visualization buffers
type DebugGeometryInfo struct {
	VertexBufferPtr   uint32
	NumVertices       uint32
	TriangleBufferPtr uint32
	NumTriangles      uint32
}

// CollisionEvent is emitted for contact start/continue/end events
type CollisionEvent struct {
	Type     EventType
	ContactA ContactPoint
	ContactB ContactPoint
	Impulse  float64
}

// TriggerEvent is emitted when a body enters or exits a trigger shape
type TriggerEvent struct {
	Type   EventType
	BodyA  HP_BodyId
	ShapeA HP_ShapeId
	BodyB  HP_BodyId
	ShapeB HP_ShapeId
}

// BodyMovedEvent reports the new world-from-body transform after a simulation step
type BodyMovedEvent struct {
	Type          EventType
	Body          HP_BodyId
	WorldFromBody QTransform
}

// RayCastResult holds the result of a ray cast query
type RayCastResult struct {
	Fraction float64
	Contact  ContactPoint
}

// PointProximityResult holds the result of a point proximity query
type PointProximityResult struct {
	Distance float64
	Contact  ContactPoint
}

// ShapeCastResult holds the result of a shape cast (sweep) query
type ShapeCastResult struct {
	Fraction        float64
	ContactOnInput  ContactPoint
	ContactOnTarget ContactPoint
}

// ShapeProximityResult holds the result of a shape proximity query
type ShapeProximityResult struct {
	Distance        float64
	ContactOnInput  ContactPoint
	ContactOnTarget ContactPoint
}

// RayCastInput describes a ray cast query
type RayCastInput struct {
	Start       Vector3
	End         Vector3
	Filter      FilterInfo
	HitTriggers bool
	IgnoreBody  HP_BodyId
}

// PointProximityInput describes a point proximity query
type PointProximityInput struct {
	Point       Vector3
	MaxDistance float64
	Filter      FilterInfo
	HitTriggers bool
	IgnoreBody  HP_BodyId
}

// ShapeCastInput describes a shape cast (sweep) query
type ShapeCastInput struct {
	Shape       HP_ShapeId
	Orientation Quaternion
	Start       Vector3
	End         Vector3
	HitTriggers bool
	IgnoreBody  HP_BodyId
}

// ShapeProximityInput describes a shape proximity query
type ShapeProximityInput struct {
	Shape       HP_ShapeId
	Position    Vector3
	Orientation Quaternion
	MaxDistance float64
	HitTriggers bool
	IgnoreBody  HP_BodyId
}
