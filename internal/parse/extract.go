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

// Extract returns the model's final answer from output, or "" when nothing
// can be parsed.
func Extract(output string) string {
	if m := boxedRE.FindStringSubmatch(output); m != nil {
		return strings.TrimSpace(m[1])
	}
	if m := markerRE.FindStringSubmatch(output); m != nil {
		return strings.TrimSpace(m[1])
	}
	if m := gsm8kRE.FindStringSubmatch(output); m != nil {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(output)
}
