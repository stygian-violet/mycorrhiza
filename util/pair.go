package util

type RenamingPair[T any] struct {
	from T
	to   T
}

func NewRenamingPair[T any](from T, to T) RenamingPair[T] {
	return RenamingPair[T]{
		from: from,
		to: to,
	}
}

func (rp RenamingPair[T]) From() T {
	return rp.from
}

func (rp RenamingPair[T]) To() T {
	return rp.to
}
