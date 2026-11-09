// Package fn содержит ветвление без операторов if: условие выбирает значение,
// поэтому функции остаются в один-два оператора.
package fn

var bits = map[bool]int{false: 0, true: 1}

func Bit(ok bool) int {
	return bits[ok]
}

func Choose[T any](ok bool, yes, no T) T {
	return []T{no, yes}[bits[ok]]
}

func Lazy[T any](ok bool, compute func() T) T {
	return []func() T{empty[T], compute}[bits[ok]]()
}

func empty[T any]() T {
	var zero T
	return zero
}

func Maybe[T any](item T, ok bool) []T {
	return []T{item}[:bits[ok]]
}

func Do(ok bool, action func()) {
	if ok {
		action()
	}
}

func Or[T any](items []T, fallback T) T {
	return firstOf(items, fallback)
}

func firstOf[T any](items []T, fallback T) T {
	return takeFirst(fallback, items)
}

func takeFirst[T any](fallback T, items []T) T {
	return firstInto(fallback, items)
}

func firstInto[T any](picked T, items []T) T {
	for _, item := range items {
		picked = item
	}
	return picked
}

func Text(ok bool, value, fallback string) string {
	return Choose(ok, value, fallback)
}

func Number(ok bool, value, fallback int) int {
	return Choose(ok, value, fallback)
}

// With возвращает копию значения, изменённую функцией change.
func With[T any](value T, change func(*T)) T {
	return *changed(&value, change)
}

func changed[T any](value *T, change func(*T)) *T {
	change(value)
	return value
}

// Either выполняет ровно одну из ветвей: вычисление не происходит зря.
func Either[T any](ok bool, yes, no func() T) T {
	return []func() T{no, yes}[bits[ok]]()
}
