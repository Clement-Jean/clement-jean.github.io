package utils

func Map[T, U any](ts []T, fn func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = fn(ts[i])
	}
	return us
}
