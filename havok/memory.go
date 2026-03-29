package havok

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/tetratelabs/wazero/api"
)

// memory helpers for reading and writing WASM linear memory.

// readI32 reads a little-endian int32 from WASM memory at offset.
func readI32(mem api.Memory, offset uint32) (int32, error) {
	b, ok := mem.Read(offset, 4)
	if !ok {
		return 0, fmt.Errorf("readI32: out of bounds at offset %d", offset)
	}
	return int32(binary.LittleEndian.Uint32(b)), nil
}

// readU32 reads a little-endian uint32 from WASM memory at offset.
func readU32(mem api.Memory, offset uint32) (uint32, error) {
	b, ok := mem.Read(offset, 4)
	if !ok {
		return 0, fmt.Errorf("readU32: out of bounds at offset %d", offset)
	}
	return binary.LittleEndian.Uint32(b), nil
}

// readI64 reads a little-endian int64 from WASM memory at offset.
func readI64(mem api.Memory, offset uint32) (int64, error) {
	b, ok := mem.Read(offset, 8)
	if !ok {
		return 0, fmt.Errorf("readI64: out of bounds at offset %d", offset)
	}
	return int64(binary.LittleEndian.Uint64(b)), nil
}

// readU64 reads a little-endian uint64 from WASM memory at offset.
func readU64(mem api.Memory, offset uint32) (uint64, error) {
	b, ok := mem.Read(offset, 8)
	if !ok {
		return 0, fmt.Errorf("readU64: out of bounds at offset %d", offset)
	}
	return binary.LittleEndian.Uint64(b), nil
}

// readF64 reads a float64 from WASM memory at offset.
func readF64(mem api.Memory, offset uint32) (float64, error) {
	b, ok := mem.Read(offset, 8)
	if !ok {
		return 0, fmt.Errorf("readF64: out of bounds at offset %d", offset)
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b)), nil
}

// writeU32 writes a little-endian uint32 to WASM memory at offset.
func writeU32(mem api.Memory, offset uint32, v uint32) error {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	if !mem.Write(offset, b) {
		return fmt.Errorf("writeU32: out of bounds at offset %d", offset)
	}
	return nil
}

// writeU64 writes a little-endian uint64 to WASM memory at offset.
func writeU64(mem api.Memory, offset uint32, v uint64) error {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	if !mem.Write(offset, b) {
		return fmt.Errorf("writeU64: out of bounds at offset %d", offset)
	}
	return nil
}

// writeF32 writes a float32 to WASM memory at offset.
func writeF32(mem api.Memory, offset uint32, v float32) error {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, math.Float32bits(v))
	if !mem.Write(offset, b) {
		return fmt.Errorf("writeF32: out of bounds at offset %d", offset)
	}
	return nil
}

// readF32 reads a float32 from WASM memory at offset.
func readF32(mem api.Memory, offset uint32) (float32, error) {
	b, ok := mem.Read(offset, 4)
	if !ok {
		return 0, fmt.Errorf("readF32: out of bounds at offset %d", offset)
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(b)), nil
}

// writeF64 writes a float64 to WASM memory at offset.
func writeF64(mem api.Memory, offset uint32, v float64) error {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, math.Float64bits(v))
	if !mem.Write(offset, b) {
		return fmt.Errorf("writeF64: out of bounds at offset %d", offset)
	}
	return nil
}

// writeVector3 writes a Vector3 to WASM memory at offset (3 × f32 = 12 bytes).
// Havok's internal float type is f32.
func writeVector3(mem api.Memory, offset uint32, v Vector3) error {
	for i := 0; i < 3; i++ {
		if err := writeF32(mem, offset+uint32(i*4), float32(v[i])); err != nil {
			return err
		}
	}
	return nil
}

// readVector3 reads a Vector3 from WASM memory at offset (3 × f32 = 12 bytes).
func readVector3(mem api.Memory, offset uint32) (Vector3, error) {
	var v Vector3
	for i := 0; i < 3; i++ {
		f, err := readF32(mem, offset+uint32(i*4))
		if err != nil {
			return v, err
		}
		v[i] = float64(f)
	}
	return v, nil
}

// writeQuaternion writes a Quaternion to WASM memory at offset (4 × f32 = 16 bytes).
func writeQuaternion(mem api.Memory, offset uint32, q Quaternion) error {
	for i := 0; i < 4; i++ {
		if err := writeF32(mem, offset+uint32(i*4), float32(q[i])); err != nil {
			return err
		}
	}
	return nil
}

// readQuaternion reads a Quaternion from WASM memory at offset (4 × f32 = 16 bytes).
func readQuaternion(mem api.Memory, offset uint32) (Quaternion, error) {
	var q Quaternion
	for i := 0; i < 4; i++ {
		f, err := readF32(mem, offset+uint32(i*4))
		if err != nil {
			return q, err
		}
		q[i] = float64(f)
	}
	return q, nil
}

// writeHP_Id writes a 64-bit handle (HP_BodyId / HP_ShapeId / etc.) to memory.
func writeHP_Id(mem api.Memory, offset uint32, id uint64) error {
	return writeU64(mem, offset, id)
}

// readHP_Id reads a 64-bit handle from memory.
func readHP_Id(mem api.Memory, offset uint32) (uint64, error) {
	return readU64(mem, offset)
}

// alignUp rounds n up to the nearest multiple of align.
func alignUp(n, align uint32) uint32 {
	return (n + align - 1) &^ (align - 1)
}
