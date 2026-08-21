package renderer

// symbolMap maps LaTeX control names to their Unicode rendering.
var symbolMap = map[string]string{
	// Greek lowercase
	"alpha":    "\u03b1", // α
	"beta":     "\u03b2", // β
	"gamma":    "\u03b3", // γ
	"delta":    "\u03b4", // δ
	"epsilon":  "\u03b5", // ε
	"varep":    "\u03f5", // ϵ
	"zeta":     "\u03b6", // ζ
	"eta":      "\u03b7", // η
	"theta":    "\u03b8", // θ
	"vartheta": "\u03d1", // ϑ
	"iota":     "\u03b9", // ι
	"kappa":    "\u03ba", // κ
	"lambda":   "\u03bb", // λ
	"mu":       "\u03bc", // μ
	"nu":       "\u03bd", // ν
	"xi":       "\u03be", // ξ
	"pi":       "\u03c0", // π
	"varpi":    "\u03d6", // ϖ
	"rho":      "\u03c1", // ρ
	"sigma":    "\u03c3", // σ
	"tau":      "\u03c4", // τ
	"upsilon":  "\u03c5", // υ
	"phi":      "\u03c6", // φ
	"varphi":   "\u03d5", // ϕ
	"chi":      "\u03c7", // χ
	"psi":      "\u03c8", // ψ
	"omega":    "\u03c9", // ω

	// Greek uppercase
	"Gamma":   "\u0393", // Γ
	"Delta":   "\u0394", // Δ
	"Theta":   "\u0398", // Θ
	"Lambda":  "\u039b", // Λ
	"Xi":      "\u039e", // Ξ
	"Pi":      "\u03a0", // Π
	"Sigma":   "\u03a3", // Σ
	"Upsilon": "\u03a5", // Υ
	"Phi":     "\u03a6", // Φ
	"Psi":     "\u03a8", // Ψ
	"Omega":   "\u03a9", // Ω

	// Binary operators
	"pm":       "\u00b1", // ±
	"mp":       "\u2213", // ∓
	"times":    "\u00d7", // ×
	"div":      "\u00f7", // ÷
	"cdot":     "\u00b7", // ·
	"ast":      "\u2217", // ∗
	"circ":     "\u2218", // ∘
	"cap":      "\u2229", // ∩
	"cup":      "\u222a", // ∪
	"setminus": "\u2216", // ∖
	"oplus":    "\u2295", // ⊕

	// Relations
	"le":       "\u2264", // ≤
	"leq":      "\u2264", // ≤
	"leqq":     "\u2266", // ≦
	"ge":       "\u2265", // ≥
	"geq":      "\u2265", // ≥
	"geqq":     "\u2267", // ≧
	"ne":       "\u2260", // ≠
	"neq":      "\u2260", // ≠
	"approx":   "\u2248", // ≈
	"sim":      "\u223c", // ∼
	"simeq":    "\u2243", // ≃
	"equiv":    "\u2261", // ≡
	"propto":   "\u221d", // ∝
	"in":       "\u2208", // ∈
	"notin":    "\u2209", // ∉
	"ni":       "\u220b", // ∋
	"subset":   "\u2282", // ⊂
	"subseteq": "\u2286", // ⊆
	"supset":   "\u2283", // ⊃
	"supseteq": "\u2287", // ⊇
	"mid":      "\u2223", // ∣
	"parallel": "\u2225", // ∥

	// Arrows
	"to":             "\u2192", // →
	"rightarrow":     "\u2192",
	"leftarrow":      "\u2190",
	"leftrightarrow": "\u2194",
	"Rightarrow":     "\u21d2",
	"Leftarrow":      "\u21d0",
	"Leftrightarrow": "\u21d4",
	"uparrow":        "\u2191",
	"downarrow":      "\u2193",
	"longrightarrow": "\u27f6",
	"longleftarrow":  "\u27f5",
	"mapsto":         "\u21a6",
	"implies":        "\u21d2", // ⇒
	"angle":          "\u2220", // ∠

	// Delimiters (growable ones are handled by the Nest renderer)
	"lfloor": "\u230a",
	"rfloor": "\u230b",
	"lceil":  "\u2308",
	"rceil":  "\u2309",
	"langle": "\u27e8",
	"rangle": "\u27e9",
	"vert":   "\u2016",
	"lbrace": "{",
	"rbrace": "}",
	"lbrack": "[",
	"rbrack": "]",

	// Misc
	"infty":       "\u221e", // ∞
	"prime":       "\u2032", // ′
	"partial":     "\u2202", // ∂
	"nabla":       "\u2207", // ∇
	"emptyset":    "\u2205", // ∅
	"cdotp":       "\u00b7",
	"ldots":       "\u2026", // …
	"cdots":       "\u22ef", // ⋯
	"dots":        "\u2026",
	"vdots":       "\u22ee", // ⋮
	"ddots":       "\u22f1", // ⋱
	"forall":      "\u2200", // ∀
	"exists":      "\u2203", // ∃
	"neg":         "\u00ac", // ¬
	"square":      "\u25a1",
	"blacksquare": "\u25a0",
	"triangle":    "\u25b3",
	"%":           "%",
}

