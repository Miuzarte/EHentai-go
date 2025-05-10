package utils

type Set[T comparable] map[T]struct{}

func (set *Set[T]) Add(v ...T) {
	for _, val := range v {
		(*set)[val] = struct{}{}
	}
}

func (set *Set[T]) Delete(v ...T) {
	for _, val := range v {
		delete(*set, val)
	}
}

// Get 不保证顺序
func (set *Set[T]) Get() (s []T) {
	s = make([]T, len(*set))
	i := 0
	for val := range *set {
		s[i] = val
		i++
	}
	return
}

func (set *Set[T]) Ok(v T) bool {
	_, ok := (*set)[v]
	return ok
}

// Clean 能保证顺序
func (set *Set[T]) Clean(s []T) []T {
	write := 0
	for read := range s {
		if !set.Ok(s[read]) {
			(*set)[s[read]] = struct{}{}
			s[write] = s[read]
			write++
		}
	}
	return s[:write]
}
