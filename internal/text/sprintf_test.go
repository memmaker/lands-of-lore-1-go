package text

import "testing"

func TestSprintf(t *testing.T) {
	for _, c := range []struct {
		format string
		args   []any
		want   string
	}{
		{"%s has %d gold", []any{"Conrad", 12}, "Conrad has 12 gold"},
		{"%5d|%-4d|%04d", []any{-3, 7, 42}, "   -3|7   |0042"},
		{"%u %x %X", []any{-1, 255, 255}, "65535 ff FF"},
		{"100%%", nil, "100%"},
	} {
		if got := Sprintf(c.format, c.args...); got != c.want {
			t.Errorf("Sprintf(%q) = %q, want %q", c.format, got, c.want)
		}
	}
}
