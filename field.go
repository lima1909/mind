package main

import (
	"fmt"
	"math"
	"strconv"
)

// FieldValues is an iterator over the string field values and convert it to the disired value
type FieldValues[T any] struct {
	pos    int
	values []string
}

func NewFieldValues[T any](values ...string) *FieldValues[T] {
	return &FieldValues[T]{values: values}
}

func (f *FieldValues[T]) First() (T, bool, error) {
	if len(f.values) == 0 {
		var zero T
		return zero, false, nil
	}

	v, err := ValueFromString[T](f.values[0])
	if err != nil {
		var zero T
		return zero, false, err
	}

	return v, true, nil
}

func (f *FieldValues[T]) Next() (T, bool, error) {
	if f.pos >= len(f.values) {
		var zero T
		return zero, false, nil
	}

	v, err := ValueFromString[T](f.values[f.pos])
	f.pos++
	if err != nil {
		var zero T
		return zero, false, err
	}

	return v, true, nil
}

func ValueFromString[T any](strValue string) (T, error) {
	var zero T

	switch any(zero).(type) {
	case string:
		return any(strValue).(T), nil
	case *string:
		return any(&strValue).(T), nil
	case bool:
		b, err := parseBool(strValue)
		if err != nil {
			return zero, err
		}
		return any(b).(T), nil
	case *bool:
		b, err := parseBool(strValue)
		if err != nil {
			return zero, err
		}
		return any(&b).(T), nil
	case float32:
		f, err := parseNumber(strValue)
		if err != nil {
			return zero, err
		}
		if f > -math.MaxFloat32 && f < math.MaxFloat32 {
			return any(float32(f)).(T), nil
		}
		return zero, fmt.Errorf("to big for float32: %v", f)
	case float64:
		f, err := parseNumber(strValue)
		if err != nil {
			return zero, err
		}
		return any(float64(f)).(T), nil
	case int:
		f, err := parseNumber(strValue)
		if err != nil {
			return zero, err
		}
		if f > math.MinInt && f < math.MaxInt {
			return any(int(f)).(T), nil
		}
		return zero, fmt.Errorf("to big for int: %v", f)
	case uint:
		f, err := parseNumber(strValue)
		if err != nil {
			return zero, err
		}
		if f >= 0 && f <= math.MaxUint {
			return any(uint(f)).(T), nil
		}
		return zero, fmt.Errorf("to big for uint: %v", f)
	case uint8:
		f, err := parseNumber(strValue)
		if err != nil {
			return zero, err
		}
		if f >= 0 && f <= math.MaxUint8 {
			return any(uint8(f)).(T), nil
		}
		return zero, fmt.Errorf("to big for uint8: %v", f)
		// int, int8, int16, int32, int64,
		// uint, uint8, uint16, uint32, uint64:
	}

	return zero, fmt.Errorf("not supported type: %T", zero)
}

// vt, _ := GetFieldType[T]()
// if expectedValueType != vt {
// 	return zero, fmt.Errorf("expected type: %d, got: %d", expectedValueType, vt)
// }

// func GetFieldType[T any]() (ValueType, ValueType) {
// 	var zero T
// 	switch any(zero).(type) {
// 	case string, *string:
// 		return TextValue, TextValue
// 	case float32, float64,
// 		int, int8, int16, int32, int64,
// 		uint, uint8, uint16, uint32, uint64:
// 		return NumberValue, NumberValue
// 	case *float32, *float64,
// 		*int, *int8, *int16, *int32, *int64,
// 		*uint, *uint8, *uint16, *uint32, *uint64:
// 		return NumberValue, NumberValue
// 	case bool, *bool:
// 		return BoolValue, BoolValue
// 	case []string, []*string:
// 		return ListValue, TextValue
// 	case []float32, []float64,
// 		[]int, []int8, []int16, []int32, []int64,
// 		[]uint, []uint8, []uint16, []uint32, []uint64:
// 		return ListValue, NumberValue
// 	case []*float32, []*float64,
// 		[]*int, []*int8, []*int16, []*int32, []*int64,
// 		[]*uint, []*uint8, []*uint16, []*uint32, []*uint64:
// 		return ListValue, NumberValue
// 	case []bool:
// 		return ListValue, BoolValue
// 	default:
// 		return NotSupportedValueType, NotSupportedValueType
// 	}
// }

func parseBool(s string) (bool, error) {
	switch len(s) {
	case 4:
		if (s[0] == 't' || s[0] == 'T') &&
			(s[1] == 'r' || s[1] == 'R') &&
			(s[2] == 'u' || s[2] == 'U') &&
			(s[3] == 'e' || s[3] == 'E') {
			return true, nil
		}
	case 5:
		if (s[0] == 'f' || s[0] == 'F') &&
			(s[1] == 'a' || s[1] == 'A') &&
			(s[2] == 'l' || s[2] == 'L') &&
			(s[3] == 's' || s[3] == 'S') &&
			(s[4] == 'e' || s[4] == 'E') {
			return false, nil
		}
	}

	return false, fmt.Errorf("is not a bool: %s", s)
}

func parseNumber(s string) (float64, error) {
	n := len(s)
	if n == 0 {
		return 0.0, fmt.Errorf("empty string is not a number")
	}

	var (
		i      int
		neg    bool
		absVal uint64
		exp    int
		dot    = -1
		digits int // Track digit count for precision safety
	)

	if i < n && s[i] == '-' {
		neg = true
		i++
	} else if i < n && s[i] == '+' {
		i++
	}

	for ; i < n; i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			// Optimization: Stay in uint64 as long as we have precision
			if digits < 19 {
				absVal = absVal*10 + uint64(c-'0')
				if dot != -1 {
					exp--
				}
				digits++
			} else {
				// Too many digits for simple uint64, fallback to stdlib
				f, err := strconv.ParseFloat(s, 64)
				return f, err
			}
		} else if c == '.' {
			if dot != -1 {
				return 0, fmt.Errorf("more than one dot is not allowed")
			}
			dot = i
			// } else if c == 'e' || c == 'E' {
			// 	// Fallback for scientific notation
			// 	return strconv.ParseFloat(s, 64)
		} else {
			return 0, fmt.Errorf("this is not a valid number char: %c", c)
		}
	}

	// apply the exponent using the extended pow10 table
	res := float64(absVal)
	if exp < 0 {
		if -exp < len(pow10_64) {
			res /= pow10_64[-exp]
		} else {
			return strconv.ParseFloat(s, 64)
		}
	} else if exp > 0 {
		if exp < len(pow10_64) {
			res *= pow10_64[exp]
		} else {
			return strconv.ParseFloat(s, 64)
		}
	}

	if neg {
		res = -res
	}

	return res, nil
}

// Expanded table for float64
var pow10_64 = []float64{
	1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10,
	1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19, 1e20,
}
