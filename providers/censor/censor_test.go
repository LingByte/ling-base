// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package censor

import "testing"

func TestBuildCensorMsg_AllLabels(t *testing.T) {
	cases := []struct {
		label string
		want  string
	}{
		{LabelNormal, "normal content"},
		{LabelSpam, "contains spam"},
		{LabelAd, "advertisement"},
		{LabelPolitics, "political content"},
		{LabelTerrorism, "terrorism content"},
		{LabelAbuse, "abusive content"},
		{LabelPorn, "pornographic content"},
		{LabelFlood, "flooding"},
		{LabelContraband, "contraband"},
		{LabelMeaningless, "meaningless content"},
		{"", "normal content"},
		{"custom_label", "unknown label: custom_label"},
	}
	for _, tc := range cases {
		got := BuildCensorMsg(tc.label)
		if got != tc.want {
			t.Errorf("BuildCensorMsg(%q) = %q, want %q", tc.label, got, tc.want)
		}
	}
}
