package dtype

import (
	"fmt"
	"strings"

	"github.com/arturoeanton/go-pandas/errs"
)

// timeDirectives maps pandas/strftime directives to Go layout fragments.
var timeDirectives = map[byte]string{
	'Y': "2006",  // four-digit year
	'y': "06",    // two-digit year
	'm': "01",    // zero-padded month
	'd': "02",    // zero-padded day
	'H': "15",    // hour 00-23
	'M': "04",    // minute 00-59
	'S': "05",    // second 00-59
	'z': "-0700", // numeric timezone offset
}

// TranslateTimeFormat converts a pandas/strftime-style format into a Go
// time layout (v0.9):
//
//	%Y-%m-%d          -> 2006-01-02
//	%Y-%m-%d %H:%M:%S -> 2006-01-02 15:04:05
//	%d/%m/%Y          -> 02/01/2006
//
// %f (microseconds, 1-6 digits) must follow a literal dot: ".%f"
// translates to ".999999". %% is a literal percent. Unknown directives
// error instead of silently mis-parsing.
func TranslateTimeFormat(format string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		i++
		if i >= len(format) {
			return "", fmt.Errorf("%w: dangling %% at end of datetime format %q", errs.ErrInvalidOperation, format)
		}
		d := format[i]
		switch d {
		case '%':
			b.WriteByte('%')
		case 'f':
			out := b.String()
			if !strings.HasSuffix(out, ".") {
				return "", fmt.Errorf("%w: %%f must follow a literal dot (\".%%f\") in datetime format %q", errs.ErrInvalidOperation, format)
			}
			b.WriteString("999999")
		default:
			layout, ok := timeDirectives[d]
			if !ok {
				return "", fmt.Errorf("%w: unsupported datetime directive %%%c in format %q", errs.ErrInvalidOperation, d, format)
			}
			b.WriteString(layout)
		}
	}
	return b.String(), nil
}

// InferTimeLayouts is the deterministic no-format inference list for
// ToDatetime (v0.9): conservative, documented, day-first for the
// slash-separated ambiguous form (02/01/2006 is tried before
// 01/02/2006). Explicit formats are preferred.
var InferTimeLayouts = []string{
	"2006-01-02T15:04:05.999999999Z07:00", // RFC3339 with optional fraction
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05.999999",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"2006/01/02",
	"02/01/2006",
	"01/02/2006",
}
