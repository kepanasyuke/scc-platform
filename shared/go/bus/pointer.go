package bus

import "reflect"

func pointerOf(handler Handler) uintptr {
	return reflect.ValueOf(handler).Pointer()
}
