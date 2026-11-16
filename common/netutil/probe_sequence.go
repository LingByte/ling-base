// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package netutil

import (
	"context"
	"fmt"
	"net/http/cookiejar"
	"time"
)

// ProbeSequenceStep is a single step in a multi-step probe sequence.
type ProbeSequenceStep struct {
	Name    string
	Request ProbeRequest
	// ExtractVars maps variable names → JSON paths (e.g. "token" → "data.access_token").
	ExtractVars map[string]string
}

// ProbeSequenceResult holds the outcome of a sequence execution.
type ProbeSequenceResult struct {
	Success   bool
	Steps     []ProbeStepResult
	Duration  time.Duration
	Error     string
	Variables map[string]string
	Timestamp time.Time
}

// ProbeStepResult holds the result of a single step within a sequence.
type ProbeStepResult struct {
	Name    string
	Result  ProbeResult
	Skipped bool
}

// ProbeSequence runs HTTP probes with variable pass-through and a shared cookie jar.
//
// If any step fails, subsequent steps are skipped (recorded as Skipped).
type ProbeSequence struct {
	prober *Prober
	steps  []ProbeSequenceStep
	vars   map[string]string
}

// NewProbeSequence creates a sequence runner. If prober is nil, a default Prober
// with an isolated cookie jar is created so cookies flow across steps.
func NewProbeSequence(prober *Prober, steps []ProbeSequenceStep) *ProbeSequence {
	jar, _ := cookiejar.New(nil)
	if prober == nil {
		prober = NewProber()
	}
	// Attach jar to the shared client (and insecure clone path) without mutating
	// an externally supplied client's jar if one was already set.
	if prober.client != nil && prober.client.Jar == nil {
		prober.client.Jar = jar
	}
	return &ProbeSequence{
		prober: prober,
		steps:  steps,
		vars:   make(map[string]string),
	}
}

// WithVariables seeds the sequence variable map before Execute.
func (s *ProbeSequence) WithVariables(vars map[string]string) *ProbeSequence {
	for k, v := range vars {
		s.vars[k] = v
	}
	return s
}

// Execute runs all steps in order.
func (s *ProbeSequence) Execute(ctx context.Context) ProbeSequenceResult {
	result := ProbeSequenceResult{
		Timestamp: time.Now(),
		Variables: make(map[string]string),
		Steps:     make([]ProbeStepResult, 0, len(s.steps)),
	}
	start := time.Now()
	defer func() { result.Duration = time.Since(start) }()

	failed := false
	for _, step := range s.steps {
		if failed {
			result.Steps = append(result.Steps, ProbeStepResult{Name: step.Name, Skipped: true})
			continue
		}

		req := step.Request.Clone()
		if req.Variables == nil {
			req.Variables = make(map[string]string)
		}
		for k, v := range s.vars {
			if _, exists := req.Variables[k]; !exists {
				req.Variables[k] = v
			}
		}

		stepResult := s.prober.Execute(ctx, req)
		result.Steps = append(result.Steps, ProbeStepResult{Name: step.Name, Result: stepResult})

		if !stepResult.Success {
			failed = true
			result.Error = fmt.Sprintf("step %q: %s", step.Name, stepResult.Error)
			continue
		}

		if len(step.ExtractVars) > 0 && stepResult.Body != "" {
			for varName, jsonPath := range step.ExtractVars {
				val, err := extractProbeJSONPath([]byte(stepResult.Body), jsonPath)
				if err != nil {
					continue
				}
				s.vars[varName] = val
				result.Variables[varName] = val
			}
		}
		for k, v := range stepResult.Extracted {
			s.vars[k] = v
			result.Variables[k] = v
		}
	}

	result.Success = !failed
	return result
}

// Variables returns a copy of the current variable map.
func (s *ProbeSequence) Variables() map[string]string {
	out := make(map[string]string, len(s.vars))
	for k, v := range s.vars {
		out[k] = v
	}
	return out
}
