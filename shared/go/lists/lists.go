package lists

import "github.com/scc-platform/shared/fn"

func Map[T any, R any](items []T, transform func(T) R) []R {
	return fillInto(make([]R, 0, len(items)), items, transform)
}

func fillInto[T any, R any](out []R, items []T, transform func(T) R) []R {
	for _, item := range items {
		out = append(out, transform(item))
	}
	return out
}

func Filter[T any](items []T, keep func(T) bool) []T {
	return keepInto(make([]T, 0, len(items)), items, keep)
}

func keepInto[T any](out []T, items []T, keep func(T) bool) []T {
	for _, item := range items {
		out = append(out, fn.Maybe(item, keep(item))...)
	}
	return out
}

func GroupBy[T any, K comparable](items []T, key func(T) K) map[K][]T {
	return groupInto(make(map[K][]T, len(items)), items, key)
}

func groupInto[T any, K comparable](groups map[K][]T, items []T, key func(T) K) map[K][]T {
	for _, item := range items {
		groups[key(item)] = append(groups[key(item)], item)
	}
	return groups
}

func Sum[T any](items []T, weight func(T) float64) float64 {
	return sumInto(0, items, weight)
}

func sumInto[T any](total float64, items []T, weight func(T) float64) float64 {
	for _, item := range items {
		total += weight(item)
	}
	return total
}

func Find[T any](items []T, match func(T) bool) (T, bool) {
	matches := Filter(items, match)
	return head(matches), len(matches) > 0
}

func head[T any](items []T) T {
	return fn.Or(items[:min(1, len(items))], Zero[T]())
}

func Zero[T any]() T {
	var zero T
	return zero
}

func Clamp(value, low, high float64) float64 {
	return min(max(value, low), high)
}

func Keys[K comparable, V any](groups map[K]V) []K {
	return keysInto(make([]K, 0, len(groups)), groups)
}

func keysInto[K comparable, V any](keys []K, groups map[K]V) []K {
	for key := range groups {
		keys = append(keys, key)
	}
	return keys
}

func Contains[T comparable](items []T, target T) bool {
	_, found := Find(items, func(item T) bool { return item == target })
	return found
}
