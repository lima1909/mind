package mind

import (
	"fmt"
	"reflect"
)

type InvalidNameError struct{ FieldName string }

func (e InvalidNameError) Error() string {
	return fmt.Sprintf("could not found index for field name: %s", e.FieldName)
}

type InvalidValueTypeError[V any] struct{ Value any }

func (e InvalidValueTypeError[V]) Error() string {
	return fmt.Sprintf("invalid index value type: %T, expected type: %v", e.Value, reflect.TypeFor[V]())
}

type InvalidOperationError struct {
	IndexName string
	Op        Op
}

func (e InvalidOperationError) Error() string {
	return fmt.Sprintf("index: %q doesn't support the operation: %s", e.IndexName, e.Op)
}

type ValueNotFoundError struct{ Value any }

func (e ValueNotFoundError) Error() string {
	return fmt.Sprintf("index value not found: %v", e.Value)
}

type NoIdIndexDefinedError struct{}

func (e NoIdIndexDefinedError) Error() string {
	return fmt.Sprintln("no ID index defined")
}

type InvalidArgsLenError struct {
	Defined string
	Got     int
}

func (e InvalidArgsLenError) Error() string {
	return fmt.Sprintf("expected: %s values, got: %d", e.Defined, e.Got)
}
