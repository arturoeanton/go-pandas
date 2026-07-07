package index

import "fmt"

// Align computes the positional mapping needed to align two indexes on
// their label union. leftPos[i] (resp. rightPos[i]) is the position in the
// left (right) index of the i-th label of the result, or -1 when the label
// is absent from that side. When the indexes are equal the mapping is the
// identity.
func Align(left, right Index) (leftPos []int, rightPos []int, result Index, err error) {
	if left.Equals(right) {
		n := left.Len()
		leftPos = make([]int, n)
		rightPos = make([]int, n)
		for i := 0; i < n; i++ {
			leftPos[i] = i
			rightPos[i] = i
		}
		return leftPos, rightPos, left.Clone(), nil
	}
	result = Union(left, right)
	n := result.Len()
	leftPos = make([]int, n)
	rightPos = make([]int, n)
	for i := 0; i < n; i++ {
		label := result.At(i)
		if p, ok := left.Pos(label); ok {
			leftPos[i] = p
		} else {
			leftPos[i] = -1
		}
		if p, ok := right.Pos(label); ok {
			rightPos[i] = p
		} else {
			rightPos[i] = -1
		}
	}
	return leftPos, rightPos, result, nil
}

// Union returns the labels of left followed by the labels of right that
// are not present in left, preserving first-seen order.
func Union(left, right Index) Index {
	seen := make(map[any]bool)
	var values []any
	for i := 0; i < left.Len(); i++ {
		v := left.At(i)
		if !seen[keyable(v)] {
			seen[keyable(v)] = true
			values = append(values, v)
		}
	}
	for i := 0; i < right.Len(); i++ {
		v := right.At(i)
		if !seen[keyable(v)] {
			seen[keyable(v)] = true
			values = append(values, v)
		}
	}
	return fromValues(values, left.Name())
}

// Intersection returns the labels of left that also appear in right,
// preserving left order.
func Intersection(left, right Index) Index {
	inRight := make(map[any]bool)
	for i := 0; i < right.Len(); i++ {
		inRight[keyable(right.At(i))] = true
	}
	var values []any
	seen := make(map[any]bool)
	for i := 0; i < left.Len(); i++ {
		v := left.At(i)
		k := keyable(v)
		if inRight[k] && !seen[k] {
			seen[k] = true
			values = append(values, v)
		}
	}
	return fromValues(values, left.Name())
}

// Difference returns the labels of left that do not appear in right.
func Difference(left, right Index) Index {
	inRight := make(map[any]bool)
	for i := 0; i < right.Len(); i++ {
		inRight[keyable(right.At(i))] = true
	}
	var values []any
	seen := make(map[any]bool)
	for i := 0; i < left.Len(); i++ {
		v := left.At(i)
		k := keyable(v)
		if !inRight[k] && !seen[k] {
			seen[k] = true
			values = append(values, v)
		}
	}
	return fromValues(values, left.Name())
}

// keyable converts a label to a map-key-safe value (Tuple/[]any labels
// from MultiIndex are not comparable).
func keyable(v any) any {
	var tuple []any
	switch t := v.(type) {
	case []any:
		tuple = t
	case Tuple:
		tuple = t
	default:
		return v
	}
	s := ""
	for _, t := range tuple {
		s += "\x00" + fmt.Sprint(t)
	}
	return s
}
