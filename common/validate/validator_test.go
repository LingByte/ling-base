// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package validate

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// parseTag / splitTag
// ──────────────────────────────────────────────

func TestParseTag_Empty(t *testing.T) {
	rules := parseTag("")
	assert.Nil(t, rules)
}

func TestParseTag_SingleRule(t *testing.T) {
	rules := parseTag("required")
	require.Len(t, rules, 1)
	assert.Equal(t, "required", rules[0].name)
	assert.Equal(t, "", rules[0].param)
}

func TestParseTag_MultipleRules(t *testing.T) {
	rules := parseTag("required,min=3,max=50")
	require.Len(t, rules, 3)
	assert.Equal(t, "required", rules[0].name)
	assert.Equal(t, "min", rules[1].name)
	assert.Equal(t, "3", rules[1].param)
	assert.Equal(t, "max", rules[2].name)
	assert.Equal(t, "50", rules[2].param)
}

func TestParseTag_RuleWithParam(t *testing.T) {
	rules := parseTag("oneof=a b c")
	require.Len(t, rules, 1)
	assert.Equal(t, "oneof", rules[0].name)
	assert.Equal(t, "a b c", rules[0].param)
}

func TestParseTag_RuleWithEmptyParam(t *testing.T) {
	rules := parseTag("contains=")
	require.Len(t, rules, 1)
	assert.Equal(t, "contains", rules[0].name)
	assert.Equal(t, "", rules[0].param)
}

func TestParseTag_LowercasesName(t *testing.T) {
	rules := parseTag("REQUIRED")
	require.Len(t, rules, 1)
	assert.Equal(t, "required", rules[0].name)
}

func TestParseTag_TrimsWhitespace(t *testing.T) {
	rules := parseTag("required , min=3")
	require.Len(t, rules, 2)
	assert.Equal(t, "required", rules[0].name)
	assert.Equal(t, "min", rules[1].name)
	assert.Equal(t, "3", rules[1].param)
}

func TestSplitTag_Simple(t *testing.T) {
	parts := splitTag("a,b,c")
	assert.Equal(t, []string{"a", "b", "c"}, parts)
}

func TestSplitTag_QuotedWithCommas(t *testing.T) {
	parts := splitTag(`regex="a,b",required`)
	require.Len(t, parts, 2)
	assert.Equal(t, `regex="a,b"`, parts[0])
	assert.Equal(t, `required`, parts[1])
}

func TestSplitTag_Empty(t *testing.T) {
	parts := splitTag("")
	assert.Nil(t, parts)
}

// ──────────────────────────────────────────────
// ValidateWithTag
// ──────────────────────────────────────────────

func TestValidateWithTag_Valid(t *testing.T) {
	assert.NoError(t, ValidateWithTag("user@example.com", "email"))
	assert.NoError(t, ValidateWithTag("hello", "required,min=3,max=10"))
}

func TestValidateWithTag_Invalid(t *testing.T) {
	err := ValidateWithTag("not-an-email", "email")
	require.Error(t, err)
	fe, ok := err.(FieldError)
	require.True(t, ok)
	assert.Equal(t, "_", fe.Field)
	assert.Equal(t, "email", fe.Rule)
}

func TestValidateWithTag_UnknownRule(t *testing.T) {
	err := ValidateWithTag("x", "nonexistent-rule")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown rule")
}

func TestValidateWithTag_MultipleRules(t *testing.T) {
	// first failing rule returns
	err := ValidateWithTag("", "required,min=3")
	require.Error(t, err)
	fe, ok := err.(FieldError)
	require.True(t, ok)
	assert.Equal(t, "required", fe.Rule)
}

func TestValidateWithTag_EmptyTag(t *testing.T) {
	assert.NoError(t, ValidateWithTag("anything", ""))
}

// ──────────────────────────────────────────────
// Validate
// ──────────────────────────────────────────────

type validStruct struct {
	Name  string `validate:"required,min=3,max=50"`
	Email string `validate:"required,email"`
	Age   int    `validate:"min=18,max=120"`
}

func TestValidate_ValidStruct(t *testing.T) {
	v := validStruct{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
	}
	assert.NoError(t, Validate(v))
}

func TestValidate_InvalidStruct(t *testing.T) {
	v := validStruct{
		Name:  "ab",
		Email: "invalid",
		Age:   5,
	}
	err := Validate(v)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, len(errs) >= 3)
	assert.True(t, errs.Has("Name"))
	assert.True(t, errs.Has("Email"))
	assert.True(t, errs.Has("Age"))
}

func TestValidate_NilInput(t *testing.T) {
	assert.NoError(t, Validate(nil))
}

