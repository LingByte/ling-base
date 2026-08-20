package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/LingByte/ling-base/common/parser"
)

// FidelityReport records how faithfully the parser extracted content from a file.
type FidelityReport struct {
	FileName   string `json:"fileName"`
	FileType   string `json:"fileType"`
	FileSize   int64  `json:"fileSize"`
	TextLength int    `json:"textLength"`
	RuneCount  int    `json:"runeCount"`

	// ExtractionRatio = textLength / fileSize. For text formats this should be
	// close to 1.0; for binary formats (docx, xlsx, epub, pdf) it will be lower
	// because the raw bytes include formatting/encoding overhead.
	ExtractionRatio float64 `json:"extractionRatio"`

	// FidelityScore is 0-100, a composite score from all checks.
	FidelityScore float64 `json:"fidelityScore"`
	FidelityGrade string  `json:"fidelityGrade"`

	// Checks
	Checks []FidelityCheck `json:"checks"`

	// Issues found (non-fatal problems).
	Issues []string `json:"issues,omitempty"`

	// KeyTerms found in the parsed text vs expected from the original file.
	KeyTermsFound    int      `json:"keyTermsFound"`
	KeyTermsExpected int      `json:"keyTermsExpected"`
	KeyTermsMissing  []string `json:"keyTermsMissing,omitempty"`
}

// FidelityCheck represents a single content fidelity verification.
type FidelityCheck struct {
	Name   string  `json:"name"`
	Passed bool    `json:"passed"`
	Score  float64 `json:"score"`
	Detail string  `json:"detail"`
}

// analyzeFidelity compares the original file content with the parsed result
// to determine how faithfully the content was extracted.
func analyzeFidelity(path string, res *parser.ParseResult, raw []byte) *FidelityReport {
	info, _ := os.Stat(path)
	ft := ""
	if res != nil {
		ft = res.FileType
	}

	r := &FidelityReport{
		FileName: baseName(path),
		FileType: ft,
		FileSize: info.Size(),
	}

	if res != nil {
		r.TextLength = len(res.Text)
		r.RuneCount = len([]rune(res.Text))
	}

	if info.Size() > 0 {
		r.ExtractionRatio = float64(r.TextLength) / float64(info.Size())
	}

	// Run all checks.
	r.Checks = runFidelityChecks(path, ft, raw, res)

	// Extract key terms from the original file and check if they appear in parsed text.
	keyTerms := extractKeyTerms(raw, ft)
	r.KeyTermsExpected = len(keyTerms)
	if res != nil {
		for _, term := range keyTerms {
			if strings.Contains(res.Text, term) {
				r.KeyTermsFound++
			} else {
				r.KeyTermsMissing = append(r.KeyTermsMissing, term)
			}
		}
	}

	// Compute composite score.
	var totalWeight, totalScore float64
	for _, c := range r.Checks {
		weight := checkWeight(c.Name)
		totalWeight += weight
		totalScore += c.Score * weight
	}
	if totalWeight > 0 {
		r.FidelityScore = totalScore / totalWeight
	}
	r.FidelityGrade = scoreToGrade(r.FidelityScore)

	// Collect issues.
	for _, c := range r.Checks {
		if !c.Passed && c.Detail != "" {
			r.Issues = append(r.Issues, fmt.Sprintf("%s: %s", c.Name, c.Detail))
		}
	}
	if r.KeyTermsExpected > 0 && r.KeyTermsFound < r.KeyTermsExpected {
		r.Issues = append(r.Issues, fmt.Sprintf("KeyTerms: %d/%d found, missing: %s",
			r.KeyTermsFound, r.KeyTermsExpected, strings.Join(r.KeyTermsMissing, ", ")))
	}

	return r
}

// runFidelityChecks executes all individual fidelity checks.
func runFidelityChecks(path, ft string, raw []byte, res *parser.ParseResult) []FidelityCheck {
	var checks []FidelityCheck

	// 1. Non-empty extraction
	checks = append(checks, checkNonEmpty(res))

	// 2. Valid UTF-8
	checks = append(checks, checkValidUTF8(res))

	// 3. No binary garbage
	checks = append(checks, checkNoBinaryGarbage(res))

	// 4. Extraction ratio reasonable for file type
	checks = append(checks, checkExtractionRatio(ft, res, len(raw)))

	// 5. Key term coverage
	checks = append(checks, checkKeyTermCoverage(raw, ft, res))

	// 6. Structural integrity (format-specific)
	checks = append(checks, checkStructuralIntegrity(ft, raw, res))

	// 7. No excessive repetition (detect parsing loops)
	checks = append(checks, checkNoExcessiveRepetition(res))

	// 8. Section count reasonable
	checks = append(checks, checkSectionCount(ft, res))

	return checks
}

