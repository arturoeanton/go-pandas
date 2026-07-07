package index

import "time"

// Typed label accessors used by the join engine (v0.6). The returned
// slices alias internal storage and must be treated as read-only.

// Strings exposes a StringIndex's labels.
func (ix *StringIndex) Strings() []string { return ix.values }

// Times exposes a DatetimeIndex's labels.
func (ix *DatetimeIndex) Times() []time.Time { return ix.values }

// Int64s exposes an Int64Index's labels.
func (ix *Int64Index) Int64s() []int64 { return ix.values }