func TestValidate_NonStructInput(t *testing.T) {
	// non-struct input returns nil (validateStruct returns nil for non-struct)
	assert.NoError(t, Validate("just a string"))
	assert.NoError(t, Validate(123))
}

func TestValidate_PointerToStruct(t *testing.T) {
	v := &validStruct{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
	}
	assert.NoError(t, Validate(v))
}

func TestValidate_NilPointerToStruct(t *testing.T) {
	var v *validStruct
	assert.NoError(t, Validate(v))
}

// ──────────────────────────────────────────────
// Nested struct validation
// ──────────────────────────────────────────────

type addressType struct {
	Street string `validate:"required"`
	City   string `validate:"required"`
}

type personType struct {
	Name    string      `validate:"required"`
	Address addressType `validate:"required"`
}

func TestValidate_NestedStruct(t *testing.T) {
	p := personType{
		Name: "Alice",
		Address: addressType{
			Street: "",
			City:   "",
		},
	}
	err := Validate(p)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, errs.Has("Address.Street"))
	assert.True(t, errs.Has("Address.City"))
}

func TestValidate_NestedStructValid(t *testing.T) {
	p := personType{
		Name: "Alice",
		Address: addressType{
			Street: "123 Main St",
			City:   "Springfield",
		},
	}
	assert.NoError(t, Validate(p))
}

// ──────────────────────────────────────────────
// Slice with dive
// ──────────────────────────────────────────────

type itemStruct struct {
	Name  string `validate:"required"`
	Price int    `validate:"min=1"`
}

type orderStruct struct {
	Items []itemStruct `validate:"required,dive"`
}

func TestValidate_SliceDive_Structs(t *testing.T) {
	o := orderStruct{
		Items: []itemStruct{
			{Name: "A", Price: 10},
			{Name: "", Price: 0}, // invalid
			{Name: "C", Price: 5},
		},
	}
	err := Validate(o)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, errs.Has("Items[1].Name"))
	assert.True(t, errs.Has("Items[1].Price"))
}

func TestValidate_SliceDive_AllValid(t *testing.T) {
	o := orderStruct{
		Items: []itemStruct{
			{Name: "A", Price: 10},
			{Name: "B", Price: 20},
		},
	}
	assert.NoError(t, Validate(o))
}

func TestValidate_SliceDive_Strings(t *testing.T) {
	type tagList struct {
		Tags []string `validate:"dive,min=2"`
	}
	tl := tagList{
		Tags: []string{"abc", "de", "x"}, // "x" fails min=2
	}
	err := Validate(tl)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, errs.Has("Tags[2]"))
}

func TestValidate_SliceDive_Empty(t *testing.T) {
	o := orderStruct{
		Items: []itemStruct{},
	}
	// required fails on empty slice
	err := Validate(o)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, errs.Has("Items"))
}

func TestValidate_SliceDive_Nil(t *testing.T) {
	o := orderStruct{
		Items: nil,
	}
	err := Validate(o)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, errs.Has("Items"))
}

// ──────────────────────────────────────────────
// Map with dive
// ──────────────────────────────────────────────

