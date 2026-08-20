// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package xinference_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LingByte/ling-base/relay/channel/xinference"
)

func TestXinference_ChannelName(t *testing.T) {
	assert.Equal(t, "xinference", xinference.ChannelName)
}

func TestXinference_ModelList(t *testing.T) {
	models := xinference.ModelList
	assert.NotEmpty(t, models)
}
