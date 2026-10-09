// Codec: byte-exact (de)serialisation for every Havok value struct.
//
// This file is NOT generated — it encodes the wire ABI of HavokPhysics.wasm,
// which was established empirically against @babylonjs/havok 1.3.14:
//
//   - All floats are f32 (Havok computes in single precision).
//   - Handles (HP_*Id) and bigints are 64-bit (i64).
//   - Enums, booleans and counts are 32-bit (i32).
//   - Aggregates are laid out with natural C++ alignment: f32/i32 align to 4,
//     i64 aligns to 8; the struct size is rounded up to its own alignment.
//
// The Size* constants are the number of bytes the caller must allocate for a
// buffer of that type (they are >= the native size where the native struct has
// trailing padding that does not affect field offsets).
package binding

import (
	"github.com/tetratelabs/wazero/api"
)

// -----------------------------------------------------------------------------
// Wire sizes (bytes)
// -----------------------------------------------------------------------------

const (
	SizeVector3              = 12
	SizeQuaternion           = 16
	SizeRotation             = 36
	SizeQTransform           = 28
	SizeQSTransform          = 40
	SizeTransform            = 48
	SizeAabb                 = 24
	SizeFilterInfo           = 8
	SizePhysicsMaterial      = 20
	SizeMassProperties       = 44
	SizeObjectStatistics     = 24
	SizeDebugGeometryInfo    = 16
	SizeShapePathIterator    = 16
	SizeContactPoint         = 64
	SizeRayCastResult        = 72
	SizePointProximityResult = 72
	SizeShapeCastResult      = 136
	SizeShapeProximityResult = 136
	SizeRayCastInput         = 48
	SizePointProximityInput  = 40
	SizeShapeCastInput       = 64
	SizeShapeProximityInput  = 56
	SizeBodyMovedEvent       = 48
	SizeCollisionEvent       = 144
	SizeTriggerEvent         = 40
)

// -----------------------------------------------------------------------------
// Scalar helpers
// -----------------------------------------------------------------------------

// WriteI32 writes a signed 32-bit integer.
func WriteI32(mem api.Memory, offset uint32, v int32) error {
	return WriteU32(mem, offset, uint32(v))
}

// WriteBool writes a boolean as a 32-bit integer (0/1).
func WriteBool(mem api.Memory, offset uint32, v bool) error {
	var u uint32
	if v {
		u = 1
	}
	return WriteU32(mem, offset, u)
}

// WriteF64AsF32 writes a float64 value truncated to f32.
func WriteF64AsF32(mem api.Memory, offset uint32, v float64) error {
	return WriteF32(mem, offset, float32(v))
}

// ReadU64AsHandle reads a 64-bit handle value.
func ReadU64AsHandle(mem api.Memory, offset uint32) (uint64, error) { return ReadU64(mem, offset) }

// ReadF32AsF64 reads an f32 and widens it to float64.
func ReadF32AsF64(mem api.Memory, offset uint32) (float64, error) {
	v, err := ReadF32(mem, offset)
	if err != nil {
		return 0, err
	}
	return float64(v), nil
}

// -----------------------------------------------------------------------------
// Vector3 / Quaternion are implemented in helpers.go.
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Rotation = [Vector3, Vector3, Vector3]  (36 bytes, three columns)
// -----------------------------------------------------------------------------

func WriteRotation(mem api.Memory, offset uint32, v Rotation) error {
	for i := 0; i < 3; i++ {
		if err := WriteVector3(mem, offset+uint32(i*12), v[i]); err != nil {
			return err
		}
	}
	return nil
}

func ReadRotation(mem api.Memory, offset uint32) (Rotation, error) {
	var r Rotation
	for i := 0; i < 3; i++ {
		col, err := ReadVector3(mem, offset+uint32(i*12))
		if err != nil {
			return r, err
		}
		r[i] = col
	}
	return r, nil
}

// -----------------------------------------------------------------------------
// QTransform = [Vector3, Quaternion]  (28 bytes)
// -----------------------------------------------------------------------------

func WriteQTransform(mem api.Memory, offset uint32, v QTransform) error {
	if err := WriteVector3(mem, offset, v.Translation); err != nil {
		return err
	}
	return WriteQuaternion(mem, offset+12, v.Rotation)
}

func ReadQTransform(mem api.Memory, offset uint32) (QTransform, error) {
	t, err := ReadVector3(mem, offset)
	if err != nil {
		return QTransform{}, err
	}
	r, err := ReadQuaternion(mem, offset+12)
	if err != nil {
		return QTransform{}, err
	}
	return QTransform{Translation: t, Rotation: r}, nil
}