func TestValidate_MapDive(t *testing.T) {
	type userMap struct {
		Users map[string]itemStruct `validate:"dive"`
	}
	um := userMap{
		Users: map[string]itemStruct{
			"a": {Name: "A", Price: 10},
			"b": {Name: "", Price: 0}, // invalid
		},
	}
	err := Validate(um)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	// map keys are dynamic; check at least one error path contains [b]
	found := false
	for _, fe := range errs {
		if fe.Field == "Users[b].Name" || fe.Field == "Users[b].Price" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestValidate_MapDive_Empty(t *testing.T) {
	type userMap struct {
		Users map[string]itemStruct `validate:"dive"`
	}
	um := userMap{
		Users: map[string]itemStruct{},
	}
	assert.NoError(t, Validate(um))
}

// ──────────────────────────────────────────────
// Cross-field validation
// ──────────────────────────────────────────────

type passwordForm struct {
	Password string `validate:"required,min=8"`
	Confirm  string `validate:"required,eqfield=Password"`
}

func TestValidate_CrossField_PasswordMatch(t *testing.T) {
	form := passwordForm{
		Password: "supersecret",
		Confirm:  "supersecret",
	}
	assert.NoError(t, Validate(form))
}

func TestValidate_CrossField_PasswordMismatch(t *testing.T) {
	form := passwordForm{
		Password: "supersecret",
		Confirm:  "different",
	}
	err := Validate(form)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, errs.Has("Confirm"))
}

// ──────────────────────────────────────────────
// Complex form validation
// ──────────────────────────────────────────────

type registrationForm struct {
	Username string   `validate:"required,min=3,max=20,alphanum"`
	Email    string   `validate:"required,email"`
	Age      int      `validate:"gte=18,lte=120"`
	Gender   string   `validate:"oneof=male female other"`
	Website  string   `validate:"url"`
	Tags     []string `validate:"unique"`
	Password string   `validate:"required,min=8"`
	Confirm  string   `validate:"required,eqfield=Password"`
	AgreeToS bool     `validate:"required"`
}

func TestValidate_ComplexForm_AllValid(t *testing.T) {
	form := registrationForm{
		Username: "alice123",
		Email:    "alice@example.com",
		Age:      30,
		Gender:   "female",
		Website:  "https://alice.example.com",
		Tags:     []string{"a", "b", "c"},
		Password: "supersecret",
		Confirm:  "supersecret",
		AgreeToS: true,
	}
	assert.NoError(t, Validate(form))
}

func TestValidate_ComplexForm_MultipleErrors(t *testing.T) {
	form := registrationForm{
		Username: "ab",               // min=3 fail
		Email:    "not-an-email",     // email fail
		Age:      15,                 // gte=18 fail
		Gender:   "unknown",          // oneof fail
		Website:  "not-a-url",        // url fail
		Tags:     []string{"a", "a"}, // unique fail
		Password: "short",            // min=8 fail
		Confirm:  "different",        // eqfield fail
		AgreeToS: false,              // required fail
	}
	err := Validate(form)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	for _, field := range []string{"Username", "Email", "Age", "Gender", "Website", "Tags", "Password", "Confirm", "AgreeToS"} {
		assert.True(t, errs.Has(field), "expected error for field %s", field)
	}
}

// ──────────────────────────────────────────────
// Unexported fields are skipped
// ──────────────────────────────────────────────

type withUnexported struct {
	Public string `validate:"required"`
	secret string `validate:"required"`
}

func TestValidate_UnexportedFieldsSkipped(t *testing.T) {
	v := withUnexported{
		Public: "ok",
		secret: "",
	}
	// unexported field is skipped, so no error
	assert.NoError(t, Validate(v))
}

func TestValidate_UnexportedFieldsSkipped_PublicInvalid(t *testing.T) {
	v := withUnexported{
		Public: "",
		secret: "",
	}
	err := Validate(v)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, errs.Has("Public"))
	assert.False(t, errs.Has("secret"))
}

// ──────────────────────────────────────────────
// nostructlevel
// ──────────────────────────────────────────────

type nestedNoStructLevel struct {
	Name    string      `validate:"required"`
	Address addressType `validate:"nostructlevel"`
}

func TestValidate_NoStructLevel_SkipsNested(t *testing.T) {
	v := nestedNoStructLevel{
		Name: "Alice",
		Address: addressType{
			Street: "", // would fail if validated
			City:   "",
		},
	}
	// nostructlevel skips nested struct validation → no error
	assert.NoError(t, Validate(v))
}

func TestValidate_NoStructLevel_StillValidatesFieldRules(t *testing.T) {
	type nestedNoStructLevel2 struct {
		Name    string      `validate:"required"`
		Address addressType `validate:"nostructlevel"`
	}
	v := nestedNoStructLevel2{
		Name: "", // fails required
		Address: addressType{
			Street: "",
			City:   "",
		},
	}
	err := Validate(v)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, errs.Has("Name"))
	// Address.Street and Address.City should NOT be present
	assert.False(t, errs.Has("Address.Street"))
	assert.False(t, errs.Has("Address.City"))
}

// ──────────────────────────────────────────────
// Pointer fields
// ──────────────────────────────────────────────

type withPointerField struct {
	Name    string       `validate:"required"`
	Address *addressType `validate:"required"`
}

func TestValidate_PointerField_NilWithRequired(t *testing.T) {
	v := withPointerField{
		Name:    "Alice",
		Address: nil,
	}
	err := Validate(v)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, errs.Has("Address"))
}

func TestValidate_PointerField_NonNilValidStruct(t *testing.T) {
	v := withPointerField{
		Name: "Alice",
		Address: &addressType{
			Street: "123 Main St",
			City:   "Springfield",
		},
	}
	assert.NoError(t, Validate(v))
}

func TestValidate_PointerField_NonNilInvalidStruct(t *testing.T) {
	v := withPointerField{
		Name: "Alice",
		Address: &addressType{
			Street: "", // invalid
			City:   "Springfield",
		},
	}
	err := Validate(v)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, errs.Has("Address.Street"))
}

// ──────────────────────────────────────────────
// Unknown rule in struct tag
// ──────────────────────────────────────────────

