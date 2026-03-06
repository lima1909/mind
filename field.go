package mind

import (
	"fmt"
	"math"
)

func ValueFromAny[T any](value any) (T, error) {
	if v, ok := value.(T); ok {
		return v, nil
	}

	var zero T

	switch v := value.(type) {
	case int:
		return intValueFromAny[T](int64(v))
	case int64:
		return intValueFromAny[T](v)
	case float64:
		switch any(zero).(type) {
		case float32:
			if v < -math.MaxFloat32 || v > math.MaxFloat32 {
				return zero, OverflowError{toBigFor: "float32", value: value}
			}
			return any(float32(v)).(T), nil
		}

	}

	return zero, InvalidValueTypeError[T]{value}
}

func intValueFromAny[T any](v int64) (T, error) {
	var zero T

	switch any(zero).(type) {
	case int:
		if v < math.MinInt || v > math.MaxInt {
			return zero, OverflowError{toBigFor: "int", value: v}
		}
		return any(int(v)).(T), nil
	case int64:
		return any(int64(v)).(T), nil
	case int32:
		if v < math.MinInt32 || v > math.MaxInt32 {
			return zero, OverflowError{toBigFor: "int32", value: v}
		}
		return any(int32(v)).(T), nil
	case int16:
		if v < math.MinInt16 || v > math.MaxInt16 {
			return zero, OverflowError{toBigFor: "int16", value: v}
		}
		return any(int16(v)).(T), nil
	case int8:
		if v < math.MinInt8 || v > math.MaxInt8 {
			return zero, OverflowError{toBigFor: "int8", value: v}
		}
		return any(int8(v)).(T), nil
	case uint:
		if v < 0 {
			return zero, OverflowError{toBigFor: "uint", value: v}
		}
		return any(uint(v)).(T), nil
	case uint64:
		if v < 0 {
			return zero, OverflowError{toBigFor: "uint64", value: v}
		}
		return any(uint64(v)).(T), nil
	case uint32:
		if v < 0 || v > math.MaxUint32 {
			return zero, OverflowError{toBigFor: "uint32", value: v}
		}
		return any(uint32(v)).(T), nil
	case uint16:
		if v < 0 || v > math.MaxUint16 {
			return zero, OverflowError{toBigFor: "uint16", value: v}
		}
		return any(uint16(v)).(T), nil
	case uint8:
		if v < 0 || v > math.MaxUint8 {
			return zero, OverflowError{toBigFor: "uint8", value: v}
		}
		return any(uint8(v)).(T), nil
	}

	return zero, InvalidValueTypeError[T]{v}
}

type OverflowError struct {
	toBigFor string
	value    any
}

func (o OverflowError) Error() string {
	return fmt.Sprintf("to big for %q: %v", o.toBigFor, o.value)
}