// -----------------------------------------------------------------------------
// QSTransform = [Vector3, Quaternion, Vector3]  (40 bytes)
// -----------------------------------------------------------------------------

func WriteQSTransform(mem api.Memory, offset uint32, v QSTransform) error {
	if err := WriteVector3(mem, offset, v.Translation); err != nil {
		return err
	}
	if err := WriteQuaternion(mem, offset+12, v.Rotation); err != nil {
		return err
	}
	return WriteVector3(mem, offset+28, v.Scale)
}

func ReadQSTransform(mem api.Memory, offset uint32) (QSTransform, error) {
	t, err := ReadVector3(mem, offset)
	if err != nil {
		return QSTransform{}, err
	}
	r, err := ReadQuaternion(mem, offset+12)
	if err != nil {
		return QSTransform{}, err
	}
	s, err := ReadVector3(mem, offset+28)
	if err != nil {
		return QSTransform{}, err
	}
	return QSTransform{Translation: t, Rotation: r, Scale: s}, nil
}

// -----------------------------------------------------------------------------
// Transform = [Vector3, Rotation]  (48 bytes)
// -----------------------------------------------------------------------------

func WriteTransform(mem api.Memory, offset uint32, v Transform) error {
	if err := WriteVector3(mem, offset, v.Translation); err != nil {
		return err
	}
	return WriteRotation(mem, offset+12, v.Rotation)
}

func ReadTransform(mem api.Memory, offset uint32) (Transform, error) {
	t, err := ReadVector3(mem, offset)
	if err != nil {
		return Transform{}, err
	}
	r, err := ReadRotation(mem, offset+12)
	if err != nil {
		return Transform{}, err
	}
	return Transform{Translation: t, Rotation: r}, nil
}

// -----------------------------------------------------------------------------
// Aabb = [Vector3, Vector3]  (24 bytes)
// -----------------------------------------------------------------------------

func WriteAabb(mem api.Memory, offset uint32, v Aabb) error {
	if err := WriteVector3(mem, offset, v.Min); err != nil {
		return err
	}
	return WriteVector3(mem, offset+12, v.Max)
}

func ReadAabb(mem api.Memory, offset uint32) (Aabb, error) {
	mn, err := ReadVector3(mem, offset)
	if err != nil {
		return Aabb{}, err
	}
	mx, err := ReadVector3(mem, offset+12)
	if err != nil {
		return Aabb{}, err
	}
	return Aabb{Min: mn, Max: mx}, nil
}

// -----------------------------------------------------------------------------
// FilterInfo = [u32 membership, u32 collision]  (8 bytes)
// -----------------------------------------------------------------------------

func WriteFilterInfo(mem api.Memory, offset uint32, v FilterInfo) error {
	if err := WriteU32(mem, offset, v.MembershipMask); err != nil {
		return err
	}
	return WriteU32(mem, offset+4, v.CollisionMask)
}

func ReadFilterInfo(mem api.Memory, offset uint32) (FilterInfo, error) {
	m, err := ReadU32(mem, offset)
	if err != nil {
		return FilterInfo{}, err
	}
	c, err := ReadU32(mem, offset+4)
	if err != nil {
		return FilterInfo{}, err
	}
	return FilterInfo{MembershipMask: m, CollisionMask: c}, nil
}

// -----------------------------------------------------------------------------
// PhysicsMaterial = [f32, f32, f32, i32, i32]  (20 bytes)
// -----------------------------------------------------------------------------

func WritePhysicsMaterial(mem api.Memory, offset uint32, v PhysicsMaterial) error {
	if err := WriteF32(mem, offset, v.StaticFriction); err != nil {
		return err
	}
	if err := WriteF32(mem, offset+4, v.DynamicFriction); err != nil {
		return err
	}
	if err := WriteF32(mem, offset+8, v.Restitution); err != nil {
		return err
	}
	if err := WriteI32(mem, offset+12, int32(v.FrictionCombine)); err != nil {
		return err
	}
	return WriteI32(mem, offset+16, int32(v.RestitutionCombine))
}

