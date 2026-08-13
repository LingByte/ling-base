// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package i18n

import (
	"context"
	"testing"
)

func TestWithLocale(t *testing.T) {
	ctx := WithLocale(context.Background(), LocaleZhCN)
	if loc := GetLocaleFromContext(ctx); loc != LocaleZhCN {
		t.Errorf("got %s, want zh-CN", loc)
	}
}

func TestGetLocaleFromContext_Default(t *testing.T) {
	if loc := GetLocaleFromContext(context.Background()); loc != DefaultLocale {
		t.Errorf("got %s, want %s", loc, DefaultLocale)
	}
}

func TestSetLocale(t *testing.T) {
	ctx := SetLocale(context.Background(), LocaleEn)
	if loc := GetLocaleFromContext(ctx); loc != LocaleEn {
		t.Errorf("got %s, want en", loc)
	}
}
