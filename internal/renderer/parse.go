package renderer

import (
	"strconv"
	"strings"
)

const maxDepth = 8

// parseMath parses one math span into an AST. It never fails: pathological or
// unsupported input degrades to a linear passthrough.
func parseMath(s string, ascii bool) Node {
	p := &parser{toks: tokenizeMath(s), ascii: ascii}
	return p.seq()
}

type parser struct {
	toks  []token
	pos   int
	ascii bool
	depth int
}

func (p *parser) peek() *token {
	if p.pos >= len(p.toks) {
		return nil
	}
	return &p.toks[p.pos]
}

func (p *parser) next() *token {
	t := p.peek()
	if t != nil {
		p.pos++
	}
	return t
}

// rawRest returns a linear rendering of every remaining token; used by the
// depth/unknown fallbacks so no input is ever dropped.
func (p *parser) rawRest() string {
	var sb strings.Builder
	for _, t := range p.toks[p.pos:] {
		sb.WriteString(tokenString(t))
	}
	p.pos = len(p.toks)
	return sb.String()
}

func tokenString(t token) string {
	switch t.kind {
	case tokWord:
		if t.val == "\\" {
			return " │ "
		}
		return "\\" + t.val
	case tokAlign:
		return "  "
	case tokSup:
		return "^"
	case tokSub:
		return "_"
	case tokOpen:
		return "{"
	case tokClose:
		return "}"
	case tokLBracket:
		return "["
	case tokRBracket:
		return "]"
	default:
		return t.val
	}
}

// seq parses a sequence of atoms until the end of input or a closing brace.
func (p *parser) seq() Group {
	return p.seqUntil(false)
}

// seqUntil parses a sequence of atoms. When stopAtRight is true the sequence
// ends (without consuming) at a \right control word; used for \left...\right
// bodies.
func (p *parser) seqUntil(stopAtRight bool) Group {
	p.depth++
	if p.depth > maxDepth {
		return Group{Children: []Node{Text{S: p.rawRest()}}}
	}
	defer func() { p.depth-- }()

	var children []Node

	for {
		t := p.peek()
		if t == nil {
			break
		}
		switch t.kind {
		case tokClose:
			// End of a {...} group; caller consumes it.
			return Group{Children: children}
		case tokOpen:
			p.next()
			inner := p.seq()
			if c := p.peek(); c != nil && c.kind == tokClose {
				p.next()
			}
			children = append(children, p.parseScripts(inner))
		case tokWord:
			name := t.val
			if name == "right" && stopAtRight {
				return Group{Children: children}
			}
			p.next()
			switch name {
			case "left":
				children = append(children, p.parseScripts(p.parseNest()))
			case "right":
				// stray \right without \left
				close := p.readNestDelim()
				children = append(children, Unknown{Raw: "\\right" + close})
			case "begin":
				children = append(children, Unknown{Raw: p.captureBegin()})
			case "end":
				// stray \end without \begin
				if c := p.peek(); c != nil && c.kind == tokOpen {
					p.next()
					p.readNameUntilClose()
				}
			default:
				if n := p.controlWord(name); n != nil {
					children = append(children, p.parseScripts(n))
				}
			}
		case tokSup, tokSub:
			// Script operator with no base atom before it.
			p.next()
			arg := p.scriptOperand()
			if t.kind == tokSub {
				children = append(children, &Script{Sub: arg, SubSet: true})
			} else {
				children = append(children, &Script{Sup: arg, SupSet: true})
			}
		default:
			p.next()
			children = append(children, p.parseScripts(Text{S: t.val}))
		}
	}
	return Group{Children: children}
}

// parseNest consumes a \left<delim>...\right<delim> construct (delimiters
// already read) and returns the matching Nest. An unclosed \left degrades to a
// flat body so nothing is dropped.
func (p *parser) parseNest() Node {
	open := p.readNestDelim()
	body := p.seqUntil(true)
	var close string
	if c := p.peek(); c != nil && c.kind == tokWord && c.val == "right" {
		p.next()
		close = p.readNestDelim()
		return Nest{Open: open, Close: close, Body: body}
	}
	// \right never arrived (EOF or enclosing brace). Keep the parsed atoms as
	// a plain group; any trailing script binds to the Nest-worthy body anyway.
	return body
}

// parseScripts greedily consumes any trailing ^/_ after an atom.
func (p *parser) parseScripts(n Node) Node {
	for {
		t := p.peek()
		if t == nil || (t.kind != tokSup && t.kind != tokSub) {
			return n
		}
		sub := t.kind == tokSub
		p.next()
		arg := p.scriptOperand()
		n = attachScript(n, sub, arg)
	}
}

