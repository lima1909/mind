package main

import (
	"fmt"
	"math"
	"strconv"
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

	return zero, ErrInvalidIndexValue[T]{value}
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

	return zero, ErrInvalidIndexValue[T]{v}
}

func ParseBool(s string) (bool, error) {
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

	return false, InvalidDataTypeError{expected: OpBool, got: s}
}

func ParseNumber(s string) (float64, error) {
	n := len(s)
	if n == 0 {
		return 0.0, InvalidDataTypeError{expected: OpNumber, got: s}
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
				return 0, InvalidDataTypeError{
					expected: OpNumber,
					got:      s,
					hint:     "more than one dot is not allowed",
				}
			}
			dot = i
			// } else if c == 'e' || c == 'E' {
			// 	// Fallback for scientific notation
			// 	return strconv.ParseFloat(s, 64)
		} else {
			return 0, InvalidDataTypeError{expected: OpNumber, got: s}
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

type InvalidDataTypeError struct {
	expected Op
	got      string
	hint     string
}

func (i InvalidDataTypeError) Error() string {
	if len(i.hint) == 0 {
		return fmt.Sprintf("invalid datatype, expected: %q, got value: %q", i.expected, i.got)
	}
	return fmt.Sprintf("invalid datatype, expected: %q, got value: %q (%s)", i.expected, i.got, i.hint)
}

type OverflowError struct {
	toBigFor string
	value    any
}

func (o OverflowError) Error() string {
	return fmt.Sprintf("to big for %q: %v", o.toBigFor, o.value)
}
