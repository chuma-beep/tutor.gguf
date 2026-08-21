// Package renderer turns LaTeX math as emitted by the tutor model into a
// terminal-friendly Unicode/ASCII layout.
//
// Scope is deliberately the subset actually produced by Qwen2.5-Math-1.5B in
// eval transcripts: nested \frac (depth 2-3), \sqrt, superscripts/subscripts,
// \sum/\int/\prod limits, \binom, \left...\right delimiters, Greek letters and
// common operators, \boxed, \overline, \text. Anything else (matrices, unknown
// macros) degrades to a linear passthrough instead of erroring.
package renderer

// Node is an element of the parsed math AST.
type Node interface{ kind() string }

// Text is literal output (letters, digits, operators, spacing).
type Text struct{ S string }

func (Text) kind() string { return "text" }

// Group is a [...] sequence of atoms laid out horizontally.
type Group struct{ Children []Node }

func (Group) kind() string { return "group" }

// Frac renders numerator over denominator with a bar.
type Frac struct{ Num, Den Node }

func (Frac) kind() string { return "frac" }

// Sqrt renders a radical over Body; Index != 0 renders an nth root.
type Sqrt struct {
	Index int
	Body  Node
}

func (Sqrt) kind() string { return "sqrt" }

// Script attaches an optional subscript/superscript to a base atom.
type Script struct {
	Base           Node
	Sub, Sup       Node
	SubSet, SupSet bool
}

func (Script) kind() string { return "script" }

// BigOp is \sum, \int, \prod or \lim with optional below/above limits.
type BigOp struct {
	Sym          string
	Lo, Hi       Node
	LoSet, HiSet bool
}

func (BigOp) kind() string { return "bigop" }

// Nest is a \left...\right pair whose delimiters grow to match the body.
type Nest struct {
	Open, Close string
	Body        Node
}

func (Nest) kind() string { return "nest" }

// Boxed is a \boxed{...} final answer.
type Boxed struct{ Body Node }

func (Boxed) kind() string { return "boxed" }

// Overline is \overline{...}.
type Overline struct{ Body Node }

func (Overline) kind() string { return "overline" }

// Binom is \binom{n}{k}.
type Binom struct{ N, K Node }

func (Binom) kind() string { return "binom" }

// Unknown is verbatim passthrough for constructs without structured layout
// (e.g. \begin{array} synthetic-division tables).
type Unknown struct{ Raw string }

func (Unknown) kind() string { return "unknown" }
