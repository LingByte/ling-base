// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qrcode

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinTemplates_NonEmpty(t *testing.T) {
	all := BuiltinTemplates()
	require.NotEmpty(t, all)

	seen := map[string]bool{}
	for _, tmpl := range all {
		assert.NotEmpty(t, tmpl.ID)
		assert.NotEmpty(t, tmpl.Name)
		assert.NotEqual(t, CategoryCustom, tmpl.Category)
		assert.False(t, seen[tmpl.ID], "duplicate id %s", tmpl.ID)
		seen[tmpl.ID] = true
	}
}

func TestListTemplates_ByCategory(t *testing.T) {
	simple := ListTemplates(CategorySimple)
	classic := ListTemplates(CategoryClassic)
	creative := ListTemplates(CategoryCreative)
	assert.NotEmpty(t, simple)
	assert.NotEmpty(t, classic)
	assert.NotEmpty(t, creative)

	for _, tmpl := range simple {
		assert.Equal(t, CategorySimple, tmpl.Category)
	}
}

func TestGenerateFromTemplate(t *testing.T) {
	data, err := GenerateFromTemplate("https://ling-base.dev", "simple-dots")
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	_, err = png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
}

func TestGenerateFromTemplate_AllBuiltins(t *testing.T) {
	for _, tmpl := range BuiltinTemplates() {
		data, err := GenerateFromTemplate("template:"+tmpl.ID, tmpl.ID)
		require.NoError(t, err, tmpl.ID)
		assert.NotEmpty(t, data, tmpl.ID)
	}
}

func TestGenerateFromTemplate_Unknown(t *testing.T) {
	_, err := GenerateFromTemplate("x", "no-such-template")
	require.Error(t, err)
}

func TestRegisterTemplate(t *testing.T) {
	t.Cleanup(ClearCustomTemplates)

	err := RegisterTemplate(Template{
		ID:   "my-brand",
		Name: "品牌蓝",
		Options: FancyOptions{
			Module:      ShapeCircle,
			Finder:      FinderRounded,
			FgColor:     color.RGBA{R: 30, G: 64, B: 175, A: 255},
			BgColor:     color.White,
			ModuleWidth: 21,
			BorderWidth: 20,
		},
	})
	require.NoError(t, err)

	tmpl, ok := GetTemplate("my-brand")
	require.True(t, ok)
	assert.Equal(t, CategoryCustom, tmpl.Category)

	custom := ListTemplates(CategoryCustom)
	require.Len(t, custom, 1)

	data, err := GenerateFromTemplate("custom", "my-brand")
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Cannot overwrite builtin.
	err = RegisterTemplate(Template{ID: "simple-dots", Name: "x"})
	require.Error(t, err)

	UnregisterTemplate("my-brand")
	_, ok = GetTemplate("my-brand")
	assert.False(t, ok)
}

func TestGenerateFromTemplate_OverrideColors(t *testing.T) {
	data, err := GenerateFromTemplate("override", "simple-square", TemplateOverride{
		FgColor: color.RGBA{R: 200, G: 0, B: 0, A: 255},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}