func checkNonEmpty(res *parser.ParseResult) FidelityCheck {
	c := FidelityCheck{Name: "NonEmpty", Score: 0}
	if res == nil || strings.TrimSpace(res.Text) == "" {
		c.Passed = false
		c.Detail = "parsed text is empty"
		return c
	}
	c.Passed = true
	c.Score = 100
	c.Detail = fmt.Sprintf("%d chars extracted", len(res.Text))
	return c
}

func checkValidUTF8(res *parser.ParseResult) FidelityCheck {
	c := FidelityCheck{Name: "ValidUTF8", Score: 0}
	if res == nil {
		c.Passed = false
		c.Detail = "no result"
		return c
	}
	if !utf8.ValidString(res.Text) {
		c.Passed = false
		c.Detail = "parsed text contains invalid UTF-8 sequences"
		return c
	}
	c.Passed = true
	c.Score = 100
	c.Detail = "all valid UTF-8"
	return c
}

func checkNoBinaryGarbage(res *parser.ParseResult) FidelityCheck {
	c := FidelityCheck{Name: "NoBinaryGarbage", Score: 0}
	if res == nil {
		c.Passed = false
		c.Detail = "no result"
		return c
	}
	text := res.Text
	if len(text) == 0 {
		c.Passed = true
		c.Score = 100
		return c
	}
	// Count printable vs non-printable runes.
	var printable, total int
	for _, r := range text {
		total++
		if r >= 32 || r == '\n' || r == '\r' || r == '\t' || r >= 0x4e00 {
			printable++
		}
	}
	ratio := float64(printable) / float64(total)
	if ratio < 0.8 {
		c.Passed = false
		c.Score = ratio * 100
		c.Detail = fmt.Sprintf("only %.1f%% printable characters", ratio*100)
		return c
	}
	c.Passed = true
	c.Score = 100
	c.Detail = fmt.Sprintf("%.1f%% printable", ratio*100)
	return c
}

func checkExtractionRatio(ft string, res *parser.ParseResult, rawSize int) FidelityCheck {
	c := FidelityCheck{Name: "ExtractionRatio", Score: 0}
	if res == nil || rawSize == 0 {
		c.Passed = false
		c.Detail = "no data"
		return c
	}
	ratio := float64(len(res.Text)) / float64(rawSize)

	// Expected ratios vary by format.
	var minRatio, maxRatio float64
	switch ft {
	case parser.FileTypeTXT, parser.FileTypeMD, parser.FileTypeCSV, parser.FileTypeTSV,
		parser.FileTypeJSON, parser.FileTypeYAML, parser.FileTypeYML, parser.FileTypeLOG,
		parser.FileTypeTOML, parser.FileTypeICS, parser.FileTypeVCF,
		parser.FileTypeEML:
		// Text formats: most content should be extracted.
		minRatio, maxRatio = 0.3, 1.5
	case parser.FileTypeXML, parser.FileTypeSVG:
		// XML/SVG: tags are stripped, only text nodes remain.
		minRatio, maxRatio = 0.05, 1.5
	case parser.FileTypeHTML, parser.FileTypeMHTML, parser.FileTypeMHT:
		// HTML: tags are stripped, so ratio is lower.
		minRatio, maxRatio = 0.1, 1.0
	case parser.FileTypeRTF:
		// RTF: control words are stripped.
		minRatio, maxRatio = 0.05, 1.0
	case parser.FileTypeDOCX, parser.FileTypeXLSX, parser.FileTypePPTX, parser.FileTypeEPUB,
		parser.FileTypeODT, parser.FileTypeODS, parser.FileTypeODP:
		// Zip-based: raw includes XML overhead + zip compression.
		minRatio, maxRatio = 0.01, 2.0
	case parser.FileTypePDF, parser.FileTypeDOC:
		// Binary: extraction ratio varies widely.
		minRatio, maxRatio = 0.0, 5.0
	default:
		minRatio, maxRatio = 0.0, 5.0
	}

	if ratio >= minRatio && ratio <= maxRatio {
		c.Passed = true
		c.Score = 100
		c.Detail = fmt.Sprintf("ratio %.2f within expected [%.2f, %.2f]", ratio, minRatio, maxRatio)
	} else {
		c.Passed = false
		c.Score = 50
		c.Detail = fmt.Sprintf("ratio %.2f outside expected [%.2f, %.2f]", ratio, minRatio, maxRatio)
	}
	return c
}

