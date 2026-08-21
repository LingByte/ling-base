//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package function

import (
	"testing"

	"github.com/LingByte/ling-base/agentkit/tool"
	"github.com/stretchr/testify/require"
)

func TestNormalizeJSONArgs_NilSchemaLeavesEmptyArgs(t *testing.T) {
	require.Nil(t, normalizeJSONArgs(nil, nil))
	require.Empty(t, normalizeJSONArgs([]byte{}, nil))
}

func TestSchemaAcceptsEmptyObject(t *testing.T) {
	require.False(t, schemaAcceptsEmptyObject(nil))
	require.True(t, schemaAcceptsEmptyObject(&tool.Schema{}))
	require.False(t, schemaAcceptsEmptyObject(&tool.Schema{
		Required: []string{"name"},
	}))
}
