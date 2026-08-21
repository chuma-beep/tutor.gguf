package renderer

import (
	"fmt"
	"strings"
)

// block is a rectangular text glyph with an explicit baseline row, the line
// from which adjacent blocks hang when laid out side by side.
type block struct {
	lines []string
	base  int
}

func tb(s string) block {
	return block{lines: strings.Split(s, "\n"), base: 0}
}

func (b block) W() int {
	w := 0
	for _, l := range b.lines {
		if n := len([]rune(l)); n > w {
			w = n
		}
	}
	return w
}

func (b block) H() int { return len(b.lines) }

func (b block) padRight(w int) block {
	if b.W() >= w {
		return b
	}
	lines := make([]string, len(b.lines))
	for i, l := range b.lines {
		lines[i] = l + strings.Repeat(" ", w-len([]rune(l)))
	}
	return block{lines: lines, base: b.base}
}

func (b block) centerTo(w int) block {
	if w <= b.W() {
		return b
	}
	lines := make([]string, len(b.lines))
	for i, l := range b.lines {
		n := len([]rune(l))
		left := (w - n) / 2
		lines[i] = strings.Repeat(" ", left) + l + strings.Repeat(" ", w-n-left)
	}
	return block{lines: lines, base: b.base}
}

// hjoin lays blocks out horizontally on a shared baseline. The composite
// baseline is the maximum of the inputs, so tall and short blocks center
// around the same visual axis.
func hjoin(bs ...block) block {
	above, below := 0, 0
	for _, b := range bs {
		if b.base > above {
			above = b.base
		}
	}
	for _, b := range bs {
		if d := b.H() - b.base - 1; d > below {
			below = d
		}
	}
	h := above + below + 1
	rows := make([][]rune, h)
	for _, b := range bs {
		top := above - b.base
		bw := b.W()
		for j, l := range b.lines {
			rows[top+j] = append(rows[top+j], []rune(l)...)
		}
		for r := 0; r < h; r++ {
			if r < top || r >= top+b.H() {
				rows[r] = append(rows[r], []rune(strings.Repeat(" ", bw))...)
			}
		}
	}
	lines := make([]string, h)
	for r := range rows {
		lines[r] = string(rows[r])
	}
	return block{lines: lines, base: above}
}

// centeredStack stacks blocks vertically, centering each to the widest block.
func centeredStack(bs []block) block {
	w := 0
	for _, b := range bs {
		if bw := b.W(); bw > w {
			w = bw
		}
	}
	var lines []string
	for _, b := range bs {
		lines = append(lines, b.centerTo(w).lines...)
	}
	if w == 0 {
		return block{lines: []string{""}, base: 0}
	}
	return block{lines: lines, base: 0}
}

func fracBlock(num, den block, ascii bool) block {
	w := max(1, num.W(), den.W())
	num, den = num.centerTo(w), den.centerTo(w)
	rule := strings.Repeat("─", w)
	if ascii {
		rule = strings.Repeat("-", w)
	}
	lines := make([]string, 0, num.H()+1+den.H())
	lines = append(lines, num.lines...)
	lines = append(lines, rule)
	lines = append(lines, den.lines...)
	return block{lines: lines, base: num.H()}
}

func sqrtBlock(body block, ascii bool, index int) block {
	if body.H() == 0 {
		body = tb("")
	}
	w := max(1, body.W())
	prefix := "√"
	rule, vert := "─", "│"
	if ascii {
		prefix, rule, vert = "V", "-", "|"
		if index > 0 {
			prefix = fmt.Sprintf("V%d", index)
		}
	}
	switch index {
	case 3:
		if !ascii {
			prefix = "\u221b" // ∛
		}
	case 4:
		if !ascii {
			prefix = "\u221c" // ∜
		}
	}
	if body.H() == 1 {
		// Flat radicand: keep the radical on one line, parenthesised so the
		// extent of the root is unambiguous.
		return tb(prefix + "(" + body.lines[0] + ")")
	}
	lines := []string{prefix + strings.Repeat(rule, w+1)}
	for _, l := range body.lines {
		lines = append(lines, vert+padRight(l, w))
	}
	return block{lines: lines, base: 1 + body.base}
}

func bigopBlock(sym string, lo, hi block, loSet, hiSet bool) block {
	var bs []block
	baseRow := 0
	if hiSet {
		bs = append(bs, hi)
		baseRow = hi.H()
	}
	bs = append(bs, tb(sym))
	if loSet {
		bs = append(bs, lo)
	}
	b := centeredStack(bs)
	b.base = baseRow
	return b
}

