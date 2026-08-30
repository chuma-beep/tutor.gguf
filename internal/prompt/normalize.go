package prompt

import (
	"os"
	"regexp"
	"strings"
)

// NumberWordsEnabled controls whether word-numbers are normalized to digits
// before prompt construction. Default off. Desktop toggles it via
// SetNumberWords; CLI env TUTOR_NUMBER_WORDS=1 enables at startup.
var NumberWordsEnabled = initNumberWordsEnabled()

func initNumberWordsEnabled() bool {
	v := os.Getenv("TUTOR_NUMBER_WORDS")
	return v == "1" || strings.EqualFold(v, "true")
}

// SetNumberWords toggles word-number normalization at runtime (Wails binding).
func SetNumberWords(enabled bool) { NumberWordsEnabled = enabled }

var numberWordMap = map[string]string{
	"zero": "0", "one": "1", "two": "2", "three": "3", "four": "4",
	"five": "5", "six": "6", "seven": "7", "eight": "8", "nine": "9",
	"ten": "10", "eleven": "11", "twelve": "12", "thirteen": "13",
	"fourteen": "14", "fifteen": "15", "sixteen": "16", "seventeen": "17",
	"eighteen": "18", "nineteen": "19",
	"twenty": "20", "thirty": "30", "forty": "40", "fifty": "50",
	"sixty": "60", "seventy": "70", "eighty": "80", "ninety": "90",
}

var compoundTens = map[string]string{
	"twenty": "20", "thirty": "30", "forty": "40", "fifty": "50",
	"sixty": "60", "seventy": "70", "eighty": "80", "ninety": "90",
}

var operatorMap = map[string]string{
	"plus":       "+",
	"minus":      "-",
	"times":      "*",
	"multiplied": "*",
	"divided":    "/",
	"over":       "/",
	"equals":     "=",
	"squared":    "^2",
	"cubed":      "^3",
	"half":       "1/2",
	"quarter":    "1/4",
}

var (
	// Hyphenated compounds: twenty-one -> 21
	reHyphenCompound = regexp.MustCompile(`(?i)\b(twenty|thirty|forty|fifty|sixty|seventy|eighty|ninety)[\s-]+(one|two|three|four|five|six|seven|eight|nine)\b`)
	reDividedBy      = regexp.MustCompile(`(?i)\bdivided\s+by\b`)
	reIsEqualTo     = regexp.MustCompile(`(?i)\bis\s+equal\s+to\b`)
	// Word boundaries for single tokens - built dynamically
)

func buildWordPattern(words []string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\b(` + strings.Join(words, "|") + `)\b`)
}

var (
	reNumberWords = buildWordPattern([]string{
		"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
		"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen",
		"twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety",
	})
	reOperators = buildWordPattern([]string{
		"plus", "minus", "times", "divided", "over", "equals", "squared", "cubed", "half", "quarter", "multiplied",
	})
)

// NormalizeNumberWords converts English number words and operator words to
// symbolic form when enabled. It is case-insensitive and preserves surrounding
// punctuation. Example: "what is one plus one" -> "what is 1 + 1".
func NormalizeNumberWords(s string) string {
	if s == "" {
		return s
	}
	// Handle multi-word operators first
	s = reIsEqualTo.ReplaceAllString(s, "=")
	s = reDividedBy.ReplaceAllString(s, "/")

	// Hyphenated/space compounds: twenty one -> 21, etc.
	s = reHyphenCompound.ReplaceAllStringFunc(s, func(m string) string {
		parts := regexp.MustCompile(`(?i)[\s-]+`).Split(strings.ToLower(m), -1)
		if len(parts) != 2 {
			return m
		}
		tens, ok1 := compoundTens[parts[0]]
		ones, ok2 := numberWordMap[parts[1]]
		if !ok1 || !ok2 {
			return m
		}
		// tens "20" + ones "1" -> "21": tens[0]='2' + ones="1"
		return string(tens[0]) + ones
	})

	// Single number words
	s = reNumberWords.ReplaceAllStringFunc(s, func(m string) string {
		if v, ok := numberWordMap[strings.ToLower(m)]; ok {
			return v
		}
		return m
	})

	// Operator words (after numbers so "one" doesn't interfere)
	s = reOperators.ReplaceAllStringFunc(s, func(m string) string {
		lower := strings.ToLower(m)
		// "multiplied" was mapped from "multiplied by" split? Keep as *
		if lower == "multiplied" {
			// Check if next word is "by" was already handled? reDividedBy handled "divided by"
			return "*"
		}
		if v, ok := operatorMap[lower]; ok {
			return v
		}
		return m
	})
	// Fix "multiplied by" -> we mapped "multiplied" to "*" but left " by"
	// Clean up "* by" -> "*"
	s = regexp.MustCompile(`(?i)\*\s+by\b`).ReplaceAllString(s, "*")
	return s
}
