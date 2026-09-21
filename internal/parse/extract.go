// Package parse extracts the final answer from a tutor model's raw
// completion output. It mirrors the extraction stage of evals/answer_assert.js
// (deliberately without that script's normalization/equivalence fallbacks):
// first a \boxed{...} literal, then "final answer"/"answer:" trailing text,
// then a GSM8K-style "#### ..." tail.
package parse

import (
	"regexp"
	"strings"
)

var (
	boxedRE  = regexp.MustCompile(`\\boxed\s*\{((?:[^{}]|\{[^{}]*\})*)\}`)
	markerRE = regexp.MustCompile(`(?i)(?:final answer(?:\s*is)?|answer is|answer:)\s*(.+)`)
	gsm8kRE  = regexp.MustCompile(`####\s*(\S+.*)`)
)

// Rule names the extraction stage that produced an answer. "none" means no
// marker was found and the answer is the whole-output fallback.
const (
	RuleBoxed = "boxed"
	RuleFinal = "final"
	RuleGSM8K = "gsm8k"
	RuleNone  = "none"
)

// Extract returns the model's final answer from output or "" when nothing
// can be parsed.
func Extract(output string) string {
	answer, _ := ExtractRule(output)
	return answer
}

// ExtractRule returns the final answer plus the rule that matched it, so
// callers (e.g. the web UI's Final badge) can hide whole-output fallbacks.
func ExtractRule(output string) (answer, rule string) {
	if m := boxedRE.FindStringSubmatch(output); m != nil {
		return strings.TrimSpace(m[1]), RuleBoxed
	}
	if m := markerRE.FindStringSubmatch(output); m != nil {
		return strings.TrimSpace(m[1]), RuleFinal
	}
	if m := gsm8kRE.FindStringSubmatch(output); m != nil {
		return strings.TrimSpace(m[1]), RuleGSM8K
	}
	return strings.TrimSpace(output), RuleNone
}
