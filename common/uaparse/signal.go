// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package uaparse

// signal is a raw, engine-agnostic parse contribution used for merging.
// It is intentionally unexported — callers only see Result.
type signal struct {
	browser, browserVer string
	os, osVer           string
	engine              string

	deviceType           DeviceType
	family, brand, model string

	category, vendor string

	bot, mobile, tablet, desktop, tv bool
}

func resultFromSignal(ua string, mode Mode, s signal) Result {
	finalizeDevice(&s, ua)
	r := Result{
		UA:       ua,
		Mode:     mode,
		Browser:  Browser{Name: s.browser, Version: s.browserVer},
		OS:       OS{Name: s.os, Version: s.osVer},
		Engine:   s.engine,
		Category: s.category,
		Vendor:   s.vendor,
		Device: Device{
			Type:   s.deviceType,
			Family: s.family,
			Brand:  s.brand,
			Model:  s.model,
		},
	}
	if r.Engine == "" {
		r.Engine = inferEngine(r.Browser.Name, ua)
	}
	return r
}

func finalizeDevice(s *signal, ua string) {
	if s.bot || s.deviceType == DeviceBot {
		s.deviceType = DeviceBot
		return
	}
	if s.deviceType != "" && s.deviceType != DeviceUnknown {
		return
	}
	switch {
	case isConsoleUA(ua) || isConsoleDevice(s.family, s.brand, s.model):
		s.deviceType = DeviceConsole
	case s.tv:
		s.deviceType = DeviceTV
	case s.tablet:
		s.deviceType = DeviceTablet
	case s.mobile:
		s.deviceType = DeviceMobile
	case s.desktop:
		s.deviceType = DeviceDesktop
	case s.category != "":
		s.deviceType = deviceTypeFromCategory(s.category)
	default:
		s.deviceType = DeviceUnknown
	}
}
