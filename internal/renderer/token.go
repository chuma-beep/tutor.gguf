package renderer

type tokKind int

const (
	tokText     tokKind = iota // literal run (may include spaces)
	tokWord                    // \control-name or \x control symbol
	tokSup                     // ^
	tokSub                     // _
	tokOpen                    // {
	tokClose                   // }
	tokAlign                   // &
	tokLBracket                // [
	tokRBracket                // ]
)

type token struct {
	kind tokKind
	val  string
	from int // rune offset of first source rune (for raw slicing)
	to   int // rune offset one past the last source rune
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// tokenizeMath splits a math span into tokens. Literal text — including
// spaces, which LaTeX math would normally ignore but which keep the model's
// output readable — is kept inside tokText runs.
func tokenizeMath(s string) []token {
	rs := []rune(s)
	var toks []token
	i := 0
	for i < len(rs) {
		r := rs[i]
		switch r {
		case '\\':
			start := i
			if i+1 < len(rs) && (rs[i+1] == ' ' || rs[i+1] == '\n') {
				toks = append(toks, token{tokText, " ", start, start + 2})
				i += 2
				continue
			}
			// A control symbol is a backslash plus a single non-letter.
			j := i + 1
			if j < len(rs) && !isLetter(rs[j]) {
				toks = append(toks, token{tokWord, string(rs[j]), start, j + 1})
				j++
			} else {
				end := j
				for end < len(rs) && isLetter(rs[end]) {
					end++
				}
				toks = append(toks, token{tokWord, string(rs[j:end]), start, end})
				j = end
			}
			i = j
		case '^':
			toks = append(toks, token{tokSup, "^", i, i + 1})
			i++
		case '_':
			toks = append(toks, token{tokSub, "_", i, i + 1})
			i++
		case '{':
			toks = append(toks, token{tokOpen, "{", i, i + 1})
			i++
		case '}':
			toks = append(toks, token{tokClose, "}", i, i + 1})
			i++
		case '&':
			toks = append(toks, token{tokAlign, "&", i, i + 1})
			i++
		case '[':
			toks = append(toks, token{tokLBracket, "[", i, i + 1})
			i++
		case ']':
			toks = append(toks, token{tokRBracket, "]", i, i + 1})
			i++
		case '(', ')':
			// Parens are atomic tokens so \left(\right) delimiters read
			// cleanly; visually they still join their neighbours.
			toks = append(toks, token{tokText, string(r), i, i + 1})
			i++
		case '~':
			toks = append(toks, token{tokText, " ", i, i + 1})
			i++
		case '\n', '\t', '\r':
			toks = append(toks, token{tokText, " ", i, i + 1})
			i++
		case '$':
			i++ // stray dollar inside an extracted span is cosmetic
		default:
			j := i
			for j < len(rs) && !isSpecial(rs[j]) {
				j++
			}
			toks = append(toks, token{tokText, string(rs[i:j]), i, j})
			i = j
		}
	}
	return toks
}

func isSpecial(r rune) bool {
	switch r {
	case '\\', '^', '_', '{', '}', '&', '[', ']', '(', ')', '~', '\n', '\t', '\r', '$':
		return true
	}
	return false
}