func TestValidate_UnknownRuleInTag(t *testing.T) {
	type withUnknown struct {
		Name string `validate:"required,bogusrule"`
	}
	v := withUnknown{Name: "Alice"}
	err := Validate(v)
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, errs.Has("Name"))
	// the error message should mention "unknown rule"
	var found bool
	for _, fe := range errs.ForField("Name") {
		if fe.Rule == "bogusrule" {
			found = true
		}
	}
	assert.True(t, found)
}

// ──────────────────────────────────────────────
// Quoted regex in tag with commas
// ──────────────────────────────────────────────

func TestValidate_RegexWithCommasInTag(t *testing.T) {
	type withRegex struct {
		Code string `validate:"regex=\"a,b\""`
	}
	// The splitTag helper preserves quotes, so the compiled pattern is
	// literally `"a,b"` (including the quote characters). A value that
	// contains those exact characters matches.
	assert.NoError(t, Validate(withRegex{Code: `"a,b"`}))
	// "xyz" does not match the pattern (which expects the literal quotes).
	err := Validate(withRegex{Code: "xyz"})
	require.Error(t, err)
	errs, ok := err.(Errors)
	require.True(t, ok)
	assert.True(t, errs.Has("Code"))
}

func TestParseTag_RegexWithCommasPreserved(t *testing.T) {
	// Verify that a quoted value containing a comma is not split and that
	// the quotes are retained in the param (this is the library's behavior).
	rules := parseTag(`regex="a,b",required`)
	require.Len(t, rules, 2)
	assert.Equal(t, "regex", rules[0].name)
	assert.Equal(t, `"a,b"`, rules[0].param)
	assert.Equal(t, "required", rules[1].name)
}

// ──────────────────────────────────────────────
// All rules pass (fully valid struct)
// ──────────────────────────────────────────────

func TestValidate_AllRulesPass(t *testing.T) {
	type full struct {
		Name     string   `validate:"required,min=2,max=10,len=5"`
		Count    int      `validate:"eq=5,ne=10,gt=4,gte=5,lt=6,lte=5"`
		Choice   string   `validate:"oneof=yes no maybe"`
		Email    string   `validate:"email"`
		URL      string   `validate:"url"`
		IP       string   `validate:"ip"`
		IPv4     string   `validate:"ipv4"`
		IPv6     string   `validate:"ipv6"`
		Alpha    string   `validate:"alpha"`
		AlphaNum string   `validate:"alphanum"`
		Numeric  string   `validate:"numeric"`
		Contains string   `validate:"contains=world"`
		Start    string   `validate:"startswith=he"`
		End      string   `validate:"endswith=lo"`
		Regex    string   `validate:"regex=^h.*o$"`
		Tags     []string `validate:"unique"`
	}
	v := full{
		Name:     "hello",
		Count:    5,
		Choice:   "yes",
		Email:    "a@b.com",
		URL:      "https://example.com",
		IP:       "192.168.1.1",
		IPv4:     "10.0.0.1",
		IPv6:     "::1",
		Alpha:    "hello",
		AlphaNum: "hello123",
		Numeric:  "12345",
		Contains: "hello world",
		Start:    "hello",
		End:      "hello",
		Regex:    "hello",
		Tags:     []string{"a", "b", "c"},
	}
	assert.NoError(t, Validate(v))
}

// ──────────────────────────────────────────────
// hasNoStructLevel helper
// ──────────────────────────────────────────────

func TestHasNoStructLevel_True(t *testing.T) {
	rules := []tagRule{{name: "required"}, {name: "nostructlevel"}}
	assert.True(t, hasNoStructLevel(rules))
}

func TestHasNoStructLevel_False(t *testing.T) {
	rules := []tagRule{{name: "required"}, {name: "min", param: "3"}}
	assert.False(t, hasNoStructLevel(rules))
}

func TestHasNoStructLevel_Empty(t *testing.T) {
	assert.False(t, hasNoStructLevel(nil))
}

// ──────────────────────────────────────────────
// extractField edge cases
// ──────────────────────────────────────────────

func TestExtractField_NilParent(t *testing.T) {
	assert.Nil(t, extractField(nil, "Name"))
}

func TestExtractField_NilPointerParent(t *testing.T) {
	var p *crossFieldParent
	assert.Nil(t, extractField(p, "Name"))
}

func TestExtractField_NonStructParent(t *testing.T) {
	assert.Nil(t, extractField("just a string", "Name"))
}

func TestExtractField_MissingField(t *testing.T) {
	parent := crossFieldParent{Password: "x"}
	assert.Nil(t, extractField(parent, "NonExistent"))
}