func attachScript(base Node, sub bool, arg Node) Node {
	switch b := base.(type) {
	case BigOp:
		if sub {
			b.Lo, b.LoSet = arg, true
		} else {
			b.Hi, b.HiSet = arg, true
		}
		return b
	case *BigOp:
		bb := *b
		if sub {
			bb.Lo, bb.LoSet = arg, true
		} else {
			bb.Hi, bb.HiSet = arg, true
		}
		return &bb
	case Script:
		bs := b
		return attachScript(&bs, sub, arg)
	case *Script:
		sc := *b
		if sub && !sc.SubSet {
			sc.Sub, sc.SubSet = arg, true
			return &sc
		}
		if !sub && !sc.SupSet {
			sc.Sup, sc.SupSet = arg, true
			return &sc
		}
		if sub {
			return &Script{Base: &sc, SubSet: true, Sub: arg}
		}
		return &Script{Base: &sc, SupSet: true, Sup: arg}
	case Text:
		if sub {
			return &Script{Base: b, SubSet: true, Sub: arg}
		}
		return &Script{Base: b, SupSet: true, Sup: arg}
	default:
		if sub {
			return &Script{Base: base, SubSet: true, Sub: arg}
		}
		return &Script{Base: base, SupSet: true, Sup: arg}
	}
}

// scriptOperand parses the operand of ^ or _: a {...} group or a single atom.
// Unbraced text runs that contain a space would otherwise swallow the atoms
// that follow (x^2 + y), so only the first rune is taken and the rest spliced
// back into the stream, mirroring TeX's single-token script binding.
func (p *parser) scriptOperand() Node {
	t := p.peek()
	if t == nil {
		return Text{}
	}
	if t.kind == tokOpen {
		return p.groupArg()
	}
	if t.kind == tokText && strings.Contains(t.val, " ") && len([]rune(t.val)) > 1 {
		p.next()
		rs := []rune(t.val)
		first := string(rs[0])
		rest := string(rs[1:])
		splice := token{tokText, rest, t.from + 1, t.to}
		p.toks = append(p.toks[:p.pos], append([]token{splice}, p.toks[p.pos:]...)...)
		return Text{S: first}
	}
	return p.arg()
}

// groupArg consumes a {...} group and returns its inner sequence.
func (p *parser) groupArg() Group {
	p.next() // {
	inner := p.seq()
	if c := p.peek(); c != nil && c.kind == tokClose {
		p.next()
	}
	return inner
}

// arg parses one required LaTeX argument: a braced group or a single atom.
func (p *parser) arg() Node {
	t := p.peek()
	if t == nil {
		return Text{}
	}
	if t.kind == tokOpen {
		return p.groupArg()
	}
	if t.kind == tokClose {
		return Text{}
	}
	return p.atomBase()
}

// atomBase parses a single atom (not a brace group) starting at the current
// token. Scripts that follow it are left for the caller's parseScripts.
func (p *parser) atomBase() Node {
	t := p.peek()
	if t == nil {
		return Text{}
	}
	p.next()
	if t.kind == tokWord {
		if n := p.controlWord(t.val); n != nil {
			return n
		}
	}
	return Text{S: t.val}
}

func (p *parser) controlWord(name string) Node {
	if sym, ok := bigOpSymbol(name, p.ascii); ok {
		return BigOp{Sym: sym}
	}
	if hasSymbol(name) {
		return Text{S: maybeASCII(symbolMap[name], name, p.ascii)}
	}
	if functionWords[name] {
		return Text{S: name}
	}

	switch name {
	case "frac", "dfrac", "tfrac":
		num, den := p.arg(), p.arg()
		return &Frac{Num: num, Den: den}
	case "sqrt":
		index := 0
		if t := p.peek(); t != nil && t.kind == tokLBracket {
			p.next()
			index = nthInt(p.readUntilRBracket())
		}
		return &Sqrt{Index: index, Body: p.arg()}
	case "binom":
		return &Binom{N: p.arg(), K: p.arg()}
	case "text", "mathrm", "operatorname", "textbf", "mathit",
		"textrm", "mbox", "textnormal", "textsf", "emph", "bf", "tt", "it":
		return Text{S: p.readLiteralGroup()}
	case "overline":
		return &Overline{Body: p.arg()}
	case "boxed", "fbox":
		return &Boxed{Body: p.arg()}
	}

	// Spacing and escaped literals.
	switch name {
	case ",", ";", "!", ":", " ", "quad", "qquad":
		return Text{S: " "}
	case "hline":
		return Text{}
	case "\\":
		return Text{S: " │ "}
	case "{":
		return Text{S: "{"}
	case "}":
		return Text{S: "}"}
	case "%", "&", "#", "$", "_", "^", "~", "|", "(":
		return Text{S: name}
	case "ldots", "cdots", "dots":
		return Text{S: "\u2026"}
	}

	spaces := map[string]string{",": " ", ";": " ", "!": " ", ":": " ", " ": " ", "quad": "  ", "qquad": "  "}
	for k, v := range spaces {
		if name == k {
			return Text{S: v}
		}
	}

	if name == " " {
		return Text{S: " "}
	}

	// Unknown macro: passthrough so nothing is dropped.
	return Text{S: "\\" + name}
}

