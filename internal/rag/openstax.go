package rag

import (
	"fmt"
	"io/fs"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

// openstaxMaxChunkChars matches the Rosen budget: 2000 chars is about 600
// dense-LaTeX tokens, safely inside the 8192-token physical embed batch.
const openstaxMaxChunkChars = 2000

// headingRE splits extracted PDF text into sections on chapter and section
// headings (e.g. "Chapter 3", "Section 2.1", "1.2 Limits"). It is
// intentionally loose: OpenStax layouts vary, and the fallback below keeps
// every byte even when no heading matches.
var headingRE = regexp.MustCompile(`(?m)^(?:Chapter\s+\d+.*|Section\s+[\d.]+.*|\d+(?:\.\d+)+\s+\S.*)$`)

// LoadOpenStaxPDF extracts text from one OpenStax PDF and returns one Chunk
// per section (split further on line boundaries when a section exceeds
// openstaxMaxChunkChars). Source is always "openstax"; subdomain is the
// caller's domain label (e.g. "calculus" or "algebra").
func LoadOpenStaxPDF(filePath string, subdomain string) ([]Chunk, error) {
	raw, err := extractPDFText(filePath)
	if err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("openstax PDF %s yielded no extractable text (scanned-image PDFs need OCR)", filePath)
	}
	if subdomain == "" {
		subdomain = "general_math"
	}
	var chunks []Chunk
	for _, section := range splitSections(raw) {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}
		for _, seg := range splitOnLines(section, openstaxMaxChunkChars) {
			chunks = append(chunks, Chunk{
				Text:      seg,
				Subdomain: subdomain,
				Source:    "openstax",
			})
		}
	}
	return chunks, nil
}

// LoadOpenStaxDir walks a directory (recursively) for .pdf files and applies
// LoadOpenStaxPDF to each. Files whose names hint at a subdomain (e.g.
// "calculus-vol1.pdf") use the inferred label; everything else falls back to
// defaultSubdomain. Non-PDF and empty files are skipped.
func LoadOpenStaxDir(dirPath string, defaultSubdomain string) ([]Chunk, error) {
	if defaultSubdomain == "" {
		defaultSubdomain = "general_math"
	}
	var chunks []Chunk
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path == dirPath {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".pdf") {
			return nil
		}
		sub := subdomainForOpenStaxFile(d.Name(), defaultSubdomain)
		cs, err := LoadOpenStaxPDF(path, sub)
		if err != nil {
			return fmt.Errorf("%s: %w", d.Name(), err)
		}
		chunks = append(chunks, cs...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenStax directory: %w", err)
	}
	return chunks, nil
}

// subdomainForOpenStaxFile infers a retrieval subdomain from the PDF file
// name so a mixed openstax/ directory (college algebra + calculus) filters
// correctly. Unknown names fall back to def.
func subdomainForOpenStaxFile(name string, def string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "calcul"):
		return "calculus"
	case strings.Contains(lower, "linear"):
		return "algebra"
	case strings.Contains(lower, "algebra"):
		return "algebra"
	case strings.Contains(lower, "precalc"):
		return "precalculus"
	case strings.Contains(lower, "geometr"):
		return "geometry"
	case strings.Contains(lower, "probab"), strings.Contains(lower, "statist"):
		return "probability"
	case strings.Contains(lower, "number"):
		return "number_theory"
	case strings.Contains(lower, "discrete"):
		return "discrete_math"
	default:
		return def
	}
}

// extractPDFText opens filePath with ledongthuc/pdf and returns the document
// text with one line per typeset row, pages separated by blank lines. It
// uses the positioned Content() interpreter (which honors Td/Tm/T*) rather
// than GetTextByRow, whose walker ignores Td offsets and collapses
// Td-positioned lines into a single row.
func extractPDFText(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open OpenStax PDF: %w", err)
	}
	defer f.Close()
	var sb strings.Builder
	n := r.NumPage()
	for i := 1; i <= n; i++ {
		sb.WriteString(pageLines(r.Page(i)))
		sb.WriteByte('\n')
	}
	return sb.String(), nil
}

// pageLines renders one page's positioned glyphs as lines. Glyphs are
// grouped by Y coordinate (3pt tolerance covers sub/superscript jitter
// without merging adjacent 12pt+ rows) and ordered top-to-bottom,
// left-to-right within a row.
func pageLines(p pdf.Page) string {
	glyphs := p.Content().Text
	if len(glyphs) == 0 {
		plain, err := p.GetPlainText(nil)
		if err != nil {
			return ""
		}
		return strings.TrimRight(plain, "\n") + "\n"
	}
	sort.Slice(glyphs, func(a, b int) bool {
		if math.Abs(glyphs[a].Y-glyphs[b].Y) > 3 {
			return glyphs[a].Y > glyphs[b].Y
		}
		return glyphs[a].X < glyphs[b].X
	})
	var sb strings.Builder
	rowY := math.MaxFloat64
	var row strings.Builder
	prevX, prevW := 0.0, 0.0
	flush := func() {
		if s := strings.TrimRight(row.String(), " "); s != "" {
			sb.WriteString(s)
			sb.WriteByte('\n')
		}
		row.Reset()
		prevX, prevW = 0, 0
	}
	for _, g := range glyphs {
		if g.S == "" {
			continue
		}
		if math.Abs(g.Y-rowY) > 3 {
			flush()
			rowY = g.Y
		} else if g.S != " " && row.Len() > 0 {
			// Real PDFs often encode inter-word gaps as cursor advances
			// with no space glyph. Infer a space from a wide X jump.
			if gap := g.X - (prevX + prevW); gap > math.Max(2, g.FontSize*0.2) {
				row.WriteByte(' ')
			}
		}
		row.WriteString(g.S)
		prevX, prevW = g.X, g.W
	}
	flush()
	return sb.String()
}

// splitSections cuts raw text on chapter/section headings, keeping the
// heading with its body. When nothing matches, the whole text is one
// section so no content is lost.
func splitSections(raw string) []string {
	locs := headingRE.FindAllStringIndex(raw, -1)
	if len(locs) == 0 {
		return []string{raw}
	}
	var sections []string
	// Keep any front matter before the first heading as its own section.
	if locs[0][0] > 0 {
		if head := strings.TrimSpace(raw[:locs[0][0]]); head != "" {
			sections = append(sections, head)
		}
	}
	for i, loc := range locs {
		end := len(raw)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		if seg := strings.TrimSpace(raw[loc[0]:end]); seg != "" {
			sections = append(sections, seg)
		}
	}
	return sections
}
