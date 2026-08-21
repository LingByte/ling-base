package goagent

import (
	"regexp"
	"strings"
	"unicode"
)

type QueryType int

const (
	QueryComplex QueryType = iota
	QueryShortFactoid
	QueryMath
)

var (
	simpleMathRegex = regexp.MustCompile(`^\s*\d+(\s*[\+\-\*\/]\s*\d+)+\s*$`)
	shortFactRegex  = regexp.MustCompile(`^[\p{L}\p{N}\s\?\!\.\,\-]{1,32}$`)
)

// classifyQuery analyzes input and returns a QueryType
func classifyQuery(input string) QueryType {
	in := strings.TrimSpace(input)
	if in == "" {
		return QueryComplex
	}

	// 🧮 1. Math expressions → skip memory
	if simpleMathRegex.MatchString(in) {
		return QueryMath
	}

	// 💬 2. Short factoid — short length, simple text structure
	if len([]rune(in)) <= 32 && shortFactRegex.MatchString(in) {
		return QueryShortFactoid
	}

	// 💬 3. Single word heuristic also → short factoid
	if len(strings.FieldsFunc(in, unicode.IsSpace)) == 1 {
		return QueryShortFactoid
	}

	// 🧠 4. Default — complex query, keep full memory
	return QueryComplex
}