func ReadPhysicsMaterial(mem api.Memory, offset uint32) (PhysicsMaterial, error) {
	sf, err := ReadF32(mem, offset)
	if err != nil {
		return PhysicsMaterial{}, err
	}
	df, err := ReadF32(mem, offset+4)
	if err != nil {
		return PhysicsMaterial{}, err
	}
	re, err := ReadF32(mem, offset+8)
	if err != nil {
		return PhysicsMaterial{}, err
	}
	fc, err := ReadI32(mem, offset+12)
	if err != nil {
		return PhysicsMaterial{}, err
	}
	rc, err := ReadI32(mem, offset+16)
	if err != nil {
		return PhysicsMaterial{}, err
	}
	return PhysicsMaterial{
		StaticFriction: sf, DynamicFriction: df, Restitution: re,
		FrictionCombine: MaterialCombine(fc), RestitutionCombine: MaterialCombine(rc),
	}, nil
}

// -----------------------------------------------------------------------------
// MassProperties = [Vector3, f32, Vector3, Quaternion]  (44 bytes)
// -----------------------------------------------------------------------------

func WriteMassProperties(mem api.Memory, offset uint32, v MassProperties) error {
	if err := WriteVector3(mem, offset, v.CenterOfMass); err != nil {
		return err
	}
	if err := WriteF32(mem, offset+12, float32(v.Mass)); err != nil {
		return err
	}
	if err := WriteVector3(mem, offset+16, v.InertiaTensorDiag); err != nil {
		return err
	}
	return WriteQuaternion(mem, offset+28, v.InertiaOrientation)
}

func ReadMassProperties(mem api.Memory, offset uint32) (MassProperties, error) {
	com, err := ReadVector3(mem, offset)
	if err != nil {
		return MassProperties{}, err
	}
	mass, err := ReadF32(mem, offset+12)
	if err != nil {
		return MassProperties{}, err
	}
	iner, err := ReadVector3(mem, offset+16)
	if err != nil {
		return MassProperties{}, err
	}
	ori, err := ReadQuaternion(mem, offset+28)
	if err != nil {
		return MassProperties{}, err
	}
	return MassProperties{CenterOfMass: com, Mass: float64(mass), InertiaTensorDiag: iner, InertiaOrientation: ori}, nil
}

// -----------------------------------------------------------------------------
// ObjectStatistics = [i32 x6]  (24 bytes)
// -----------------------------------------------------------------------------

func ReadObjectStatistics(mem api.Memory, offset uint32) (ObjectStatistics, error) {
	var s ObjectStatistics
	fields := []*int32{&s.NumBodies, &s.NumShapes, &s.NumConstraints, &s.NumDebugGeometries, &s.NumWorlds, &s.NumQueryCollectors}
	for i, f := range fields {
		v, err := ReadI32(mem, offset+uint32(i*4))
		if err != nil {
			return s, err
		}
		*f = v
	}
	return s, nil
}

// -----------------------------------------------------------------------------
// DebugGeometryInfo = [i32 x4]  (16 bytes)
// -----------------------------------------------------------------------------

func ReadDebugGeometryInfo(mem api.Memory, offset uint32) (DebugGeometryInfo, error) {
	var d DebugGeometryInfo
	fields := []*uint32{&d.VertexBufferPtr, &d.NumVertices, &d.TriangleBufferPtr, &d.NumTriangles}
	for i, f := range fields {
		v, err := ReadU32(mem, offset+uint32(i*4))
		if err != nil {
			return d, err
		}
		*f = v
	}
	return d, nil
}

// -----------------------------------------------------------------------------
// ShapePathIterator = [u64, u64]  (16 bytes)
// -----------------------------------------------------------------------------

func WriteShapePathIterator(mem api.Memory, offset uint32, v ShapePathIterator) error {
	if err := WriteU64(mem, offset, v.ShapeId); err != nil {
		return err
	}
	return WriteU64(mem, offset+8, v.PathData)
}

func ReadShapePathIterator(mem api.Memory, offset uint32) (ShapePathIterator, error) {
	id, err := ReadU64(mem, offset)
	if err != nil {
		return ShapePathIterator{}, err
	}
	pd, err := ReadU64(mem, offset+8)
	if err != nil {
		return ShapePathIterator{}, err
	}
	return ShapePathIterator{ShapeId: id, PathData: pd}, nil
}

// -----------------------------------------------------------------------------
// ContactPoint  (60 bytes of fields; allocate 64)
//
// Layout: u64 bodyId @0, u64 colliderId @8, ShapePathIterator @16,
//         Vector3 position @32, Vector3 normal @44, f32 triangleIndex @56.
// -----------------------------------------------------------------------------

