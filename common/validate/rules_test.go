// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// required
// ──────────────────────────────────────────────

func TestRuleRequired(t *testing.T) {
	rule := GetRule("required")
	require.NotNil(t, rule)

	// string
	assert.Error(t, rule("", "", nil))
	assert.NoError(t, rule("x", "", nil))

	// int
	assert.Error(t, rule(0, "", nil))
	assert.NoError(t, rule(1, "", nil))

	// nil
	assert.Error(t, rule(nil, "", nil))

	// slice
	assert.Error(t, rule([]string{}, "", nil))
	assert.NoError(t, rule([]string{"a"}, "", nil))

	// bool
	assert.Error(t, rule(false, "", nil))
	assert.NoError(t, rule(true, "", nil))

	// pointer
	var p *int
	assert.Error(t, rule(p, "", nil))
	v := 5
	assert.NoError(t, rule(&v, "", nil))

	// map
	assert.Error(t, rule(map[string]int{}, "", nil))
	assert.NoError(t, rule(map[string]int{"a": 1}, "", nil))
}

// ──────────────────────────────────────────────
// min / max / len
// ──────────────────────────────────────────────

func TestRuleMin(t *testing.T) {
	rule := GetRule("min")
	require.NotNil(t, rule)

	// string length
	assert.NoError(t, rule("hello", "3", nil))
	assert.Error(t, rule("hi", "3", nil))

	// int value
	assert.NoError(t, rule(5, "3", nil))
	assert.Error(t, rule(2, "3", nil))

	// uint value
	assert.NoError(t, rule(uint(5), "3", nil))
	assert.Error(t, rule(uint(2), "3", nil))

	// float value
	assert.NoError(t, rule(3.5, "3", nil))
	assert.Error(t, rule(2.5, "3", nil))

	// slice length
	assert.NoError(t, rule([]int{1, 2, 3}, "3", nil))
	assert.Error(t, rule([]int{1, 2}, "3", nil))

	// map length
	assert.NoError(t, rule(map[string]int{"a": 1, "b": 2, "c": 3}, "3", nil))
	assert.Error(t, rule(map[string]int{"a": 1}, "3", nil))

	// invalid param
	assert.Error(t, rule(5, "abc", nil))

	// unsupported type
	assert.Error(t, rule(struct{}{}, "3", nil))
}

func TestRuleMax(t *testing.T) {
	rule := GetRule("max")
	require.NotNil(t, rule)

	// string length
	assert.NoError(t, rule("hi", "3", nil))
	assert.Error(t, rule("hello", "3", nil))

	// int value
	assert.NoError(t, rule(2, "3", nil))
	assert.Error(t, rule(5, "3", nil))

	// uint value
	assert.NoError(t, rule(uint(2), "3", nil))
	assert.Error(t, rule(uint(5), "3", nil))

	// float value
	assert.NoError(t, rule(2.5, "3", nil))
	assert.Error(t, rule(3.5, "3", nil))

	// slice length
	assert.NoError(t, rule([]int{1, 2}, "3", nil))
	assert.NoError(t, rule([]int{1, 2, 3}, "3", nil)) // equal is allowed
	assert.Error(t, rule([]int{1, 2, 3, 4}, "3", nil))

	// map length
	assert.NoError(t, rule(map[string]int{"a": 1}, "3", nil))
	assert.NoError(t, rule(map[string]int{"a": 1, "b": 2, "c": 3}, "3", nil)) // equal is allowed
	assert.Error(t, rule(map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}, "3", nil))

	// invalid param
	assert.Error(t, rule(5, "abc", nil))

	// unsupported type
	assert.Error(t, rule(struct{}{}, "3", nil))
}

