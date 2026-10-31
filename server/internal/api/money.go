package api

import (
	"math"
	"strings"
)

// round2 四舍五入到分。
func round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}

var cnDigits = []string{"零", "壹", "贰", "叁", "肆", "伍", "陆", "柒", "捌", "玖"}
var cnUnits = []string{"", "拾", "佰", "仟"}
var cnBigUnits = []string{"", "万", "亿", "万亿"}

// RMBUpper 金额转人民币大写，例如 1234.56 -> 壹仟贰佰叁拾肆元伍角陆分。
func RMBUpper(amount float64) string {
	neg := amount < 0
	if neg {
		amount = -amount
	}
	cents := int64(math.Round(amount * 100))
	if cents == 0 {
		return "零元整"
	}
	yuan := cents / 100
	fen := cents % 100
	jiao := fen / 10
	frac := fen % 10

	var sb strings.Builder
	if neg {
		sb.WriteString("负")
	}
	if yuan == 0 {
		sb.WriteString("零元")
	} else {
		sb.WriteString(intUpper(yuan))
		sb.WriteString("元")
	}
	if jiao == 0 && frac == 0 {
		sb.WriteString("整")
	} else {
		if jiao > 0 {
			sb.WriteString(cnDigits[jiao])
			sb.WriteString("角")
		} else if yuan > 0 {
			sb.WriteString("零")
		}
		if frac > 0 {
			sb.WriteString(cnDigits[frac])
			sb.WriteString("分")
		}
	}
	return sb.String()
}

// intUpper 整数转大写，按 4 位分节。
func intUpper(n int64) string {
	if n == 0 {
		return "零"
	}
	var sections []int64
	for n > 0 {
		sections = append(sections, n%10000)
		n /= 10000
	}
	var sb strings.Builder
	for i := len(sections) - 1; i >= 0; i-- {
		sec := sections[i]
		if sec == 0 {
			// 中间整节为零时补一个零（末尾节不补）
			if sb.Len() > 0 && i != 0 {
				s := sb.String()
				if !strings.HasSuffix(s, "零") {
					sb.WriteString("零")
				}
			}
			continue
		}
		text := sectionUpper(sec)
		// 高位节之后、低位节不足千位时要补零，如 1002003
		if sb.Len() > 0 && sec < 1000 {
			s := sb.String()
			if !strings.HasSuffix(s, "零") {
				sb.WriteString("零")
			}
		}
		sb.WriteString(text)
		sb.WriteString(cnBigUnits[i])
	}
	out := sb.String()
	out = strings.TrimSuffix(out, "零")
	return out
}

func sectionUpper(sec int64) string {
	var sb strings.Builder
	zero := false
	for i := 3; i >= 0; i-- {
		d := (sec / int64(math.Pow10(i))) % 10
		if d == 0 {
			zero = true
			continue
		}
		if zero && sb.Len() > 0 {
			sb.WriteString("零")
		}
		zero = false
		sb.WriteString(cnDigits[d])
		sb.WriteString(cnUnits[i])
	}
	return sb.String()
}