func scriptBlock(base, sup, sub block, supSet, subSet bool) block {
	var bs []block
	baseRow := 0
	if supSet {
		bs = append(bs, sup)
		baseRow = sup.H()
	}
	bs = append(bs, base)
	if subSet {
		bs = append(bs, sub)
	}
	b := centeredStack(bs)
	b.base = baseRow
	return b
}

// nestBlock grows the delimiters around a multi-line body.
func nestBlock(open, close string, body block, ascii bool) block {
	if body.H() == 0 {
		body = tb("")
	}
	if body.H() == 1 {
		return hjoin(tb(open), body, tb(close))
	}
	oc := delimColumn(open, body.H(), ascii)
	cc := delimColumn(close, body.H(), ascii)
	lines := make([]string, body.H())
	for i := range lines {
		lines[i] = oc.lines[i] + body.lines[i] + cc.lines[i]
	}
	return block{lines: lines, base: body.base}
}

func delimColumn(c string, h int, ascii bool) block {
	if c == "." {
		lines := make([]string, h)
		return block{lines: lines, base: 0}
	}
	if h <= 1 {
		return tb(c)
	}
	top, mid, bot := c, c, c
	if ascii {
		switch c {
		case "(":
			top, mid, bot = "(", "|", "|"
		case ")":
			top, mid, bot = "|", "|", ")"
		case "{":
			top, mid, bot = "{", "|", "|"
		case "}":
			top, mid, bot = "|", "|", "}"
		case "[":
			top, mid, bot = "[", "|", "|"
		case "]":
			top, mid, bot = "|", "|", "]"
		case "⟨":
			top, mid, bot = "<", "<", "<"
		case "⟩":
			top, mid, bot = ">", ">", ">"
		case "‖":
			top, mid, bot = "||", "||", "||"
		}
	} else {
		switch c {
		case "(":
			top, mid, bot = "\u239b", "\u239c", "\u239d" // ⎛⎜⎝
		case ")":
			top, mid, bot = "\u239e", "\u239f", "\u23a0" // ⎞⎟⎠
		case "{":
			top, mid, bot = "\u23a7", "\u23a8", "\u23a9" // ⎧⎨⎩
		case "}":
			top, mid, bot = "\u23ab", "\u23ac", "\u23ad" // ⎫⎬⎭
		case "[":
			top, mid, bot = "\u23a1", "\u23a2", "\u23a3" // ⎡⎢⎣
		case "]":
			top, mid, bot = "\u23a4", "\u23a5", "\u23a6" // ⎤⎥⎦
		case "|":
			top, mid, bot = "\u2502", "\u2502", "\u2502" // │
		}
	}
	lines := make([]string, h)
	for i := range lines {
		switch i {
		case 0:
			lines[i] = top
		case h - 1:
			lines[i] = bot
		default:
			lines[i] = mid
		}
	}
	return block{lines: lines, base: 0}
}

func boxedBlock(body block, ascii bool) block {
	if body.H() == 0 {
		body = tb(" ")
	}
	w := max(1, body.W())
	var tl, tr, bl, br, hor, vert string
	if ascii {
		tl, tr, bl, br, hor, vert = "+", "+", "+", "+", "-", "|"
	} else {
		tl, tr, bl, br, hor, vert = "┌", "┐", "└", "┘", "─", "│"
	}
	lines := []string{tl + strings.Repeat(hor, w+2) + tr}
	for _, l := range body.lines {
		lines = append(lines, vert+" "+padRight(l, w)+" "+vert)
	}
	lines = append(lines, bl+strings.Repeat(hor, w+2)+br)
	return block{lines: lines, base: 1 + body.base}
}

func overlineBlock(body block, ascii bool) block {
	w := max(1, body.W())
	rule := strings.Repeat("─", w+2)
	if ascii {
		rule = strings.Repeat("-", w+2)
	}
	lines := []string{rule}
	for _, l := range body.lines {
		lines = append(lines, " "+padRight(l, w)+" ")
	}
	return block{lines: lines, base: 1 + body.base}
}

func padRight(s string, w int) string {
	if n := len([]rune(s)); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func max(a int, rest ...int) int {
	for _, r := range rest {
		if r > a {
			a = r
		}
	}
	return a
}