func TestRuleLen(t *testing.T) {
	rule := GetRule("len")
	require.NotNil(t, rule)

	// string exact
	assert.NoError(t, rule("abc", "3", nil))
	assert.Error(t, rule("ab", "3", nil))
	assert.Error(t, rule("abcd", "3", nil))

	// slice exact
	assert.NoError(t, rule([]int{1, 2, 3}, "3", nil))
	assert.Error(t, rule([]int{1, 2}, "3", nil))

	// map exact
	assert.NoError(t, rule(map[string]int{"a": 1, "b": 2}, "2", nil))
	assert.Error(t, rule(map[string]int{"a": 1}, "2", nil))

	// int exact
	assert.NoError(t, rule(3, "3", nil))
	assert.Error(t, rule(4, "3", nil))

	// float exact
	assert.NoError(t, rule(3.0, "3", nil))
	assert.Error(t, rule(3.5, "3", nil))

	// invalid param
	assert.Error(t, rule(3, "abc", nil))
}

// ──────────────────────────────────────────────
// eq / ne / gt / gte / lt / lte
// ──────────────────────────────────────────────

func TestRuleEq(t *testing.T) {
	rule := GetRule("eq")
	require.NotNil(t, rule)

	// int equals
	assert.NoError(t, rule(5, "5", nil))
	assert.Error(t, rule(5, "6", nil))

	// string equals (via length)
	assert.NoError(t, rule("abc", "3", nil))
	assert.Error(t, rule("ab", "3", nil))

	// float equals
	assert.NoError(t, rule(3.0, "3", nil))
	assert.Error(t, rule(3.5, "3", nil))
}

func TestRuleNe(t *testing.T) {
	rule := GetRule("ne")
	require.NotNil(t, rule)

	// not equal (valid)
	assert.NoError(t, rule(5, "6", nil))

	// equal (error)
	assert.Error(t, rule(5, "5", nil))
}

func TestRuleGt(t *testing.T) {
	rule := GetRule("gt")
	require.NotNil(t, rule)

	// greater (valid)
	assert.NoError(t, rule(5, "3", nil))

	// not greater (error)
	assert.Error(t, rule(3, "5", nil))
	assert.Error(t, rule(3, "3", nil))

	// invalid param
	assert.Error(t, rule(5, "abc", nil))
}

func TestRuleGte(t *testing.T) {
	rule := GetRule("gte")
	require.NotNil(t, rule)

	// greater (valid)
	assert.NoError(t, rule(5, "3", nil))

	// equal (valid)
	assert.NoError(t, rule(3, "3", nil))

	// less (error)
	assert.Error(t, rule(2, "3", nil))

	// invalid param
	assert.Error(t, rule(5, "abc", nil))
}

func TestRuleLt(t *testing.T) {
	rule := GetRule("lt")
	require.NotNil(t, rule)

	// less (valid)
	assert.NoError(t, rule(3, "5", nil))

	// not less (error)
	assert.Error(t, rule(5, "3", nil))
	assert.Error(t, rule(3, "3", nil))

	// invalid param
	assert.Error(t, rule(5, "abc", nil))
}

func TestRuleLte(t *testing.T) {
	rule := GetRule("lte")
	require.NotNil(t, rule)

	// less (valid)
	assert.NoError(t, rule(2, "3", nil))

	// equal (valid)
	assert.NoError(t, rule(3, "3", nil))

	// greater (error)
	assert.Error(t, rule(5, "3", nil))

	// invalid param
	assert.Error(t, rule(5, "abc", nil))
}

// ──────────────────────────────────────────────
// oneof
// ──────────────────────────────────────────────

func TestRuleOneOf(t *testing.T) {
	rule := GetRule("oneof")
	require.NotNil(t, rule)

	// matching value
	assert.NoError(t, rule("a", "a b c", nil))
	assert.NoError(t, rule("b", "a b c", nil))

	// non-matching value
	assert.Error(t, rule("d", "a b c", nil))

	// multiple options
	assert.NoError(t, rule("c", "a b c d e", nil))
	assert.Error(t, rule("z", "a b c d e", nil))

	// non-string value matching via Sprint
	assert.NoError(t, rule(1, "1 2 3", nil))
	assert.Error(t, rule(9, "1 2 3", nil))
}

// ──────────────────────────────────────────────
// email / url / ip / ipv4 / ipv6
// ──────────────────────────────────────────────