func TestExtractField_DottedPath(t *testing.T) {
	type inner struct {
		Value string
	}
	type outer struct {
		Inner inner
	}
	o := outer{Inner: inner{Value: "found"}}
	assert.Equal(t, "found", extractField(o, "Inner.Value"))
}

func TestExtractField_PointerInPath(t *testing.T) {
	type inner struct {
		Value string
	}
	type outer struct {
		Inner *inner
	}
	o := outer{Inner: &inner{Value: "found"}}
	assert.Equal(t, "found", extractField(o, "Inner.Value"))
}

func TestExtractField_NilPointerInPath(t *testing.T) {
	type inner struct {
		Value string
	}
	type outer struct {
		Inner *inner
	}
	o := outer{Inner: nil}
	assert.Nil(t, extractField(o, "Inner.Value"))
}

// ──────────────────────────────────────────────
// toFloat helper
// ──────────────────────────────────────────────

func TestToFloat(t *testing.T) {
	f, err := toFloat(5)
	assert.NoError(t, err)
	assert.Equal(t, 5.0, f)

	f, err = toFloat(uint(7))
	assert.NoError(t, err)
	assert.Equal(t, 7.0, f)

	f, err = toFloat(3.14)
	assert.NoError(t, err)
	assert.InDelta(t, 3.14, f, 0.001)

	_, err = toFloat("not numeric")
	assert.Error(t, err)

	_, err = toFloat(nil)
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// parseNum helper
// ──────────────────────────────────────────────

func TestParseNum(t *testing.T) {
	assert.Equal(t, int64(42), parseNum("42"))
	assert.Equal(t, int64(0), parseNum("not-a-number"))
	assert.Equal(t, int64(-5), parseNum("-5"))
}

// ──────────────────────────────────────────────
// compareValue unsupported type
// ──────────────────────────────────────────────

func TestCompareValue_UnsupportedType(t *testing.T) {
	err := compareValue(struct{}{}, 5, "min",
		func(a, b int64) bool { return a >= b },
		func(a, b float64) bool { return a >= b },
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

// ──────────────────────────────────────────────
// compareFields non-numeric
// ──────────────────────────────────────────────

func TestCompareFields_NonNumeric(t *testing.T) {
	err := compareFields("abc", "def", "gt", func(a, b float64) bool { return a > b })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot compare non-numeric")
}

func TestCompareFields_FailedComparison(t *testing.T) {
	err := compareFields(1, 5, "gt", func(a, b float64) bool { return a > b })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field comparison failed")
}

// ──────────────────────────────────────────────
// validateStruct non-struct / nil pointer
// ──────────────────────────────────────────────

func TestValidateStruct_NonStruct(t *testing.T) {
	errs := validateStruct(reflect.ValueOf("just a string"), "")
	assert.Nil(t, errs)
}

func TestValidateStruct_NilPointer(t *testing.T) {
	var v *validStruct
	errs := validateStruct(reflect.ValueOf(v), "")
	assert.Nil(t, errs)
}

func TestValidateStruct_NonNilPointerDeref(t *testing.T) {
	v := &validStruct{Name: "Alice", Email: "alice@example.com", Age: 30}
	errs := validateStruct(reflect.ValueOf(v), "")
	assert.Nil(t, errs)
}

// ──────────────────────────────────────────────
// validateDive edge cases
// ──────────────────────────────────────────────

func TestValidateDive_NilPointer(t *testing.T) {
	var s *[]itemStruct
	errs := validateDive(reflect.ValueOf(s), "Items", []tagRule{{name: "dive"}})
	assert.Nil(t, errs)
}

func TestValidateDive_NonSliceNonMap(t *testing.T) {
	errs := validateDive(reflect.ValueOf("just a string"), "X", []tagRule{{name: "dive"}})
	assert.Nil(t, errs)
}

func TestValidateDive_ArrayOfStructs(t *testing.T) {
	arr := [2]itemStruct{
		{Name: "A", Price: 10},
		{Name: "", Price: 0},
	}
	errs := validateDive(reflect.ValueOf(arr), "Items", []tagRule{{name: "dive"}})
	require.NotEmpty(t, errs)
	assert.True(t, errs.Has("Items[1].Name"))
	assert.True(t, errs.Has("Items[1].Price"))
}

func TestValidateDive_SliceOfStringsUnknownRule(t *testing.T) {
	// unknown rule in dive is skipped (no fn)
	errs := validateDive(reflect.ValueOf([]string{"a", "b"}), "Tags", []tagRule{{name: "dive"}, {name: "bogusrule"}})
	assert.Nil(t, errs)
}
