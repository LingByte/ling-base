// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package uaparse

import "strings"

// DeviceType is the coarse device class used for analytics.
type DeviceType string

const (
	DeviceDesktop   DeviceType = "desktop"
	DeviceMobile    DeviceType = "mobile"
	DeviceTablet    DeviceType = "tablet"
	DeviceBot       DeviceType = "bot"
	DeviceConsole   DeviceType = "console"
	DeviceTV        DeviceType = "tv"
	DeviceAppliance DeviceType = "appliance"
	DeviceUnknown   DeviceType = "unknown"
)

// Browser is the client application identity.
type Browser struct {
	Name    string `json:"name,omitempty"`    // Chrome, Safari, Firefox, Edge, ...
	Version string `json:"version,omitempty"` // e.g. "120.0.0"
}

// OS is the operating system identity.
type OS struct {
	Name    string `json:"name,omitempty"`    // Windows, macOS, iOS, Android, ...
	Version string `json:"version,omitempty"` // e.g. "10", "17.2"
}

// Device is the hardware / form-factor identity.
type Device struct {
	Type   DeviceType `json:"type,omitempty"`
	Family string     `json:"family,omitempty"` // iPhone, Pixel 7, Spider, ...
	Brand  string     `json:"brand,omitempty"`  // Apple, Samsung, Google, ...
	Model  string     `json:"model,omitempty"`
}

// Result is the unified User-Agent parse output.
type Result struct {
	UA string `json:"ua,omitempty"`

	Browser Browser `json:"browser"`
	OS      OS      `json:"os"`
	Device  Device  `json:"device"`

	// Engine is the rendering engine: Blink, Gecko, WebKit, Trident.
	Engine string `json:"engine,omitempty"`

	// Category is a woothee-style bucket: pc, smartphone, crawler, appliance, ...
	Category string `json:"category,omitempty"`
	// Vendor is a software/hardware vendor hint when known.
	Vendor string `json:"vendor,omitempty"`

	// Mode is the strategy that produced this result.
	Mode Mode `json:"mode"`
}

// Bot reports whether the UA is a crawler / bot.
func (r Result) Bot() bool { return r.Device.Type == DeviceBot }

// Mobile reports whether the device class is a phone.
func (r Result) Mobile() bool { return r.Device.Type == DeviceMobile }

// Tablet reports whether the device class is a tablet.
func (r Result) Tablet() bool { return r.Device.Type == DeviceTablet }

// Desktop reports whether the device class is a desktop.
func (r Result) Desktop() bool { return r.Device.Type == DeviceDesktop }

// Empty reports whether nothing useful was parsed.
func (r Result) Empty() bool {
	return r.Browser.Name == "" && r.OS.Name == "" && r.Device.Type == "" && r.Device.Family == ""
}

// String returns a short human-readable summary.
func (r Result) String() string {
	parts := make([]string, 0, 4)
	if r.Browser.Name != "" {
		if r.Browser.Version != "" {
			parts = append(parts, r.Browser.Name+"/"+r.Browser.Version)
		} else {
			parts = append(parts, r.Browser.Name)
		}
	}
	if r.OS.Name != "" {
		if r.OS.Version != "" {
			parts = append(parts, r.OS.Name+"/"+r.OS.Version)
		} else {
			parts = append(parts, r.OS.Name)
		}
	}
	if r.Device.Type != "" {
		parts = append(parts, string(r.Device.Type))
	}
	if r.Engine != "" {
		parts = append(parts, r.Engine)
	}
	return strings.Join(parts, " ")
}
