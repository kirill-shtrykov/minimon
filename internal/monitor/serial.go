package monitor

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

func IntToBytes(i int) ([]byte, error) {
	if i < 0 {
		return nil, fmt.Errorf("%w: %d", ErrIntegerOutOfRange, i)
	}

	num := int64(i)
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.LittleEndian, num); err != nil {
		return nil, fmt.Errorf("failed to convert integer to bytes: %w", err)
	}

	return buf.Bytes(), nil
}

func Float64ToBytes(f float64) ([]byte, error) {
	bits := math.Float64bits(f)
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.LittleEndian, bits); err != nil {
		return nil, fmt.Errorf("failed to convert float to bytes: %w", err)
	}

	return buf.Bytes(), nil
}

func BytesToInt(b []byte) (int, error) {
	u := binary.LittleEndian.Uint64(b)
	if u > math.MaxInt {
		return 0, fmt.Errorf("%w: %d", ErrIntegerOutOfRange, u)
	}

	return int(u), nil
}

func BytesToFloat64(b []byte) (float64, error) {
	bits := binary.LittleEndian.Uint64(b)
	f := math.Float64frombits(bits)

	return f, nil
}