func ReadContactPoint(mem api.Memory, offset uint32) (ContactPoint, error) {
	var cp ContactPoint
	body, err := ReadU64(mem, offset)
	if err != nil {
		return cp, err
	}
	coll, err := ReadU64(mem, offset+8)
	if err != nil {
		return cp, err
	}
	hier, err := ReadShapePathIterator(mem, offset+16)
	if err != nil {
		return cp, err
	}
	pos, err := ReadVector3(mem, offset+32)
	if err != nil {
		return cp, err
	}
	nrm, err := ReadVector3(mem, offset+44)
	if err != nil {
		return cp, err
	}
	tri, err := ReadF32(mem, offset+56)
	if err != nil {
		return cp, err
	}
	cp.BodyId = HP_BodyId{body}
	cp.ColliderId = HP_ShapeId{coll}
	cp.ShapeHierarchy = hier
	cp.Position = pos
	cp.Normal = nrm
	cp.TriangleIndex = float64(tri)
	return cp, nil
}

// -----------------------------------------------------------------------------
// Query results
// -----------------------------------------------------------------------------

// ReadRayCastResult reads [f32 fraction, ContactPoint].
func ReadRayCastResult(mem api.Memory, offset uint32) (RayCastResult, error) {
	frac, err := ReadF32(mem, offset)
	if err != nil {
		return RayCastResult{}, err
	}
	cp, err := ReadContactPoint(mem, offset+8)
	if err != nil {
		return RayCastResult{}, err
	}
	return RayCastResult{Fraction: float64(frac), Contact: cp}, nil
}

// ReadPointProximityResult reads [f32 distance, ContactPoint].
func ReadPointProximityResult(mem api.Memory, offset uint32) (PointProximityResult, error) {
	dist, err := ReadF32(mem, offset)
	if err != nil {
		return PointProximityResult{}, err
	}
	cp, err := ReadContactPoint(mem, offset+8)
	if err != nil {
		return PointProximityResult{}, err
	}
	return PointProximityResult{Distance: float64(dist), Contact: cp}, nil
}

// ReadShapeCastResult reads [f32 fraction, ContactPoint, ContactPoint].
func ReadShapeCastResult(mem api.Memory, offset uint32) (ShapeCastResult, error) {
	frac, err := ReadF32(mem, offset)
	if err != nil {
		return ShapeCastResult{}, err
	}
	a, err := ReadContactPoint(mem, offset+8)
	if err != nil {
		return ShapeCastResult{}, err
	}
	b, err := ReadContactPoint(mem, offset+72)
	if err != nil {
		return ShapeCastResult{}, err
	}
	return ShapeCastResult{Fraction: float64(frac), ContactOnInput: a, ContactOnTarget: b}, nil
}

// ReadShapeProximityResult reads [f32 distance, ContactPoint, ContactPoint].
func ReadShapeProximityResult(mem api.Memory, offset uint32) (ShapeProximityResult, error) {
	dist, err := ReadF32(mem, offset)
	if err != nil {
		return ShapeProximityResult{}, err
	}
	a, err := ReadContactPoint(mem, offset+8)
	if err != nil {
		return ShapeProximityResult{}, err
	}
	b, err := ReadContactPoint(mem, offset+72)
	if err != nil {
		return ShapeProximityResult{}, err
	}
	return ShapeProximityResult{Distance: float64(dist), ContactOnInput: a, ContactOnTarget: b}, nil
}

// -----------------------------------------------------------------------------
// Query inputs
// -----------------------------------------------------------------------------

// WriteRayCastInput writes [Vector3 start, Vector3 end, FilterInfo, bool, u64 ignoreBody] (48 bytes).
func WriteRayCastInput(mem api.Memory, offset uint32, v RayCastInput) error {
	if err := WriteVector3(mem, offset, v.Start); err != nil {
		return err
	}
	if err := WriteVector3(mem, offset+12, v.End); err != nil {
		return err
	}
	if err := WriteFilterInfo(mem, offset+24, v.Filter); err != nil {
		return err
	}
	if err := WriteBool(mem, offset+32, v.HitTriggers); err != nil {
		return err
	}
	return WriteU64(mem, offset+40, v.IgnoreBody[0])
}

// WritePointProximityInput writes [Vector3 point, f32 maxDistance, FilterInfo, bool, u64] (40 bytes).
func WritePointProximityInput(mem api.Memory, offset uint32, v PointProximityInput) error {
	if err := WriteVector3(mem, offset, v.Point); err != nil {
		return err
	}
	if err := WriteF32(mem, offset+12, float32(v.MaxDistance)); err != nil {
		return err
	}
	if err := WriteFilterInfo(mem, offset+16, v.Filter); err != nil {
		return err
	}
	if err := WriteBool(mem, offset+24, v.HitTriggers); err != nil {
		return err
	}
	return WriteU64(mem, offset+32, v.IgnoreBody[0])
}

