package havok

import "github.com/gsw945/havok-go/havok/binding"

// Re-export all types from the binding sub-package for backwards compatibility.
// Using type aliases (=) preserves full assignability between havok.X and generated.X.
type (
	Vector3    = binding.Vector3
	Quaternion = binding.Quaternion

	HP_BodyId          = binding.HP_BodyId
	HP_ShapeId         = binding.HP_ShapeId
	HP_WorldId         = binding.HP_WorldId
	HP_ConstraintId    = binding.HP_ConstraintId
	HP_CollectorId     = binding.HP_CollectorId
	HP_DebugGeometryId = binding.HP_DebugGeometryId

	Result                  = binding.Result
	MotionType              = binding.MotionType
	ShapeType               = binding.ShapeType
	ActivationState         = binding.ActivationState
	ActivationControl       = binding.ActivationControl
	ConstraintAxis          = binding.ConstraintAxis
	MaterialCombine         = binding.MaterialCombine
	EventType               = binding.EventType
	ConstraintAxisLimitMode = binding.ConstraintAxisLimitMode
	ConstraintMotorType     = binding.ConstraintMotorType

	ObjectStatistics     = binding.ObjectStatistics
	FilterInfo           = binding.FilterInfo
	PhysicsMaterial      = binding.PhysicsMaterial
	MassProperties       = binding.MassProperties
	QTransform           = binding.QTransform
	QSTransform          = binding.QSTransform
	Aabb                 = binding.Aabb
	ShapePathIterator    = binding.ShapePathIterator
	ContactPoint         = binding.ContactPoint
	DebugGeometryInfo    = binding.DebugGeometryInfo
	CollisionEvent       = binding.CollisionEvent
	TriggerEvent         = binding.TriggerEvent
	BodyMovedEvent       = binding.BodyMovedEvent
	RayCastResult        = binding.RayCastResult
	PointProximityResult = binding.PointProximityResult
	ShapeCastResult      = binding.ShapeCastResult
	ShapeProximityResult = binding.ShapeProximityResult

	RayCastInput        = binding.RayCastInput
	PointProximityInput = binding.PointProximityInput
	ShapeCastInput      = binding.ShapeCastInput
	ShapeProximityInput = binding.ShapeProximityInput
)

// Re-export all enum constants.
// Type aliases do not automatically re-export constants, so we forward them explicitly.
const (
	Result_OK             = binding.Result_OK
	Result_FAIL           = binding.Result_FAIL
	Result_INVALIDHANDLE  = binding.Result_INVALIDHANDLE
	Result_INVALIDARGS    = binding.Result_INVALIDARGS
	Result_NOTIMPLEMENTED = binding.Result_NOTIMPLEMENTED

	MotionType_STATIC    = binding.MotionType_STATIC
	MotionType_KINEMATIC = binding.MotionType_KINEMATIC
	MotionType_DYNAMIC   = binding.MotionType_DYNAMIC

	ShapeType_COLLIDER  = binding.ShapeType_COLLIDER
	ShapeType_CONTAINER = binding.ShapeType_CONTAINER

	ActivationState_ACTIVE   = binding.ActivationState_ACTIVE
	ActivationState_INACTIVE = binding.ActivationState_INACTIVE

	ActivationControl_SIMULATION_CONTROLLED = binding.ActivationControl_SIMULATION_CONTROLLED
	ActivationControl_ALWAYS_ACTIVE         = binding.ActivationControl_ALWAYS_ACTIVE
	ActivationControl_ALWAYS_INACTIVE       = binding.ActivationControl_ALWAYS_INACTIVE

	ConstraintAxis_LINEAR_X        = binding.ConstraintAxis_LINEAR_X
	ConstraintAxis_LINEAR_Y        = binding.ConstraintAxis_LINEAR_Y
	ConstraintAxis_LINEAR_Z        = binding.ConstraintAxis_LINEAR_Z
	ConstraintAxis_ANGULAR_X       = binding.ConstraintAxis_ANGULAR_X
	ConstraintAxis_ANGULAR_Y       = binding.ConstraintAxis_ANGULAR_Y
	ConstraintAxis_ANGULAR_Z       = binding.ConstraintAxis_ANGULAR_Z
	ConstraintAxis_LINEAR_DISTANCE = binding.ConstraintAxis_LINEAR_DISTANCE

	MaterialCombine_GEOMETRIC_MEAN  = binding.MaterialCombine_GEOMETRIC_MEAN
	MaterialCombine_MINIMUM         = binding.MaterialCombine_MINIMUM
	MaterialCombine_MAXIMUM         = binding.MaterialCombine_MAXIMUM
	MaterialCombine_ARITHMETIC_MEAN = binding.MaterialCombine_ARITHMETIC_MEAN
	MaterialCombine_MULTIPLY        = binding.MaterialCombine_MULTIPLY

	EventType_COLLISION_STARTED   = binding.EventType_COLLISION_STARTED
	EventType_COLLISION_CONTINUED = binding.EventType_COLLISION_CONTINUED
	EventType_COLLISION_FINISHED  = binding.EventType_COLLISION_FINISHED
	EventType_TRIGGER_ENTERED     = binding.EventType_TRIGGER_ENTERED
	EventType_TRIGGER_EXITED      = binding.EventType_TRIGGER_EXITED

	ConstraintAxisLimitMode_FREE    = binding.ConstraintAxisLimitMode_FREE
	ConstraintAxisLimitMode_LIMITED = binding.ConstraintAxisLimitMode_LIMITED
	ConstraintAxisLimitMode_LOCKED  = binding.ConstraintAxisLimitMode_LOCKED

	ConstraintMotorType_NONE                = binding.ConstraintMotorType_NONE
	ConstraintMotorType_VELOCITY            = binding.ConstraintMotorType_VELOCITY
	ConstraintMotorType_POSITION            = binding.ConstraintMotorType_POSITION
	ConstraintMotorType_SPRING_FORCE        = binding.ConstraintMotorType_SPRING_FORCE
	ConstraintMotorType_SPRING_ACCELERATION = binding.ConstraintMotorType_SPRING_ACCELERATION
)
