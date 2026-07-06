package pandas

import "github.com/arturoeanton/go-pandas/errs"

// Sentinel errors, matchable with errors.Is.
var (
	ErrColumnNotFound     = errs.ErrColumnNotFound
	ErrIndexOutOfBounds   = errs.ErrIndexOutOfBounds
	ErrLengthMismatch     = errs.ErrLengthMismatch
	ErrTypeMismatch       = errs.ErrTypeMismatch
	ErrShapeMismatch      = errs.ErrShapeMismatch
	ErrBroadcastMismatch  = errs.ErrBroadcastMismatch
	ErrInvalidOperation   = errs.ErrInvalidOperation
	ErrInvalidDType       = errs.ErrInvalidDType
	ErrInvalidAxis        = errs.ErrInvalidAxis
	ErrInvalidJoin        = errs.ErrInvalidJoin
	ErrInvalidIndex       = errs.ErrInvalidIndex
	ErrNotImplementedBase = errs.ErrNotImplementedBase
)

// ErrNotImplemented returns an error naming an unsupported feature,
// wrapping ErrNotImplementedBase.
func ErrNotImplemented(feature string) error { return errs.NotImplemented(feature) }
