// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package pinyin

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPinyin_SingleChar(t *testing.T) {
	assert.Equal(t, "yi", PinyinFirst('一'))
	assert.Equal(t, "hao", PinyinFirst('好'))
	assert.Equal(t, "shi", PinyinFirst('世'))
	assert.Equal(t, "jie", PinyinFirst('界'))
}

func TestPinyin_NonCJK(t *testing.T) {
	assert.Equal(t, "", PinyinFirst('a'))
	assert.Equal(t, "", PinyinFirst('1'))
	assert.Equal(t, "", PinyinFirst(' '))
}

func TestPinyin_AllReadings(t *testing.T) {
	// 乐 is a multi-tone character: le luo yao yue
	all := PinyinAll('乐')
	assert.Contains(t, all, "le")
	assert.Contains(t, all, "yue")
	assert.Greater(t, len(all), 1)
}

func TestPinyin_FullString(t *testing.T) {
	assert.Equal(t, "yi", Pinyin('一'))
	assert.Equal(t, "le luo yao yue", Pinyin('乐'))
}

func TestPinyinAll_NonCJK(t *testing.T) {
	assert.Nil(t, PinyinAll('a'))
	assert.Nil(t, PinyinAll('1'))
}

func TestIsCJK(t *testing.T) {
	assert.True(t, IsCJK('一'))
	assert.True(t, IsCJK('龥')) // U+9FA5, last in basic block
	assert.False(t, IsCJK('a'))
	assert.False(t, IsCJK(' '))
	assert.False(t, IsCJK(rune(0x3400))) // Extension A, not covered
	assert.False(t, IsCJK(rune(0x4DFF))) // just outside basic block
}

func TestHasPinyin(t *testing.T) {
	assert.True(t, HasPinyin('中'))
	assert.True(t, HasPinyin('国'))
	assert.False(t, HasPinyin('a'))
	assert.False(t, HasPinyin('1'))
}

func TestConvert_Basic(t *testing.T) {
	result := Convert("你好世界")
	assert.Equal(t, "ni hao shi jie", result)
}

func TestConvert_EmptyString(t *testing.T) {
	assert.Equal(t, "", Convert(""))
}

func TestConvert_MixedString(t *testing.T) {
	// By default, non-CJK is dropped
	result := Convert("Hello世界")
	assert.Equal(t, "shi jie", result)
}

func TestConvert_KeepNonCJK(t *testing.T) {
	result := Convert("Hello世界", WithKeepNonCJK(true))
	assert.Equal(t, "Hello shi jie", result)
}

func TestConvert_CustomSeparator(t *testing.T) {
	result := Convert("你好世界", WithSeparator("_"))
	assert.Equal(t, "ni_hao_shi_jie", result)
}

func TestConvert_NoSeparator(t *testing.T) {
	result := Convert("你好世界", WithNoSeparator())
	assert.Equal(t, "nihaoshijie", result)
}

func TestConvert_UnderscoreSeparator(t *testing.T) {
	result := Convert("你好", WithUnderscoreSeparator())
	assert.Equal(t, "ni_hao", result)
}

func TestConvert_HyphenSeparator(t *testing.T) {
	result := Convert("你好", WithHyphenSeparator())
	assert.Equal(t, "ni-hao", result)
}

func TestConvert_AllTones(t *testing.T) {
	// 乐 has multiple readings; with AllTones they should all appear
	result := Convert("乐", WithAllTones(true))
	assert.Equal(t, "le luo yao yue", result)
}

func TestConvert_FirstToneOnly(t *testing.T) {
	result := Convert("乐")
	assert.Equal(t, "le", result)
}

func TestConvert_Uppercase(t *testing.T) {
	result := Convert("你好", WithUppercase(true))
	assert.Equal(t, "NI HAO", result)
}

func TestConvert_TitleCase(t *testing.T) {
	result := Convert("你好", WithTitleCase(true))
	assert.Equal(t, "Ni Hao", result)
}

func TestConvert_Punctuation(t *testing.T) {
	// Punctuation is non-CJK, dropped by default
	result := Convert("你好，世界！")
	assert.Equal(t, "ni hao shi jie", result)

	// With keepNonCJK, punctuation is kept
	result = Convert("你好，世界！", WithKeepNonCJK(true))
	assert.Equal(t, "ni hao ， shi jie ！", result)
}

func TestConvert_Numbers(t *testing.T) {
	result := Convert("第3条", WithKeepNonCJK(true))
	assert.Equal(t, "di 3 tiao", result)
}