func TestRuleEmail(t *testing.T) {
	rule := GetRule("email")
	require.NotNil(t, rule)

	// valid
	assert.NoError(t, rule("user@example.com", "", nil))
	assert.NoError(t, rule("a.b+c@sub.example.com", "", nil))

	// invalid
	assert.Error(t, rule("not-an-email", "", nil))
	assert.Error(t, rule("user@", "", nil))
	assert.Error(t, rule("@example.com", "", nil))

	// non-string (error)
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

func TestRuleURL(t *testing.T) {
	rule := GetRule("url")
	require.NotNil(t, rule)

	// valid
	assert.NoError(t, rule("http://example.com", "", nil))
	assert.NoError(t, rule("https://example.com/path?q=1", "", nil))

	// invalid
	assert.Error(t, rule("not-a-url", "", nil))

	// missing scheme
	assert.Error(t, rule("example.com", "", nil))
	assert.Error(t, rule("://example.com", "", nil))

	// non-string (error)
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

func TestRuleIP(t *testing.T) {
	rule := GetRule("ip")
	require.NotNil(t, rule)

	// valid IPv4
	assert.NoError(t, rule("192.168.1.1", "", nil))
	// valid IPv6
	assert.NoError(t, rule("::1", "", nil))
	assert.NoError(t, rule("2001:db8::1", "", nil))

	// invalid
	assert.Error(t, rule("not-an-ip", "", nil))
	assert.Error(t, rule("999.999.999.999", "", nil))

	// non-string (error)
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

func TestRuleIPv4(t *testing.T) {
	rule := GetRule("ipv4")
	require.NotNil(t, rule)

	// valid IPv4
	assert.NoError(t, rule("192.168.1.1", "", nil))
	assert.NoError(t, rule("10.0.0.1", "", nil))

	// IPv6 (error)
	assert.Error(t, rule("::1", "", nil))
	assert.Error(t, rule("2001:db8::1", "", nil))

	// invalid
	assert.Error(t, rule("not-an-ip", "", nil))
	assert.Error(t, rule("999.999.999.999", "", nil))

	// non-string (error)
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

func TestRuleIPv6(t *testing.T) {
	rule := GetRule("ipv6")
	require.NotNil(t, rule)

	// valid IPv6
	assert.NoError(t, rule("::1", "", nil))
	assert.NoError(t, rule("2001:db8::1", "", nil))

	// IPv4 (error)
	assert.Error(t, rule("192.168.1.1", "", nil))
	assert.Error(t, rule("10.0.0.1", "", nil))

	// invalid
	assert.Error(t, rule("not-an-ip", "", nil))

	// non-string (error)
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

// ──────────────────────────────────────────────
// alpha / alphanum / numeric
// ──────────────────────────────────────────────

func TestRuleAlpha(t *testing.T) {
	rule := GetRule("alpha")
	require.NotNil(t, rule)

	// valid
	assert.NoError(t, rule("hello", "", nil))
	assert.NoError(t, rule("HelloWorld", "", nil))
	// empty (valid - no chars to fail)
	assert.NoError(t, rule("", "", nil))

	// invalid (contains digits)
	assert.Error(t, rule("hello123", "", nil))
	assert.Error(t, rule("hello world", "", nil))

	// non-string (error)
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

func TestRuleAlphaNum(t *testing.T) {
	rule := GetRule("alphanum")
	require.NotNil(t, rule)

	// valid
	assert.NoError(t, rule("hello123", "", nil))
	assert.NoError(t, rule("Hello123", "", nil))
	assert.NoError(t, rule("", "", nil))

	// invalid (underscore)
	assert.Error(t, rule("hello_123", "", nil))
	assert.Error(t, rule("hello world", "", nil))

	// non-string (error)
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

func TestRuleNumeric(t *testing.T) {
	rule := GetRule("numeric")
	require.NotNil(t, rule)

	// valid
	assert.NoError(t, rule("12345", "", nil))
	assert.NoError(t, rule("0", "", nil))

	// invalid (decimal)
	assert.Error(t, rule("12.34", "", nil))
	// empty (error)
	assert.Error(t, rule("", "", nil))
	// letters
	assert.Error(t, rule("12a34", "", nil))

	// non-string (error)
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

// ──────────────────────────────────────────────
// contains / startswith / endswith / regex
// ──────────────────────────────────────────────

func TestRuleContains(t *testing.T) {
	rule := GetRule("contains")
	require.NotNil(t, rule)

	// contains
	assert.NoError(t, rule("hello world", "world", nil))
	assert.NoError(t, rule("hello world", "hello", nil))

	// doesn't contain
	assert.Error(t, rule("hello world", "xyz", nil))

	// non-string (error)
	assert.Error(t, rule(123, "1", nil))
	assert.ErrorIs(t, rule(123, "1", nil), ErrInvalidType)
}

func TestRuleStartsWith(t *testing.T) {
	rule := GetRule("startswith")
	require.NotNil(t, rule)

	// starts with
	assert.NoError(t, rule("hello", "he", nil))

	// doesn't start with
	assert.Error(t, rule("hello", "xy", nil))

	// non-string (error)
	assert.Error(t, rule(123, "1", nil))
	assert.ErrorIs(t, rule(123, "1", nil), ErrInvalidType)
}

func TestRuleEndsWith(t *testing.T) {
	rule := GetRule("endswith")
	require.NotNil(t, rule)

	// ends with
	assert.NoError(t, rule("hello", "lo", nil))

	// doesn't end with
	assert.Error(t, rule("hello", "xy", nil))

	// non-string (error)
	assert.Error(t, rule(123, "3", nil))
	assert.ErrorIs(t, rule(123, "3", nil), ErrInvalidType)
}

func TestRuleRegex(t *testing.T) {
	rule := GetRule("regex")
	require.NotNil(t, rule)

	// valid match
	assert.NoError(t, rule("hello123", "^[a-z]+[0-9]+$", nil))
	assert.NoError(t, rule("abc", "^a.c$", nil))

	// no match
	assert.Error(t, rule("HELLO", "^[a-z]+$", nil))

	// invalid regex pattern
	assert.Error(t, rule("abc", "([a-z", nil))

	// non-string (error)
	assert.Error(t, rule(123, "^[0-9]+$", nil))
	assert.ErrorIs(t, rule(123, "^[0-9]+$", nil), ErrInvalidType)
}

// ──────────────────────────────────────────────
// Cross-field rules
// ──────────────────────────────────────────────

type crossFieldParent struct {
	Password string
	Confirm  string
	Min      int
	Max      int
}

func TestRuleEqField(t *testing.T) {
	rule := GetRule("eqfield")
	require.NotNil(t, rule)

	// equal fields
	parent := crossFieldParent{Password: "secret", Confirm: "secret"}
	assert.NoError(t, rule("secret", "Password", parent))

	// not equal (error)
	parent2 := crossFieldParent{Password: "secret", Confirm: "different"}
	assert.Error(t, rule("different", "Password", parent2))

	// missing field (nil comparison, error unless value is nil)
	assert.Error(t, rule("x", "NonExistent", parent))
}

func TestRuleNeField(t *testing.T) {
	rule := GetRule("nefield")
	require.NotNil(t, rule)

	// not equal (valid)
	parent := crossFieldParent{Password: "secret", Confirm: "different"}
	assert.NoError(t, rule("different", "Password", parent))

	// equal (error)
	parent2 := crossFieldParent{Password: "secret", Confirm: "secret"}
	assert.Error(t, rule("secret", "Password", parent2))
}

func TestRuleGtField(t *testing.T) {
	rule := GetRule("gtfield")
	require.NotNil(t, rule)

	// greater (valid)
	parent := crossFieldParent{Min: 3, Max: 10}
	assert.NoError(t, rule(10, "Min", parent))

	// not greater (error)
	assert.Error(t, rule(3, "Min", parent))
	assert.Error(t, rule(2, "Min", parent))

	// non-numeric (error)
	parentStr := crossFieldParent{Password: "abc"}
	assert.Error(t, rule("xyz", "Password", parentStr))
}

func TestRuleGteField(t *testing.T) {
	rule := GetRule("gtefield")
	require.NotNil(t, rule)

	// greater (valid)
	parent := crossFieldParent{Min: 3, Max: 10}
	assert.NoError(t, rule(10, "Min", parent))

	// equal (valid)
	assert.NoError(t, rule(3, "Min", parent))

	// less (error)
	assert.Error(t, rule(2, "Min", parent))
}

func TestRuleLteField(t *testing.T) {
	rule := GetRule("ltefield")
	require.NotNil(t, rule)

	// less (valid)
	parent := crossFieldParent{Min: 3, Max: 10}
	assert.NoError(t, rule(2, "Min", parent))

	// equal (valid)
	assert.NoError(t, rule(3, "Min", parent))

	// greater (error)
	assert.Error(t, rule(10, "Min", parent))
}

// ──────────────────────────────────────────────
// unique
// ──────────────────────────────────────────────

func TestRuleUnique(t *testing.T) {
	rule := GetRule("unique")
	require.NotNil(t, rule)

	// unique slice
	assert.NoError(t, rule([]int{1, 2, 3, 4}, "", nil))
	assert.NoError(t, rule([]string{"a", "b", "c"}, "", nil))

	// duplicate (error)
	assert.Error(t, rule([]int{1, 2, 2, 3}, "", nil))
	assert.Error(t, rule([]string{"a", "b", "a"}, "", nil))

	// empty slice (valid)
	assert.NoError(t, rule([]int{}, "", nil))

	// non-slice (error)
	assert.Error(t, rule("abc", "", nil))
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

// ──────────────────────────────────────────────
// Custom rules: AddRule / GetRule
// ──────────────────────────────────────────────

func TestAddRule_AndGetRule(t *testing.T) {
	// Save the original "phone" rule so we can restore it after the test.
	original := GetRule("phone")

	// register a custom rule
	AddRule("phone", func(value any, param string, parent any) error {
		s, ok := value.(string)
		if !ok {
			return ErrInvalidType
		}
		if len(s) != 10 {
			return errCustom("invalid phone number")
		}
		return nil
	})

	fn := GetRule("phone")
	require.NotNil(t, fn)
	assert.NoError(t, fn("1234567890", "", nil))
	assert.Error(t, fn("123", "", nil))

	// use it in validation via ValidateWithTag
	assert.NoError(t, ValidateWithTag("1234567890", "phone"))
	assert.Error(t, ValidateWithTag("123", "phone"))

	// restore the original rule so other tests are not affected
	if original != nil {
		AddRule("phone", original)
	}
}

func TestGetRule_NonExistent(t *testing.T) {
	fn := GetRule("does-not-exist-xyz")
	assert.Nil(t, fn)
}

func TestAddRule_EmptyName(t *testing.T) {
	// empty name is a no-op
	AddRule("", func(value any, param string, parent any) error {
		return nil
	})
	assert.Nil(t, GetRule(""))
}

func TestAddRule_OverwritesBuiltin(t *testing.T) {
	// AddRule overwrites built-in rules. Save the original and restore it
	// afterwards so other tests are not affected.
	original := GetRule("required")
	require.NotNil(t, original)
	AddRule("required", func(value any, param string, parent any) error {
		return errCustom("custom required")
	})
	custom := GetRule("required")
	require.NotNil(t, custom)
	// Funcs cannot be compared with == (except against nil), so verify the
	// behavior changed: the custom rule always errors, while the original
	// "required" treats a non-empty string as valid.
	assert.Error(t, custom("non-empty", "", nil))
	assert.NoError(t, original("non-empty", "", nil))

	// restore original
	AddRule("required", original)
	assert.NoError(t, GetRule("required")("non-empty", "", nil))
}

// errCustom is a tiny helper to create an error.
type customErr string

func (c customErr) Error() string { return string(c) }

func errCustom(s string) error { return customErr(s) }

// ──────────────────────────────────────────────
// phone
// ──────────────────────────────────────────────

func TestRulePhone(t *testing.T) {
	rule := GetRule("phone")
	require.NotNil(t, rule)

	// valid Chinese mobile numbers
	assert.NoError(t, rule("13812345678", "", nil))
	assert.NoError(t, rule("15912345678", "", nil))
	assert.NoError(t, rule("18612345678", "", nil))
	assert.NoError(t, rule("17012345678", "", nil))

	// invalid
	assert.Error(t, rule("12345678901", "", nil))  // starts with 12
	assert.Error(t, rule("10012345678", "", nil))  // starts with 10
	assert.Error(t, rule("1381234567", "", nil))   // too short (10 digits)
	assert.Error(t, rule("138123456789", "", nil)) // too long (12 digits)
	assert.Error(t, rule("23812345678", "", nil))  // doesn't start with 1
	assert.Error(t, rule("", "", nil))

	// non-string
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

// ──────────────────────────────────────────────
// uuid
// ──────────────────────────────────────────────

func TestRuleUUID(t *testing.T) {
	rule := GetRule("uuid")
	require.NotNil(t, rule)

	// valid UUID (any version)
	assert.NoError(t, rule("550e8400-e29b-41d4-a716-446655440000", "", nil))
	assert.NoError(t, rule("550e8400-e29b-31d4-a716-446655440000", "", nil)) // v3
	assert.NoError(t, rule("550e8400-e29b-51d4-a716-446655440000", "", nil)) // v5

	// invalid
	assert.Error(t, rule("not-a-uuid", "", nil))
	assert.Error(t, rule("550e8400-e29b-41d4-a716", "", nil))     // too short
	assert.Error(t, rule("550e8400-e29b-41d4-a716-44665544000g", "", nil)) // invalid char

	// non-string
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

func TestRuleUUID_V4(t *testing.T) {
	rule := GetRule("uuid")
	require.NotNil(t, rule)

	// valid UUID v4
	assert.NoError(t, rule("550e8400-e29b-41d4-a716-446655440000", "v4", nil))
	assert.NoError(t, rule("f47ac10b-58cc-4372-a567-0e02b2c3d479", "v4", nil))

	// invalid UUID v4 (version 3)
	assert.Error(t, rule("550e8400-e29b-31d4-a716-446655440000", "v4", nil))
	// invalid UUID v4 (version 5)
	assert.Error(t, rule("550e8400-e29b-51d4-a716-446655440000", "v4", nil))
	// invalid UUID v4 (wrong variant)
	assert.Error(t, rule("550e8400-e29b-41d4-c716-446655440000", "v4", nil))
}

// ──────────────────────────────────────────────
// slug
// ──────────────────────────────────────────────

func TestRuleSlug(t *testing.T) {
	rule := GetRule("slug")
	require.NotNil(t, rule)

	// valid slugs
	assert.NoError(t, rule("hello", "", nil))
	assert.NoError(t, rule("hello-world", "", nil))
	assert.NoError(t, rule("hello-world-123", "", nil))
	assert.NoError(t, rule("a-b-c", "", nil))
	assert.NoError(t, rule("123", "", nil))
	assert.NoError(t, rule("a1-b2-c3", "", nil))

	// invalid
	assert.Error(t, rule("Hello", "", nil))           // uppercase
	assert.Error(t, rule("hello_world", "", nil))     // underscore
	assert.Error(t, rule("hello--world", "", nil))    // double hyphen
	assert.Error(t, rule("-hello", "", nil))          // leading hyphen
	assert.Error(t, rule("hello-", "", nil))          // trailing hyphen
	assert.Error(t, rule("hello world", "", nil))     // space
	assert.Error(t, rule("", "", nil))

	// non-string
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

// ──────────────────────────────────────────────
// creditcard (Luhn)
// ──────────────────────────────────────────────

func TestRuleCreditCard(t *testing.T) {
	rule := GetRule("creditcard")
	require.NotNil(t, rule)

	// valid credit card numbers (Luhn-valid test numbers)
	assert.NoError(t, rule("4111111111111111", "", nil)) // Visa test
	assert.NoError(t, rule("4242424242424242", "", nil)) // Visa test
	assert.NoError(t, rule("5555555555554444", "", nil)) // Mastercard test
	assert.NoError(t, rule("378282246310005", "", nil))  // Amex test

	// with spaces and hyphens
	assert.NoError(t, rule("4111 1111 1111 1111", "", nil))
	assert.NoError(t, rule("4111-1111-1111-1111", "", nil))

	// invalid (Luhn check fails)
	assert.Error(t, rule("4111111111111112", "", nil))
	assert.Error(t, rule("1234567890123456", "", nil))

	// invalid length
	assert.Error(t, rule("4111111", "", nil))       // too short
	assert.Error(t, rule("41111111111111111111", "", nil)) // too long

	// non-numeric
	assert.Error(t, rule("abcd1234efgh5678", "", nil))

	// non-string
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

func TestLuhnCheck(t *testing.T) {
	assert.True(t, luhnCheck("4111111111111111"))
	assert.True(t, luhnCheck("5555555555554444"))
	assert.True(t, luhnCheck("378282246310005"))
	assert.False(t, luhnCheck("4111111111111112"))
	assert.False(t, luhnCheck("1234567890123456"))
	assert.False(t, luhnCheck("abcdef"))
}

// ──────────────────────────────────────────────
// json
// ──────────────────────────────────────────────

func TestRuleJSON(t *testing.T) {
	rule := GetRule("json")
	require.NotNil(t, rule)

	// valid JSON
	assert.NoError(t, rule(`{"key":"value"}`, "", nil))
	assert.NoError(t, rule(`[1,2,3]`, "", nil))
	assert.NoError(t, rule(`"string"`, "", nil))
	assert.NoError(t, rule(`123`, "", nil))
	assert.NoError(t, rule(`true`, "", nil))
	assert.NoError(t, rule(`null`, "", nil))
	assert.NoError(t, rule(`{"nested":{"a":1},"arr":[1,2]}`, "", nil))

	// invalid
	assert.Error(t, rule(`{key:"value"}`, "", nil))  // unquoted key
	assert.Error(t, rule(`not json`, "", nil))
	assert.Error(t, rule(`{`, "", nil))
	assert.Error(t, rule(`[1,2,`, "", nil))

	// non-string
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

// ──────────────────────────────────────────────
// base64
// ──────────────────────────────────────────────

func TestRuleBase64(t *testing.T) {
	rule := GetRule("base64")
	require.NotNil(t, rule)

	// valid Base64
	assert.NoError(t, rule("aGVsbG8=", "", nil))       // "hello"
	assert.NoError(t, rule("aGVsbG8gd29ybGQ=", "", nil)) // "hello world"
	assert.NoError(t, rule("dGVzdA==", "", nil))       // "test"
	assert.NoError(t, rule("SGVsbG8gV29ybGQh", "", nil))

	// invalid
	assert.Error(t, rule("not base64!", "", nil))
	assert.Error(t, rule("aGVsbG8", "", nil))  // missing padding
	assert.Error(t, rule("", "", nil))
	assert.Error(t, rule("!!!", "", nil))

	// non-string
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}

// ──────────────────────────────────────────────
// hex
// ──────────────────────────────────────────────

func TestRuleHex(t *testing.T) {
	rule := GetRule("hex")
	require.NotNil(t, rule)

	// valid hex
	assert.NoError(t, rule("0123456789abcdef", "", nil))
	assert.NoError(t, rule("ABCDEF", "", nil))
	assert.NoError(t, rule("0x"[:0] + "deadbeef", "", nil))
	assert.NoError(t, rule("123456", "", nil))
	assert.NoError(t, rule("a", "", nil))

	// invalid
	assert.Error(t, rule("xyz", "", nil))
	assert.Error(t, rule("12g4", "", nil))
	assert.Error(t, rule("", "", nil))
	assert.Error(t, rule("0xdeadbeef", "", nil)) // contains 'x'

	// non-string
	assert.Error(t, rule(123, "", nil))
	assert.ErrorIs(t, rule(123, "", nil), ErrInvalidType)
}
