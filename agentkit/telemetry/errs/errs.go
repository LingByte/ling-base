// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.

// Package errs provides helpers to convert between the agent ResponseError
package errs

import compat "github.com/LingByte/ling-base/relay/compat"

// ToResponseError converts an error to a ResponseError.
var ToResponseError = func(err error) *compat.ResponseError {
	respErr := compat.ResponseErrorFromError(err, "")
	if respErr == nil {
		return nil
	}
	respErr.Message = err.Error()
	return respErr
}
