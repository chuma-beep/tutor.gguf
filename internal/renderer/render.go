package renderer

import (
	"strings"
)

// RenderSpan renders a single math span (without delimiters) to terminal text.
func RenderSpan(s string, ascii bool) string {
	return blockString(blockOf(parseMath(s, ascii), ascii))
}

// Render converts raw model output — prose interleaved with \(...\), \[...\],
// $...$ or $$...$$ math spans — into terminal-friendly text. Unclosed spans
// (mid-streaming) render as far as they can; unknown constructs degrade to a
// linear passthrough rather than being dropped.
func Render(text string, ascii bool) string {
	segs := extractSpans(text)
	var sb strings.Builder
	for i, s := range segs {
		switch s.typ {
		case segText:
			sb.WriteString(s.s)
		case segInline:
			sb.WriteString(RenderSpan(s.s, ascii))
		case segDisplay:
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(RenderSpan(s.s, ascii))
			if i < len(segs)-1 {
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

func blockString(b block) string {
	return strings.Join(b.lines, "\n")
}

// blockOf renders an AST node into a boxed layout block.
func blockOf(n Node, ascii bool) block {
	switch v := n.(type) {
	case Text:
		return tb(v.S)
	case Group:
		children := make([]block, 0, len(v.Children))
		for _, c := range v.Children {
			children = append(children, blockOf(c, ascii))
		}
		if len(children) == 0 {
			return tb("")
		}
		return hjoin(children...)
	case *Frac:
		return fracBlock(blockOf(v.Num, ascii), blockOf(v.Den, ascii), ascii)
	case *Sqrt:
		return sqrtBlock(blockOf(v.Body, ascii), ascii, v.Index)
	case *Script:
		var sup, sub block
		if v.SupSet {
			sup = blockOf(v.Sup, ascii)
		}
		if v.SubSet {
			sub = blockOf(v.Sub, ascii)
		}
		return scriptBlockOf(v, blockOf(v.Base, ascii), sup, sub, ascii)
	case *BigOp:
		var lo, hi block
		if v.LoSet {
			lo = blockOf(v.Lo, ascii)
		}
		if v.HiSet {
			hi = blockOf(v.Hi, ascii)
		}
		return bigopBlock(v.Sym, lo, hi, v.LoSet, v.HiSet)
	case BigOp:
		b := v
		return blockOf(&b, ascii)
	case *Nest:
		return nestBlock(v.Open, v.Close, blockOf(v.Body, ascii), ascii)
	case Nest:
		b := v
		return blockOf(&b, ascii)
	case *Boxed:
		return boxedBlock(blockOf(v.Body, ascii), ascii)
	case *Overline:
		return overlineBlock(blockOf(v.Body, ascii), ascii)
	case *Binom:
		return binomBlock(blockOf(v.N, ascii), blockOf(v.K, ascii), ascii)
	case *Unknown:
		return tb(v.Raw)
	case Unknown:
		return tb(v.Raw)
	}
	return tb("")
}

func binomBlock(n, k block, ascii bool) block {
	body := centeredStack([]block{n, k})
	return nestBlock("(", ")", body, ascii)
}

// scriptBlockOf prefers compact Unicode superscript/subscript glyphs for
// single-character scripts (x^2 → x²); anything else stacks base with the
// script above and/or below. In ASCII mode single-line scripts render linearly
// (x^2, x_i) to keep prose flowing.
func scriptBlockOf(s *Script, base, sup, sub block, ascii bool) block {
	if ascii {
		if base.H() == 1 {
			var sb strings.Builder
			sb.WriteString(strings.TrimRight(base.lines[0], " "))
			if s.SubSet {
				sb.WriteString("_" + linear(s.Sub, true))
			}
			if s.SupSet {
				sb.WriteString("^" + linear(s.Sup, true))
			}
			return tb(sb.String())
		}
	}
	both := s.SupSet && s.SubSet
	if !ascii && base.H() == 1 && !both {
		if s.SupSet {
			if g := scriptGlyph(linear(s.Sup, false), false); g != "" {
				return hjoin(base, tb(g))
			}
		}
		if s.SubSet {
			if g := scriptGlyph(linear(s.Sub, false), true); g != "" {
				return hjoin(base, tb(g))
			}
		}
	}
	return stackedScript(base, sup, sub, s.SupSet, s.SubSet)
}

func scriptGlyph(s string, sub bool) string {
	g, ok := compactScript(s, sub)
	if !ok {
		return ""
	}
	return g
}

func stackedScript(base, sup, sub block, supSet, subSet bool) block {
	return scriptBlock(base, sup, sub, supSet, subSet)
}

// linear reduces any node to a best-effort single-line string. Used for
// compact-script detection and read-only fallbacks.
func linear(n Node, ascii bool) string {
	switch v := n.(type) {
	case Text:
		return v.S
	case Group:
		var sb strings.Builder
		for _, c := range v.Children {
			sb.WriteString(linear(c, ascii))
		}
		return sb.String()
	case *Frac:
		return "(" + linear(v.Num, ascii) + ")/(" + linear(v.Den, ascii) + ")"
	case *Sqrt:
		body := linear(v.Body, ascii)
		if v.Index > 1 {
			return "nthroot(" + body + ")"
		}
		return "sqrt(" + body + ")"
	case *Script:
		var sb strings.Builder
		if v.Base != nil {
			sb.WriteString(linear(v.Base, ascii))
		}
		if v.SubSet {
			sb.WriteString("_" + linear(v.Sub, ascii))
		}
		if v.SupSet {
			sb.WriteString("^" + linear(v.Sup, ascii))
		}
		return sb.String()
	case *BigOp:
		s := v.Sym
		if v.LoSet {
			s += "_" + linear(v.Lo, ascii)
		}
		if v.HiSet {
			s += "^" + linear(v.Hi, ascii)
		}
		return s
	case BigOp:
		b := v
		return linear(&b, ascii)
	case *Nest:
		return v.Open + linear(v.Body, ascii) + v.Close
	case Nest:
		b := v
		return linear(&b, ascii)
	case *Boxed:
		return "[" + linear(v.Body, ascii) + "]"
	case *Overline:
		return linear(v.Body, ascii)
	case *Binom:
		return "(" + linear(v.N, ascii) + " " + linear(v.K, ascii) + ")"
	case *Unknown:
		return v.Raw
	}
	return ""
}

type segType int

const (
	segText segType = iota
	segInline
	segDisplay
)

type segment struct {
	typ segType
	s   string
}

// extractSpans splits text into prose and math spans. A span whose closing
// delimiter never arrives is treated as math through the end of input so
// streaming partial output renders progressively.
func extractSpans(text string) []segment {
	var segs []segment
	i := 0    // scan position
	prev := 0 // end of the last emitted chunk
	for i < len(text) {
		var opener, closer string
		var typ segType
		switch {
		case text[i] == '$' && i+1 < len(text) && text[i+1] == '$':
			opener, closer, typ = "$$", "$$", segDisplay
		case text[i] == '$':
			opener, closer, typ = "$", "$", segInline
		case text[i] == '\\' && i+1 < len(text) && text[i+1] == '(':
			opener, closer, typ = "\\(", "\\)", segInline
		case text[i] == '\\' && i+1 < len(text) && text[i+1] == '[':
			opener, closer, typ = "\\[", "\\]", segDisplay
		}
		if opener == "" {
			i++
			continue
		}
		if i > prev {
			segs = append(segs, segment{segText, text[prev:i]})
		}
		bodyStart := i + len(opener)
		j := strings.Index(text[bodyStart:], closer)
		if j >= 0 {
			segs = append(segs, segment{typ, text[bodyStart : bodyStart+j]})
			i = bodyStart + j + len(closer)
		} else {
			segs = append(segs, segment{typ, text[bodyStart:]})
			i = len(text)
		}
		prev = i
	}
	if i > prev {
		segs = append(segs, segment{segText, text[prev:]})
	}
	return segs
}
