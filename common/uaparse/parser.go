// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package uaparse is a multi-engine User-Agent facade for analytics and
// request classification.
//
// It does not invent matching rules. It wraps mature parsers and merges
// their strengths into one Result:
//
//   - github.com/ua-parser/uap-go       — browser/OS/device identity (ua-parser-js)
//   - github.com/mssola/user_agent      — rendering engine, mobile/bot hints
//   - github.com/woothee/woothee-go    — category / crawler / Asia mobile
//   - github.com/medama-io/go-useragent — fast device class / TV / bot
//
// Default [Parse] uses [ModeStandard]: uap-go as the source of truth for
// identity, with the other engines filling engine, category, and device class.
//
// Use [ModeFast] for high-QPS log pipelines, [ModeUAP] when you must mirror
// frontend ua-parser-js exactly, [ModeLite] for cheap mobile/browser checks.
package uaparse

import (
	"fmt"
	"sync"

	medama "github.com/medama-io/go-useragent"
	"github.com/ua-parser/uap-go/uaparser"
	wootheelib "github.com/woothee/woothee-go"
)

// Mode selects which underlying library (or merge strategy) to run.
type Mode uint8

const (
	// ModeStandard merges all engines: best for product analytics / 埋点.
	// Identity comes from uap-go; engine/category/class are enriched.
	ModeStandard Mode = iota
	// ModeUAP uses only ua-parser/uap-go (align with ua-parser-js).
	ModeUAP
	// ModeFast uses only medama-io/go-useragent (high throughput).
	ModeFast
	// ModeLite uses only mssola/user_agent (cheap mobile/browser/bot).
	ModeLite
	// ModeWoothee uses only woothee-go (stable categories).
	ModeWoothee
)

func (m Mode) String() string {
	switch m {
	case ModeStandard:
		return "standard"
	case ModeUAP:
		return "uap"
	case ModeFast:
		return "fast"
	case ModeLite:
		return "lite"
	case ModeWoothee:
		return "woothee"
	default:
		return fmt.Sprintf("mode(%d)", m)
	}
}

// Option configures a Parser.
type Option func(*Parser)

// WithMode sets the parse strategy. Default is ModeStandard.
func WithMode(m Mode) Option {
	return func(p *Parser) { p.mode = m }
}

// Parser holds initialized engines for one Mode. Safe for concurrent use.
type Parser struct {
	mode Mode

	uap     *uaparser.Parser
	medama  *medama.Parser
	woothee *wootheelib.Parser
}

var (
	registry sync.Map // Mode -> *Parser
)

// New builds a Parser for the given options and initializes required engines.
func New(opts ...Option) (*Parser, error) {
	p := &Parser{mode: ModeStandard}
	for _, o := range opts {
		o(p)
	}
	if err := p.init(); err != nil {
		return nil, err
	}
	return p, nil
}

// MustNew is like New but panics on failure.
func MustNew(opts ...Option) *Parser {
	p, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return p
}

// Default returns a process-wide ModeStandard parser.
func Default() *Parser {
	return For(ModeStandard)
}

// For returns a process-wide parser for mode (created once, shared).
func For(mode Mode) *Parser {
	if v, ok := registry.Load(mode); ok {
		return v.(*Parser)
	}
	p := MustNew(WithMode(mode))
	actual, _ := registry.LoadOrStore(mode, p)
	return actual.(*Parser)
}

// Parse parses ua with the package-default ModeStandard parser.
func Parse(ua string) Result {
	return Default().Parse(ua)
}

// ParseMode parses ua with a shared parser for mode.
func ParseMode(ua string, mode Mode) Result {
	return For(mode).Parse(ua)
}

func (p *Parser) init() error {
	switch p.mode {
	case ModeStandard:
		if err := p.initUAP(); err != nil {
			return err
		}
		p.medama = medama.NewParser()
		p.woothee = wootheelib.NewParser()
	case ModeUAP:
		return p.initUAP()
	case ModeFast:
		p.medama = medama.NewParser()
	case ModeLite:
		// mssola is stateless
	case ModeWoothee:
		p.woothee = wootheelib.NewParser()
	default:
		return fmt.Errorf("uaparse: unknown mode %d", p.mode)
	}
	return nil
}

func (p *Parser) initUAP() error {
	uap, err := uaparser.New()
	if err != nil {
		return fmt.Errorf("uaparse: init uap-go: %w", err)
	}
	p.uap = uap
	return nil
}

// Mode returns the configured strategy.
func (p *Parser) Mode() Mode { return p.mode }

// Parse runs the configured mode against ua.
func (p *Parser) Parse(ua string) Result {
	if ua == "" {
		return Result{Mode: p.mode}
	}
	switch p.mode {
	case ModeUAP:
		return resultFromSignal(ua, p.mode, adaptUAP(p.uap.Parse(ua)))
	case ModeFast:
		return resultFromSignal(ua, p.mode, adaptMedama(p.medama.Parse(ua)))
	case ModeLite:
		return resultFromSignal(ua, p.mode, adaptMssola(ua))
	case ModeWoothee:
		w, _ := p.woothee.Parse(ua)
		return resultFromSignal(ua, p.mode, adaptWoothee(w))
	default:
		return mergeStandard(ua, p)
	}
}