// WriteShapeCastInput writes [u64 shape, Quaternion, Vector3 start, Vector3 end, bool, u64] (64 bytes).
func WriteShapeCastInput(mem api.Memory, offset uint32, v ShapeCastInput) error {
	if err := WriteU64(mem, offset, v.Shape[0]); err != nil {
		return err
	}
	if err := WriteQuaternion(mem, offset+8, v.Orientation); err != nil {
		return err
	}
	if err := WriteVector3(mem, offset+24, v.Start); err != nil {
		return err
	}
	if err := WriteVector3(mem, offset+36, v.End); err != nil {
		return err
	}
	if err := WriteBool(mem, offset+48, v.HitTriggers); err != nil {
		return err
	}
	return WriteU64(mem, offset+56, v.IgnoreBody[0])
}

// WriteShapeProximityInput writes [u64 shape, Vector3, Quaternion, f32, bool, u64] (56 bytes).
func WriteShapeProximityInput(mem api.Memory, offset uint32, v ShapeProximityInput) error {
	if err := WriteU64(mem, offset, v.Shape[0]); err != nil {
		return err
	}
	if err := WriteVector3(mem, offset+8, v.Position); err != nil {
		return err
	}
	if err := WriteQuaternion(mem, offset+20, v.Orientation); err != nil {
		return err
	}
	if err := WriteF32(mem, offset+36, float32(v.MaxDistance)); err != nil {
		return err
	}
	if err := WriteBool(mem, offset+40, v.HitTriggers); err != nil {
		return err
	}
	return WriteU64(mem, offset+48, v.IgnoreBody[0])
}

// -----------------------------------------------------------------------------
// Events
// -----------------------------------------------------------------------------

// ReadBodyMovedEvent reads [i32 type, u64 body, QTransform] (48 bytes).
func ReadBodyMovedEvent(mem api.Memory, offset uint32) (BodyMovedEvent, error) {
	var ev BodyMovedEvent
	t, err := ReadI32(mem, offset)
	if err != nil {
		return ev, err
	}
	body, err := ReadU64(mem, offset+8)
	if err != nil {
		return ev, err
	}
	xf, err := ReadQTransform(mem, offset+16)
	if err != nil {
		return ev, err
	}
	ev.Type = EventType(t)
	ev.Body = HP_BodyId{body}
	ev.WorldFromBody = xf
	return ev, nil
}

// ReadCollisionEvent reads [i32 type, ContactPoint, ContactPoint, f32 impulse] (144 bytes).
func ReadCollisionEvent(mem api.Memory, offset uint32) (CollisionEvent, error) {
	var ev CollisionEvent
	t, err := ReadI32(mem, offset)
	if err != nil {
		return ev, err
	}
	a, err := ReadContactPoint(mem, offset+8)
	if err != nil {
		return ev, err
	}
	b, err := ReadContactPoint(mem, offset+72)
	if err != nil {
		return ev, err
	}
	imp, err := ReadF32(mem, offset+136)
	if err != nil {
		return ev, err
	}
	ev.Type = EventType(t)
	ev.ContactA = a
	ev.ContactB = b
	ev.Impulse = float64(imp)
	return ev, nil
}

// ReadTriggerEvent reads [i32 type, u64 bodyA, u64 shapeA, u64 bodyB, u64 shapeB] (40 bytes).
func ReadTriggerEvent(mem api.Memory, offset uint32) (TriggerEvent, error) {
	var ev TriggerEvent
	t, err := ReadI32(mem, offset)
	if err != nil {
		return ev, err
	}
	ba, err := ReadU64(mem, offset+8)
	if err != nil {
		return ev, err
	}
	sa, err := ReadU64(mem, offset+16)
	if err != nil {
		return ev, err
	}
	bb, err := ReadU64(mem, offset+24)
	if err != nil {
		return ev, err
	}
	sb, err := ReadU64(mem, offset+32)
	if err != nil {
		return ev, err
	}
	ev.Type = EventType(t)
	ev.BodyA = HP_BodyId{ba}
	ev.ShapeA = HP_ShapeId{sa}
	ev.BodyB = HP_BodyId{bb}
	ev.ShapeB = HP_ShapeId{sb}
	return ev, nil
}