// asciiMap is the fallback used when ASCII mode is enabled.
var asciiMap = map[string]string{
	"alpha":   "alpha",
	"beta":    "beta",
	"gamma":   "gamma",
	"delta":   "delta",
	"epsilon": "epsilon",
	"zeta":    "zeta",
	"theta":   "theta",
	"iota":    "iota",
	"kappa":   "kappa",
	"lambda":  "lambda",
	"mu":      "mu",
	"nu":      "nu",
	"xi":      "xi",
	"pi":      "pi",
	"rho":     "rho",
	"sigma":   "sigma",
	"tau":     "tau",
	"upsilon": "upsilon",
	"phi":     "phi",
	"chi":     "chi",
	"psi":     "psi",
	"omega":   "omega",

	"Gamma":   "Gamma",
	"Delta":   "Delta",
	"Theta":   "Theta",
	"Lambda":  "Lambda",
	"Xi":      "Xi",
	"Pi":      "Pi",
	"Sigma":   "Sigma",
	"Upsilon": "Upsilon",
	"Phi":     "Phi",
	"Psi":     "Psi",
	"Omega":   "Omega",

	"pm":       "+/-",
	"mp":       "-/+",
	"times":    "x",
	"div":      "/",
	"cdot":     "*",
	"ast":      "*",
	"circ":     "o",
	"cap":      "n",
	"cup":      "u",
	"setminus": "\\",
	"oplus":    "(+)",

	"le":       "<=",
	"leq":      "<=",
	"ge":       ">=",
	"geq":      ">=",
	"ne":       "!=",
	"neq":      "!=",
	"approx":   "~",
	"sim":      "~",
	"simeq":    "~",
	"equiv":    "==",
	"propto":   "~",
	"in":       "in",
	"notin":    "notin",
	"ni":       "contains",
	"subset":   "subset",
	"subseteq": "subset",
	"supset":   "supset",
	"supseteq": "supset",
	"mid":      "|",
	"parallel": "||",

	"to":             "->",
	"rightarrow":     "->",
	"leftarrow":      "<-",
	"leftrightarrow": "<->",
	"Rightarrow":     "=>",
	"Leftarrow":      "<=",
	"Leftrightarrow": "<=>",
	"uparrow":        "^",
	"downarrow":      "v",
	"longrightarrow": "->",
	"longleftarrow":  "<-",
	"mapsto":         "|->",
	"implies":        "=>",
	"angle":          "angle",

	"lfloor": "floor(",
	"rfloor": ")",
	"lceil":  "ceil(",
	"rceil":  ")",
	"langle": "<",
	"rangle": ">",
	"vert":   "||",
	"lbrace": "{",
	"rbrace": "}",
	"lbrack": "[",
	"rbrack": "]",

	"infty":       "inf",
	"prime":       "'",
	"partial":     "d",
	"nabla":       "nabla",
	"emptyset":    "{}",
	"ldots":       "...",
	"cdots":       "...",
	"dots":        "...",
	"forall":      "forall",
	"exists":      "exists",
	"neg":         "not",
	"square":      "[x]",
	"blacksquare": "[x]",
	"triangle":    "delta",
	"%":           "%",
}

