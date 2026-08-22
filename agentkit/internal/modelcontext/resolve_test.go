//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package modelcontext

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	compat "github.com/LingByte/ling-base/relay/compat"
)

func TestResolveContextWindowUsesInstanceBeforeRegistry(t *testing.T) {
	const modelName = "resolve-instance-window"
	compat.RegisterModelContextWindow(modelName, 12345)

	window, ok := ResolveContextWindow(contextWindowTestModel{
		name:   modelName,
		window: 54321,
	})
	assert.True(t, ok)
	assert.Equal(t, 54321, window)
}

func TestResolveContextWindowFallsBackToRegistry(t *testing.T) {
	window, ok := ResolveContextWindow(contextWindowTestModel{
		name: "gpt-4o-mini",
	})
	assert.True(t, ok)
	assert.Equal(t, 128000, window)
}

func TestResolveInputTokenBudgetUsesOptionalCapability(t *testing.T) {
	_, ok := ResolveInputTokenBudget(
		context.Background(),
		nil,
		&compat.Request{},
	)
	assert.False(t, ok)

	m := contextWindowTestModel{name: "budget", budget: 1234}
	budget, ok := ResolveInputTokenBudget(
		context.Background(),
		m,
		&compat.Request{},
	)
	assert.True(t, ok)
	assert.Equal(t, 1234, budget)

	_, ok = ResolveInputTokenBudget(
		context.Background(),
		contextWindowOnlyTestModel{},
		&compat.Request{},
	)
	assert.False(t, ok)

	_, ok = ResolveInputTokenBudget(
		context.Background(),
		contextWindowTestModel{name: "zero-budget"},
		&compat.Request{},
	)
	assert.False(t, ok)
}

type contextWindowTestModel struct {
	name   string
	window int
	budget int
}

func (m contextWindowTestModel) GenerateContent(
	context.Context,
	*compat.Request,
) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response)
	close(ch)
	return ch, nil
}

func (m contextWindowTestModel) Info() compat.Info {
	return compat.Info{
		Name:          m.name,
		ContextWindow: m.window,
	}
}

func (m contextWindowTestModel) InputTokenBudget(
	context.Context,
	*compat.Request,
) int {
	return m.budget
}

type contextWindowOnlyTestModel struct{}

func (contextWindowOnlyTestModel) GenerateContent(
	context.Context,
	*compat.Request,
) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response)
	close(ch)
	return ch, nil
}

func (contextWindowOnlyTestModel) Info() compat.Info {
	return compat.Info{Name: "context-only"}
}