func checkKeyTermCoverage(raw []byte, ft string, res *parser.ParseResult) FidelityCheck {
	c := FidelityCheck{Name: "KeyTermCoverage", Score: 0}
	terms := extractKeyTerms(raw, ft)
	if len(terms) == 0 {
		c.Passed = true
		c.Score = 100
		c.Detail = "no key terms to check"
		return c
	}
	if res == nil {
		c.Passed = false
		c.Score = 0
		c.Detail = "no result"
		return c
	}
	found := 0
	for _, t := range terms {
		if strings.Contains(res.Text, t) {
			found++
		}
	}
	coverage := float64(found) / float64(len(terms))
	c.Score = coverage * 100
	c.Passed = coverage >= 0.5
	c.Detail = fmt.Sprintf("%d/%d key terms found (%.0f%%)", found, len(terms), coverage*100)
	return c
}

func checkStructuralIntegrity(ft string, raw []byte, res *parser.ParseResult) FidelityCheck {
	c := FidelityCheck{Name: "StructuralIntegrity", Score: 0}
	if res == nil {
		c.Passed = false
		c.Detail = "no result"
		return c
	}

	switch ft {
	case parser.FileTypeJSON:
		// Round-trip: re-parse the extracted text as JSON.
		var v any
		if err := json.Unmarshal([]byte(res.Text), &v); err == nil {
			c.Passed = true
			c.Score = 100
			c.Detail = "JSON round-trip valid"
		} else {
			c.Passed = false
			c.Score = 30
			c.Detail = "JSON round-trip failed: " + err.Error()
		}
	case parser.FileTypeXML, parser.FileTypeSVG:
		// Check that the number of text nodes roughly matches.
		rawTextNodes := countXMLTextNodes(raw)
		if rawTextNodes > 0 {
			// Use word count instead of line count (lines may be collapsed).
			parsedWords := len(strings.Fields(res.Text))
			ratio := float64(parsedWords) / float64(rawTextNodes)
			if ratio > 0.2 && ratio < 10.0 {
				c.Passed = true
				c.Score = 100
				c.Detail = fmt.Sprintf("text nodes %d → %d words (ratio %.1f)", rawTextNodes, parsedWords, ratio)
			} else {
				c.Passed = false
				c.Score = 50
				c.Detail = fmt.Sprintf("text nodes %d → %d words (ratio %.1f)", rawTextNodes, parsedWords, ratio)
			}
		} else {
			c.Passed = true
			c.Score = 100
			c.Detail = "no text nodes to compare"
		}
	case parser.FileTypeCSV, parser.FileTypeTSV:
		// Check that the number of data rows is preserved.
		rawLines := countNonEmptyLines(raw)
		// When PreserveLineBreaks is off, text is collapsed to one line.
		// Use word count comparison instead.
		rawWords := len(strings.Fields(string(raw)))
		parsedWords := len(strings.Fields(res.Text))
		if rawLines > 0 && rawWords > 0 {
			ratio := float64(parsedWords) / float64(rawWords)
			// CSV/TSV may expand cells into more words (tabs join differently).
			if ratio > 0.3 && ratio < 5.0 {
				c.Passed = true
				c.Score = 100
				c.Detail = fmt.Sprintf("words %d → %d (ratio %.1f), rows %d", rawWords, parsedWords, ratio, rawLines)
			} else {
				c.Passed = false
				c.Score = 50
				c.Detail = fmt.Sprintf("words %d → %d (ratio %.1f)", rawWords, parsedWords, ratio)
			}
		} else {
			c.Passed = true
			c.Score = 100
		}
	case parser.FileTypeICS:
		// Count VEVENT blocks in raw vs sections in result.
		rawEvents := strings.Count(strings.ToUpper(string(raw)), "BEGIN:VEVENT")
		if rawEvents > 0 && res != nil {
			if strings.Contains(res.Text, "Event") || len(res.Sections) >= 1 {
				c.Passed = true
				c.Score = 100
				c.Detail = fmt.Sprintf("%d VEVENT blocks processed", rawEvents)
			} else {
				c.Passed = false
				c.Score = 30
				c.Detail = "VEVENT blocks found but no events in output"
			}
		} else {
			c.Passed = true
			c.Score = 100
			c.Detail = "no events to verify"
		}
	case parser.FileTypeVCF:
		rawCards := strings.Count(strings.ToUpper(string(raw)), "BEGIN:VCARD")
		if rawCards > 0 && res != nil {
			if strings.Contains(res.Text, "Contact") {
				c.Passed = true
				c.Score = 100
				c.Detail = fmt.Sprintf("%d VCARD blocks processed", rawCards)
			} else {
				c.Passed = false
				c.Score = 30
				c.Detail = "VCARD blocks found but no contacts in output"
			}
		} else {
			c.Passed = true
			c.Score = 100
		}
	case parser.FileTypeLOG:
		// Check that log content is preserved (word count comparison,
		// since line breaks may be collapsed).
		rawWords := len(strings.Fields(string(raw)))
		parsedWords := len(strings.Fields(res.Text))
		if rawWords > 0 {
			ratio := float64(parsedWords) / float64(rawWords)
			if ratio > 0.8 && ratio < 1.2 {
				c.Passed = true
				c.Score = 100
				c.Detail = fmt.Sprintf("words %d → %d (ratio %.1f)", rawWords, parsedWords, ratio)
			} else {
				c.Passed = false
				c.Score = 60
				c.Detail = fmt.Sprintf("words %d → %d (ratio %.1f)", rawWords, parsedWords, ratio)
			}
		} else {
			c.Passed = true
			c.Score = 100
		}
	default:
		c.Passed = true
		c.Score = 100
		c.Detail = "no structural check for this format"
	}
	return c
}

