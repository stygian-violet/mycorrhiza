package util

import (
	"iter"
)

func Map[T any, U any](fn func(T) U, seq iter.Seq[T]) iter.Seq[U] {
	return func(yield func(U) bool) {
		for val := range seq {
			if !yield(fn(val)) {
				return
			}
		}
	}
}

func Filter[T any](cond func(T) bool, seq iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for val := range seq {
			if cond(val) {
				if !yield(val) {
					return
				}
			}
		}
	}
}
