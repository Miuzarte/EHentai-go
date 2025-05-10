package utils

type Stack[T any] []T

func (s *Stack[T]) Len() int {
	return len(*s)
}

func (s *Stack[T]) IsEmpty() bool {
	return s.Len() == 0
}

func (s *Stack[T]) Push(v T) {
	*s = append(*s, v)
}

func (s *Stack[T]) Pop() (v T, ok bool) {
	if s.IsEmpty() {
		return v, false
	}
	v = (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return v, true
}
