// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package pinyin

// config holds conversion options.
type config struct {
	separator  string // separator between readings
	keepNonCJK bool   // keep non-Chinese characters in output
	allTones   bool   // include all readings (for Convert)
	uppercase  bool   // uppercase output
	titleCase  bool   // title case output
}

func defaultConfig() config {
	return config{
		separator:  " ",
		keepNonCJK: false,
		allTones:   false,
	}
}

// Option configures Convert / ConvertAll behavior.
type Option func(*config)

// WithSeparator sets the separator between pinyin readings.
// Default is a single space.
func WithSeparator(sep string) Option {
	return func(c *config) { c.separator = sep }
}

// WithKeepNonCJK keeps non-Chinese characters (letters, digits, punctuation)
// in the output instead of dropping them.
func WithKeepNonCJK(keep bool) Option {
	return func(c *config) { c.keepNonCJK = keep }
}

// WithAllTones makes Convert return all readings for multi-tone characters
// (joined by the separator), instead of just the first reading.
func WithAllTones(all bool) Option {
	return func(c *config) { c.allTones = all }
}

// WithUppercase converts all pinyin output to uppercase.
func WithUppercase(upper bool) Option {
	return func(c *config) { c.uppercase = upper }
}

// WithTitleCase converts pinyin output to title case (first letter uppercase).
func WithTitleCase(title bool) Option {
	return func(c *config) { c.titleCase = title }
}

// WithNoSeparator is a shortcut for WithSeparator("").
func WithNoSeparator() Option {
	return WithSeparator("")
}

// WithUnderscoreSeparator is a shortcut for WithSeparator("_").
func WithUnderscoreSeparator() Option {
	return WithSeparator("_")
}

// WithHyphenSeparator is a shortcut for WithSeparator("-").
func WithHyphenSeparator() Option {
	return WithSeparator("-")
}
