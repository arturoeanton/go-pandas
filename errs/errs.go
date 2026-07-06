// Package errs defines the sentinel errors shared by every go-pandas
// package. The root package re-exports them so users can match with
// errors.Is(err, pd.ErrColumnNotFound).
package errs

import (
	"errors"
	"fmt"
)

var (
	ErrColumnNotFound     = errors.New("column not found")
	ErrIndexOutOfBounds   = errors.New("index out of bounds")
	ErrLengthMismatch     = errors.New("length mismatch")
	ErrTypeMismatch       = errors.New("type mismatch")
	ErrShapeMismatch      = errors.New("shape mismatch")
	ErrBroadcastMismatch  = errors.New("broadcast shape mismatch")
	ErrInvalidOperation   = errors.New("invalid operation")
	ErrInvalidDType       = errors.New("invalid dtype")
	ErrInvalidAxis        = errors.New("invalid axis")
	ErrInvalidJoin        = errors.New("invalid join")
	ErrInvalidIndex       = errors.New("invalid index")
	ErrNotImplementedBase = errors.New("not implemented")
)

// NotImplemented returns an error wrapping ErrNotImplementedBase that names
// the unsupported feature, e.g. errs.NotImplemented("DataFrame.Stack").
func NotImplemented(feature string) error {
	return fmt.Errorf("%w: %s", ErrNotImplementedBase, feature)
}
