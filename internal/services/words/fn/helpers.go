package fn

func Map[T any, V any](items []T, selector func(T) V) []V {
	var results []V
	for _, item := range items {
		results = append(results, selector(item))
	}
	return results
}
