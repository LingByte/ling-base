//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package codeact_test

import "github.com/LingByte/ling-base/agentkit/codeexecutor/codeact"

// Keep the original positional literal source-compatible.
var _ codeact.Runtime = codeact.LocalRunner{"python3"}