// readUntilRBracket consumes tokens up to and including a closing bracket,
// returning the accumulated text.
func (p *parser) readUntilRBracket() string {
	var sb strings.Builder
	for {
		t := p.next()
		if t == nil {
			return sb.String()
		}
		if t.kind == tokRBracket {
			return sb.String()
		}
		sb.WriteString(tokenString(*t))
	}
}

// readNameUntilClose consumes a {...} group (braces already consumed) and
// returns the raw inside text used for environment names.
func (p *parser) readNameUntilClose() string {
	var sb strings.Builder
	for {
		t := p.next()
		if t == nil {
			return sb.String()
		}
		if t.kind == tokClose {
			return sb.String()
		}
		sb.WriteString(tokenString(*t))
	}
}

// readLiteralGroup reads a {...} group as raw literal text (for \text, etc.),
// converting embedded \\ and & so a quoted macro never leaks control.
func (p *parser) readLiteralGroup() string {
	p.next() // {
	var sb strings.Builder
	for {
		t := p.next()
		if t == nil {
			return sb.String()
		}
		if t.kind == tokClose {
			return sb.String()
		}
		sb.WriteString(tokenString(*t))
	}
}

// captureBegin reads a \begin{env}...\end{env} block and linearizes it for the
// Unknown passthrough (synthetic-division tables, cases).
func (p *parser) captureBegin() string {
	var sb strings.Builder
	if t := p.peek(); t != nil && t.kind == tokOpen {
		p.next()
		env := p.readNameUntilClose()
		sb.WriteString(env)
		sb.WriteString(": ")
	}
	for {
		t := p.peek()
		if t == nil {
			return strings.TrimSpace(sb.String())
		}
		if t.kind == tokWord && t.val == "end" {
			p.next()
			if c := p.peek(); c != nil && c.kind == tokOpen {
				p.next()
				p.readNameUntilClose()
			}
			return strings.TrimSpace(sb.String())
		}
		p.next()
		switch t.kind {
		case tokAlign:
			sb.WriteString("  ")
		case tokWord:
			sb.WriteString(tokenString(*t))
		default:
			sb.WriteString(tokenString(*t))
		}
	}
}

// readNestDelim reads the delimiter that follows \left or \right.
func (p *parser) readNestDelim() string {
	t := p.next()
	if t == nil {
		return ""
	}
	switch t.kind {
	case tokText, tokLBracket, tokRBracket:
		return t.val
	case tokWord:
		switch t.val {
		case ".":
			return ""
		case "lbrace":
			return "{"
		case "rbrace":
			return "}"
		case "vert":
			return "|"
		case "Vert":
			return "‖"
		case "lbrack":
			return "["
		case "rbrack":
			return "]"
		case "langle":
			return maybeASCII("\u27e8", t.val, p.ascii)
		case "rangle":
			return maybeASCII("\u27e9", t.val, p.ascii)
		case "lfloor":
			return maybeASCII("\u230a", t.val, p.ascii)
		case "rfloor":
			return maybeASCII("\u230b", t.val, p.ascii)
		case "lceil":
			return maybeASCII("\u2308", t.val, p.ascii)
		case "rceil":
			return maybeASCII("\u2309", t.val, p.ascii)
		}
		return "\\" + t.val
	}
	return t.val
}

// maybeASCII returns unicode, or the ASCII fallback for name when ascii mode is
// enabled and the asciiMap has an entry for it.
func maybeASCII(unicode, name string, ascii bool) string {
	if ascii {
		if s, ok := asciiMap[name]; ok {
			return s
		}
	}
	return unicode
}

func nthInt(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
