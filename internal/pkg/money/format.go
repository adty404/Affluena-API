// Package money provides small formatting helpers for IDR minor-unit amounts.
//
// Affluena stores money as integer minor units. For IDR there is no decimal
// subunit in practice — a stored value of 20000000 means Rp 20.000.000 — so
// these helpers format the integer directly without dividing by 100.
package money

import "strconv"

// GroupIDR formats an integer amount using "." as the thousands separator,
// matching Indonesian rupiah conventions (e.g. 20000000 -> "20.000.000").
// Negative values keep a leading "-" (e.g. -1500 -> "-1.500").
func GroupIDR(n int64) string {
	negative := n < 0
	// Work on the absolute value as a string. Use uint64 so math.MinInt64
	// negates without overflow.
	var abs uint64
	if negative {
		abs = uint64(-(n + 1)) + 1
	} else {
		abs = uint64(n)
	}

	digits := strconv.FormatUint(abs, 10)

	// Insert a "." before every group of three digits from the right.
	n3 := len(digits)
	// Number of separators to add.
	sep := (n3 - 1) / 3
	if sep == 0 {
		if negative {
			return "-" + digits
		}
		return digits
	}

	buf := make([]byte, 0, n3+sep+1)
	if negative {
		buf = append(buf, '-')
	}
	// Leading group may be 1-3 digits.
	lead := n3 % 3
	if lead == 0 {
		lead = 3
	}
	buf = append(buf, digits[:lead]...)
	for i := lead; i < n3; i += 3 {
		buf = append(buf, '.')
		buf = append(buf, digits[i:i+3]...)
	}
	return string(buf)
}
