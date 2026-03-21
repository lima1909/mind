package mind

import (
	"fmt"
	"math"
	"reflect"
	"unsafe"
)

// FromField is a function, which returns a value from an given object.
// example:
// Person{name string}
// func (p *Person) Name() { return p.name }
// (*Person).Name is the FieldGetFn
type FromField[OBJ any, V any] = func(*OBJ) V

// FromValue returns a Getter that simply returns the value itself.
// Use this when your list contains the raw values you want to index.
func FromValue[V any]() FromField[V, V] { return func(v *V) V { return *v } }

// FromName returns per reflection the propery (field) value from the given object.
func FromName[OBJ any, V any](fieldName string) FromField[OBJ, V] {
	var zero OBJ
	typ := reflect.TypeOf(zero)
	isPtr := false
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		isPtr = true
	}

	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("expected struct, got %s", typ.Kind()))
	}

	field, ok := typ.FieldByName(fieldName)
	if !ok {
		panic(fmt.Sprintf("field %s not found", fieldName))
	}
	// reflection cannot access lowercase (unexported) fields via .Interface()
	// unless we use unsafe, but let's stick to standard safety checks at setup time.
	// Actually, unsafe access works on unexported fields too, but usually discouraged.
	// But let's fail as per original behavior.
	if !field.IsExported() {
		panic(fmt.Sprintf("field %s is unexported", fieldName))
	}

	offset := field.Offset

	if isPtr {
		// OBJ is *Struct. input is **Struct.
		return func(obj *OBJ) V {
			// *obj is the *Struct.
			// We need unsafe.Pointer(*obj) + offset
			structPtr := *(**unsafe.Pointer)(unsafe.Pointer(obj))
			if structPtr == nil {
				var zero V
				return zero // Or panic? Original reflect would panic on nil pointer deref usually.
			}
			return *(*V)(unsafe.Add(*structPtr, offset))
		}
	}

	// OBJ is Struct. input is *Struct.
	return func(obj *OBJ) V {
		// obj is *Struct
		return *(*V)(unsafe.Add(unsafe.Pointer(obj), offset))
	}
}

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
