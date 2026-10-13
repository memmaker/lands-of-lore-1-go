package text

import (
	"strconv"
	"strings"
)

// Sprintf formats like the original's C library vsprintf for the conversions the game's
// strings use: d, i, u, x, X, c and s with the flags -, +, space, # and 0 and a width.
// Numbers are 16-bit: d and i signed, the others unsigned. args holds ints and strings.
func Sprintf(format string, args ...any) string {
	var b strings.Builder
	next := 0
	arg := func() any {
		if next >= len(args) {
			return nil
		}
		next++
		return args[next-1]
	}
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		i++
		var left, plus, space, alt, zero bool
	flags:
		for ; i < len(format); i++ {
			switch format[i] {
			case '-':
				left = true
			case '+':
				plus = true
			case ' ':
				space = true
			case '#':
				alt = true
			case '0':
				zero = true
			default:
				break flags
			}
		}
		width := 0
		for ; i < len(format) && format[i] >= '0' && format[i] <= '9'; i++ {
			width = width*10 + int(format[i]-'0')
		}
		if i >= len(format) {
			break
		}
		var s, sign string
		switch conv := format[i]; conv {
		case 'd', 'i':
			n, _ := arg().(int)
			v := int(int16(n))
			switch {
			case v < 0:
				sign, v = "-", -v
			case plus:
				sign = "+"
			case space:
				sign = " "
			}
			s = strconv.Itoa(v)
		case 'u':
			n, _ := arg().(int)
			s = strconv.Itoa(int(uint16(n)))
		case 'x', 'X':
			n, _ := arg().(int)
			s = strconv.FormatUint(uint64(uint16(n)), 16)
			if conv == 'X' {
				s = strings.ToUpper(s)
			}
			if alt && n != 0 {
				sign = "0" + string(conv)
			}
		case 'c':
			n, _ := arg().(int)
			s = string(rune(byte(n)))
			zero = false
		case 's':
			s, _ = arg().(string)
			zero = false
		case '%':
			s = "%"
		default:
			b.WriteByte(conv)
			continue
		}
		pad := width - len(sign) - len(s)
		switch {
		case pad <= 0:
			b.WriteString(sign + s)
		case left:
			b.WriteString(sign + s + strings.Repeat(" ", pad))
		case zero:
			b.WriteString(sign + strings.Repeat("0", pad) + s)
		default:
			b.WriteString(strings.Repeat(" ", pad) + sign + s)
		}
	}
	return b.String()
}
