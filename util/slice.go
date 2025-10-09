package util

import (
	"slices"
)

func DeleteSorted[S ~[]E, E any](
	slice S,
	compare func(E, E) int,
	remove... E,
) S {
	for _, el := range remove {
		i, found := slices.BinarySearchFunc(slice, el, compare)
		if found {
			slice = slices.Delete(slice, i, i + 1)
		}
	}
	return slice
}

func InsertSorted[S ~[]E, E any](
	slice S,
	compare func(E, E) int,
	insert... E,
) S {
	for _, el := range insert {
		i, found := slices.BinarySearchFunc(slice, el, compare)
		if found {
			slice[i] = el
		} else {
			slice = slices.Insert(slice, i, el)
		}
	}
	return slice
}

func ReplaceSorted[S ~[]E, E any](
	slice S,
	compare func(E, E) int,
	old E,
	new E,
) S {
	i, foundOld := slices.BinarySearchFunc(slice, old, compare)
	j, foundNew := slices.BinarySearchFunc(slice, new, compare)
	switch {
	case !foundOld && foundNew:
		return slice
	case !foundOld:
		return slices.Insert(slice, j, new)
	case i == j:
		slice[j] = new
	case foundNew:
		return slices.Delete(slice, i, i + 1)
	case i < j:
		copy(slice[i:j - 1], slice[i + 1:])
		slice[j - 1] = new
	default:
		copy(slice[j + 1:i + 1], slice[j:])
		slice[j] = new
	}
	return slice
}

func ModifySorted[S ~[]E, E any](
	slice S,
	compare func(E, E) int,
	remove S,
	insert S,
) S {
	slice = DeleteSorted(slice, compare, remove...)
	slice = InsertSorted(slice, compare, insert...)
	return slice
}
