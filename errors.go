package mind

import (
	"fmt"
	"reflect"
)

type InvalidNameError struct{ fieldName string }

func (e InvalidNameError) Error() string {
	return fmt.Sprintf("could not found index for field name: %s", e.fieldName)
}

type InvalidValueTypeError[V any] struct{ value any }

func (e InvalidValueTypeError[V]) Error() string {
	return fmt.Sprintf("invalid index value type: %T, expected type: %v", e.value, reflect.TypeFor[V]())
}

type InvalidOperationError struct {
	indexName string
	op        Op
}

func (e InvalidOperationError) Error() string {
	return fmt.Sprintf("index: %q doesn't support the operation: %s", e.indexName, e.op)
}

type ValueNotFoundError struct{ value any }

func (e ValueNotFoundError) Error() string {
	return fmt.Sprintf("index value not found: %v", e.value)
}

type NoIdIndexDefinedError struct{}

func (e NoIdIndexDefinedError) Error() string {
	return fmt.Sprintln("no ID index defined")
}

type InvalidArgsLenError struct {
	defined string
	got     int
}

func (e InvalidArgsLenError) Error() string {
	return fmt.Sprintf("expected: %s values, got: %d", e.defined, e.got)
}
