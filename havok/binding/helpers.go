// Exported WASM memory helpers used by generated/bindings_gen.go.
// This file is NOT generated — it is hand-maintained.
package binding

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/tetratelabs/wazero/api"
)

func ReadI32(mem api.Memory, offset uint32) (int32, error) {
	b, ok := mem.Read(offset, 4)
	if !ok {
		return 0, fmt.Errorf("ReadI32: out of bounds at offset %d", offset)
	}
	return int32(binary.LittleEndian.Uint32(b)), nil
}

func ReadU32(mem api.Memory, offset uint32) (uint32, error) {
	b, ok := mem.Read(offset, 4)
	if !ok {
		return 0, fmt.Errorf("ReadU32: out of bounds at offset %d", offset)
	}
	return binary.LittleEndian.Uint32(b), nil
}

func ReadU64(mem api.Memory, offset uint32) (uint64, error) {
	b, ok := mem.Read(offset, 8)
	if !ok {
		return 0, fmt.Errorf("ReadU64: out of bounds at offset %d", offset)
	}
	return binary.LittleEndian.Uint64(b), nil
}

func ReadF32(mem api.Memory, offset uint32) (float32, error) {
	b, ok := mem.Read(offset, 4)
	if !ok {
		return 0, fmt.Errorf("ReadF32: out of bounds at offset %d", offset)
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(b)), nil
}

func WriteU32(mem api.Memory, offset uint32, v uint32) error {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	if !mem.Write(offset, b) {
		return fmt.Errorf("WriteU32: out of bounds at offset %d", offset)
	}
	return nil
}

func WriteF32(mem api.Memory, offset uint32, v float32) error {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, math.Float32bits(v))
	if !mem.Write(offset, b) {
		return fmt.Errorf("WriteF32: out of bounds at offset %d", offset)
	}
	return nil
}

func WriteU64(mem api.Memory, offset uint32, v uint64) error {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	if !mem.Write(offset, b) {
		return fmt.Errorf("WriteU64: out of bounds at offset %d", offset)
	}
	return nil
}

func WriteVector3(mem api.Memory, offset uint32, v Vector3) error {
	for i := 0; i < 3; i++ {
		if err := WriteF32(mem, offset+uint32(i*4), float32(v[i])); err != nil {
			return err
		}
	}
	return nil
}

func ReadVector3(mem api.Memory, offset uint32) (Vector3, error) {
	var v Vector3
	for i := 0; i < 3; i++ {
		f, err := ReadF32(mem, offset+uint32(i*4))
		if err != nil {
			return v, err
		}
		v[i] = float64(f)
	}
	return v, nil
}

func WriteQuaternion(mem api.Memory, offset uint32, q Quaternion) error {
	for i := 0; i < 4; i++ {
		if err := WriteF32(mem, offset+uint32(i*4), float32(q[i])); err != nil {
			return err
		}
	}
	return nil
}

func ReadQuaternion(mem api.Memory, offset uint32) (Quaternion, error) {
	var q Quaternion
	for i := 0; i < 4; i++ {
		f, err := ReadF32(mem, offset+uint32(i*4))
		if err != nil {
			return q, err
		}
		q[i] = float64(f)
	}
	return q, nil
}
