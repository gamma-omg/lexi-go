package fn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	t.Parallel()

	numbers := []int{1, 2, 3, 4, 5}
	squared := Map(numbers, func(n int) int {
		return n * n
	})

	expected := []int{1, 4, 9, 16, 25}
	assert.Equal(t, expected, squared)
}

func TestMap_EmptySlice(t *testing.T) {
	t.Parallel()

	numbers := []int{}
	squared := Map(numbers, func(n int) int {
		return n * n
	})

	var expected []int
	assert.Equal(t, expected, squared)
}

func TestMap_SingleElement(t *testing.T) {
	t.Parallel()

	numbers := []int{7}
	squared := Map(numbers, func(n int) int {
		return n * n
	})

	expected := []int{49}
	assert.Equal(t, expected, squared)
}

func TestMap_DifferentTypes(t *testing.T) {
	t.Parallel()

	strings := []string{"a", "bb", "ccc"}
	lengths := Map(strings, func(s string) int {
		return len(s)
	})

	expected := []int{1, 2, 3}
	assert.Equal(t, expected, lengths)
}

func TestMap_SameType(t *testing.T) {
	t.Parallel()

	strings := []string{"hello", "world"}
	uppercased := Map(strings, func(s string) string {
		return s + "!"
	})

	expected := []string{"hello!", "world!"}
	assert.Equal(t, expected, uppercased)
}
