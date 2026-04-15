package otel

import "reflect"

func safeErrorString(err error) string {
	if isNilInterfaceValue(err) {
		return "<nil>"
	}

	return err.Error()
}

func isNilInterfaceValue(value any) bool {
	if value == nil {
		return true
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
