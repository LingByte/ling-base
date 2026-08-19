// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package apidocs

import "encoding/json"

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func jsonMarshalString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