func TestConvertAll_Basic(t *testing.T) {
	result := ConvertAll("你好")
	assert.Len(t, result, 2)
	assert.Equal(t, []string{"ni"}, result[0])
	assert.Equal(t, []string{"hao"}, result[1])
}

func TestConvertAll_MultiTone(t *testing.T) {
	result := ConvertAll("重庆")
	assert.Len(t, result, 2)
	// 重 has multiple readings
	assert.Greater(t, len(result[0]), 0)
	assert.Contains(t, result[0], "chong")
	assert.Contains(t, result[1], "qing")
}

func TestConvertAll_KeepNonCJK(t *testing.T) {
	result := ConvertAll("a中b", WithKeepNonCJK(true))
	assert.Len(t, result, 3)
	assert.Equal(t, []string{"a"}, result[0])
	assert.Equal(t, []string{"zhong"}, result[1])
	assert.Equal(t, []string{"b"}, result[2])
}

func TestConvertAll_SkipNonCJK(t *testing.T) {
	result := ConvertAll("a中b")
	assert.Len(t, result, 1)
	assert.Equal(t, []string{"zhong"}, result[0])
}

func TestConvertAll_Empty(t *testing.T) {
	result := ConvertAll("")
	assert.Nil(t, result)
}

func TestConvertAll_Uppercase(t *testing.T) {
	result := ConvertAll("你好", WithUppercase(true))
	assert.Equal(t, []string{"NI"}, result[0])
	assert.Equal(t, []string{"HAO"}, result[1])
}

func TestConvertSlice_Basic(t *testing.T) {
	result := ConvertSlice("你好世界")
	assert.Equal(t, []string{"ni", "hao", "shi", "jie"}, result)
}

func TestConvertSlice_Empty(t *testing.T) {
	result := ConvertSlice("")
	assert.Nil(t, result)
}

func TestConvertSlice_KeepNonCJK(t *testing.T) {
	result := ConvertSlice("你好123", WithKeepNonCJK(true))
	// Consecutive non-CJK chars are grouped into a single element.
	assert.Equal(t, []string{"ni", "hao", "123"}, result)
}

func TestConvertSlice_Uppercase(t *testing.T) {
	result := ConvertSlice("你好", WithUppercase(true))
	assert.Equal(t, []string{"NI", "HAO"}, result)
}

func TestCount(t *testing.T) {
	count := Count()
	assert.Greater(t, count, 20000) // should have ~20812 entries
	t.Logf("Total characters: %d", count)
}

func TestConvert_LongString(t *testing.T) {
	input := "中华人民共和国万岁"
	result := Convert(input)
	parts := strings.Fields(result)
	assert.Len(t, parts, 9) // 9 characters
	assert.Equal(t, "zhong", parts[0])
	assert.Equal(t, "hua", parts[1])
}

func TestConvert_AllBasicBlock(t *testing.T) {
	// Test first and last characters of the basic block
	assert.Equal(t, "yi", PinyinFirst('一'))  // U+4E00
	assert.NotEqual(t, "", PinyinFirst('龥')) // U+9FA5
}

func TestConvert_SpecialChars(t *testing.T) {
	// Newlines and tabs are non-CJK
	result := Convert("你好\n世界\t", WithKeepNonCJK(true))
	assert.Contains(t, result, "ni")
	assert.Contains(t, result, "hao")
	assert.Contains(t, result, "shi")
	assert.Contains(t, result, "jie")
}

func TestPinyinFirst_MultiTone(t *testing.T) {
	// 行: hang xing
	assert.Equal(t, "hang", PinyinFirst('行'))
}

func TestPinyinFirst_SingleTone(t *testing.T) {
	assert.Equal(t, "zhong", PinyinFirst('中'))
	assert.Equal(t, "guo", PinyinFirst('国'))
}

func TestToTitleCase(t *testing.T) {
	assert.Equal(t, "Ni", toTitleCase("ni"))
	assert.Equal(t, "Hao", toTitleCase("hao"))
	assert.Equal(t, "", toTitleCase(""))
	assert.Equal(t, "A", toTitleCase("a"))
	assert.Equal(t, "Ab", toTitleCase("ab"))
}

func TestConvert_CombinedOptions(t *testing.T) {
	result := Convert("你好世界",
		WithSeparator("-"),
		WithUppercase(true),
		WithKeepNonCJK(true),
	)
	assert.Equal(t, "NI-HAO-SHI-JIE", result)
}

func TestConvert_OnlyNonCJK(t *testing.T) {
	// String with no Chinese characters
	result := Convert("Hello World 123")
	assert.Equal(t, "", result)

	result = Convert("Hello World 123", WithKeepNonCJK(true))
	assert.Equal(t, "Hello World 123", result)
}