func checkNoExcessiveRepetition(res *parser.ParseResult) FidelityCheck {
	c := FidelityCheck{Name: "NoExcessiveRepetition", Score: 0}
	if res == nil || len(res.Text) == 0 {
		c.Passed = true
		c.Score = 100
		c.Detail = "no text to check"
		return c
	}
	// Check if any single line repeats more than 10 times consecutively.
	lines := strings.Split(res.Text, "\n")
	maxRepeat := 1
	currentRepeat := 1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == strings.TrimSpace(lines[i-1]) && strings.TrimSpace(lines[i]) != "" {
			currentRepeat++
			if currentRepeat > maxRepeat {
				maxRepeat = currentRepeat
			}
		} else {
			currentRepeat = 1
		}
	}
	if maxRepeat > 10 {
		c.Passed = false
		c.Score = 30
		c.Detail = fmt.Sprintf("line repeated %d times consecutively (possible parsing loop)", maxRepeat)
	} else {
		c.Passed = true
		c.Score = 100
		c.Detail = fmt.Sprintf("max consecutive repeat: %d", maxRepeat)
	}
	return c
}

func checkSectionCount(ft string, res *parser.ParseResult) FidelityCheck {
	c := FidelityCheck{Name: "SectionCount", Score: 0}
	if res == nil {
		c.Passed = false
		c.Detail = "no result"
		return c
	}
	count := len(res.Sections)
	// Most formats should have at least 1 section.
	if count < 1 {
		c.Passed = false
		c.Score = 0
		c.Detail = "no sections"
		return c
	}
	// For multi-page formats (pdf, epub, pptx, xlsx), more sections is expected.
	switch ft {
	case parser.FileTypePDF, parser.FileTypeEPUB, parser.FileTypePPTX, parser.FileTypeXLSX:
		if count >= 1 {
			c.Passed = true
			c.Score = 100
			c.Detail = fmt.Sprintf("%d sections", count)
		}
	default:
		c.Passed = true
		c.Score = 100
		c.Detail = fmt.Sprintf("%d sections", count)
	}
	return c
}