// theseSkip are control words that are purely typographic and render nothing.
var skipWords = map[string]bool{
	"left": true, "right": true,
	"big": true, "Big": true, "bigg": true, "Bigg": true,
	"bigl": true, "bigr": true, "Bigl": true, "Bigr": true,
	"displaystyle": true, "textstyle": true, "scriptstyle": true,
	"scriptscriptstyle": true, "quad": true, "qquad": true,
	"hspace": true, "enspace": true, "thinspace": true,
	"noindent": true,
}

// functionWords render upright as text (e.g. sin, log, det).
var functionWords = map[string]bool{
	"log": true, "ln": true, "lg": true,
	"sin": true, "cos": true, "tan": true, "cot": true,
	"sec": true, "csc": true,
	"arcsin": true, "arccos": true, "arctan": true,
	"sinh": true, "cosh": true, "tanh": true, "coth": true,
	"min": true, "max": true, "inf": true, "sup": true,
	"lim": true, "det": true, "gcd": true, "arg": true,
	"exp": true, "Re": true, "Im": true,
	"mod": true, "Pr": true,
}

// bigOps are operators whose scripts become below/above limits.
var bigOps = map[string]string{
	"sum":  "\u2211", // ∑
	"prod": "\u220f", // ∏
	"int":  "\u222b", // ∫
	"iint": "\u222c", // ∬
	"lim":  "lim",
	"inf":  "inf",
	"sup":  "sup",
	"min":  "min",
	"max":  "max",
}

func resolveWord(name string, ascii bool) string {
	if ascii {
		if s, ok := asciiMap[name]; ok {
			return s
		}
	}
	if s, ok := symbolMap[name]; ok {
		return s
	}
	if s, ok := bigOps[name]; ok {
		return s
	}
	return ""
}

// hasSymbol reports whether name has a Unicode glyph in symbolMap.
func hasSymbol(name string) bool {
	_, ok := symbolMap[name]
	return ok
}

// bigOpSymbol resolves name against the big-operator table, returning its
// display symbol and whether it matched. In ASCII mode the fallback from
// asciiMap is used when present.
func bigOpSymbol(name string, ascii bool) (string, bool) {
	s, ok := bigOps[name]
	if !ok {
		return "", false
	}
	if ascii {
		if a, ok := asciiMap[name]; ok {
			return a, true
		}
	}
	return s, true
}

// unicode superscript/subscript glyphs for compact inline scripting.
var superDigits = map[rune]string{
	'0': "\u2070", '1': "\u00b9", '2': "\u00b2", '3': "\u00b3",
	'4': "\u2074", '5': "\u2075", '6': "\u2076", '7': "\u2077",
	'8': "\u2078", '9': "\u2079",
	'+': "\u207a", '-': "\u207b", '=': "\u207c", '(': "\u207d", ')': "\u207e",
	'n': "\u207f", 'i': "\u2071",
}
var subDigits = map[rune]string{
	'0': "\u2080", '1': "\u2081", '2': "\u2082", '3': "\u2083",
	'4': "\u2084", '5': "\u2085", '6': "\u2086", '7': "\u2087",
	'8': "\u2088", '9': "\u2089",
	'+': "\u208a", '-': "\u208b", '=': "\u208c", '(': "\u208d", ')': "\u208e",
	'a': "\u2090", 'e': "\u2091", 'i': "\u1d62", 'n': "\u2099",
	'o': "\u2092", 'r': "\u1d63", 's': "\u209b", 't': "\u209c",
	'u': "\u1d64", 'v': "\u1d65", 'x': "\u2093",
}

// compactScript reports the unicode superscript/subscript glyph of s when the
// whole script is representable on a single visual character.
func compactScript(s string, sub bool) (string, bool) {
	if s == "" {
		return "", false
	}
	m := superDigits
	if sub {
		m = subDigits
	}
	glyph := ""
	for _, r := range s {
		g, ok := m[r]
		if !ok {
			return "", false
		}
		glyph += g
	}
	return glyph, true
}