// extractKeyTerms pulls significant terms from the raw file content that should
// appear in the parsed output. For text formats, these are meaningful words.
// For structured formats, these are values from key fields.
func extractKeyTerms(raw []byte, ft string) []string {
	rawStr := string(raw)
	var terms []string

	switch ft {
	case parser.FileTypeJSON:
		// Extract string values from JSON.
		re := regexp.MustCompile(`"([^"]{3,})"`)
		matches := re.FindAllStringSubmatch(rawStr, -1)
		for _, m := range matches {
			val := strings.TrimSpace(m[1])
			// Skip keys (heuristic: keys are typically short, snake_case).
			if len(val) > 3 && !strings.Contains(val, "_") {
				terms = append(terms, val)
			}
		}
	case parser.FileTypeYAML, parser.FileTypeYML:
		// Extract values after colons.
		for _, line := range strings.Split(rawStr, "\n") {
			if idx := strings.Index(line, ":"); idx >= 0 {
				val := strings.TrimSpace(line[idx+1:])
				val = strings.Trim(val, `"'`)
				if len(val) > 3 {
					terms = append(terms, val)
				}
			}
		}
	case parser.FileTypeXML, parser.FileTypeSVG:
		// Extract text between tags.
		re := regexp.MustCompile(`>([^<]{3,})<`)
		matches := re.FindAllStringSubmatch(rawStr, -1)
		for _, m := range matches {
			val := strings.TrimSpace(m[1])
			if len(val) > 2 {
				terms = append(terms, val)
			}
		}
	case parser.FileTypeTOML:
		// Extract values after =.
		for _, line := range strings.Split(rawStr, "\n") {
			if idx := strings.Index(line, "="); idx >= 0 {
				val := strings.TrimSpace(line[idx+1:])
				val = strings.Trim(val, `"'`)
				if len(val) > 3 {
					terms = append(terms, val)
				}
			}
		}
	case parser.FileTypeICS:
		// Extract SUMMARY and DESCRIPTION values.
		for _, line := range strings.Split(rawStr, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToUpper(line), "SUMMARY:") {
				terms = append(terms, strings.TrimSpace(line[len("SUMMARY:"):]))
			}
			if strings.HasPrefix(strings.ToUpper(line), "DESCRIPTION:") {
				terms = append(terms, strings.TrimSpace(line[len("DESCRIPTION:"):]))
			}
		}
	case parser.FileTypeVCF:
		// Extract FN values.
		for _, line := range strings.Split(rawStr, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToUpper(line), "FN:") {
				terms = append(terms, strings.TrimSpace(line[len("FN:"):]))
			}
		}
	case parser.FileTypeHTML, parser.FileTypeMHTML, parser.FileTypeMHT:
		// Extract text from common elements.
		re := regexp.MustCompile(`>([^<]{3,})<`)
		matches := re.FindAllStringSubmatch(rawStr, -1)
		for _, m := range matches {
			val := strings.TrimSpace(m[1])
			if len(val) > 2 {
				terms = append(terms, val)
			}
		}
	case parser.FileTypeCSV, parser.FileTypeTSV:
		// Extract cell values.
		sep := ","
		if ft == parser.FileTypeTSV {
			sep = "\t"
		}
		for _, line := range strings.Split(rawStr, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			for _, cell := range strings.Split(line, sep) {
				cell = strings.TrimSpace(cell)
				if len(cell) > 3 {
					terms = append(terms, cell)
				}
			}
		}
	default:
		// For text-based formats (txt, md, log, etc.), extract significant words.
		words := strings.Fields(rawStr)
		seen := make(map[string]bool)
		for _, w := range words {
			w = strings.Trim(w, ".,;:!?\"'()[]{}")
			if len(w) > 4 && !seen[w] && !isCommonWord(w) {
				terms = append(terms, w)
				seen[w] = true
			}
			if len(terms) >= 10 {
				break
			}
		}
	}

	// Deduplicate and limit.
	seen := make(map[string]bool)
	var unique []string
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		unique = append(unique, t)
		if len(unique) >= 15 {
			break
		}
	}
	return unique
}

var commonWords = map[string]bool{
	"the": true, "and": true, "for": true, "are": true, "but": true,
	"not": true, "you": true, "all": true, "can": true, "her": true,
	"was": true, "one": true, "our": true, "out": true, "has": true,
	"have": true, "from": true, "this": true, "that": true, "with": true,
	"they": true, "will": true, "each": true, "make": true, "like": true,
	"been": true, "more": true, "some": true, "them": true, "than": true,
	"its": true, "over": true, "into": true,
}

func isCommonWord(w string) bool {
	return commonWords[strings.ToLower(w)]
}

func countXMLTextNodes(data []byte) int {
	count := 0
	re := regexp.MustCompile(`>([^<]{2,})<`)
	matches := re.FindAllSubmatch(data, -1)
	for _, m := range matches {
		if len(strings.TrimSpace(string(m[1]))) > 0 {
			count++
		}
	}
	return count
}

func countNonEmptyLines(data []byte) int {
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func checkWeight(name string) float64 {
	weights := map[string]float64{
		"NonEmpty":              2.0,
		"ValidUTF8":             1.5,
		"NoBinaryGarbage":       1.5,
		"ExtractionRatio":       1.0,
		"KeyTermCoverage":       3.0,
		"StructuralIntegrity":   2.0,
		"NoExcessiveRepetition": 1.0,
		"SectionCount":          0.5,
	}
	if w, ok := weights[name]; ok {
		return w
	}
	return 1.0
}

func scoreToGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

func baseName(path string) string {
	idx := strings.LastIndexByte(path, '/')
	if idx < 0 {
		return path
	}
	return path[idx+1:]
}
