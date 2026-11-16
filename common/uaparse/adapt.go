// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package uaparse

import (
	"strings"

	medama "github.com/medama-io/go-useragent"
	"github.com/mssola/user_agent"
	"github.com/ua-parser/uap-go/uaparser"
	"github.com/woothee/woothee-go"
)

func adaptUAP(c *uaparser.Client) signal {
	var s signal
	if c == nil {
		return s
	}
	if c.UserAgent != nil {
		s.browser = normalizeBrowser(c.UserAgent.Family)
		s.browserVer = c.UserAgent.ToVersionString()
	}
	if c.Os != nil {
		s.os = normalizeOS(c.Os.Family)
		s.osVer = c.Os.ToVersionString()
	}
	if c.Device != nil {
		s.family = c.Device.Family
		s.brand = c.Device.Brand
		s.model = c.Device.Model
	}
	s.deviceType = deviceTypeFromUAP(s)
	s.bot = s.deviceType == DeviceBot
	s.mobile = s.deviceType == DeviceMobile
	s.tablet = s.deviceType == DeviceTablet
	s.desktop = s.deviceType == DeviceDesktop
	s.tv = s.deviceType == DeviceTV
	return s
}

func adaptMedama(m medama.UserAgent) signal {
	s := signal{
		browser:    normalizeBrowser(m.GetBrowser()),
		browserVer: m.GetVersion(),
		os:         normalizeOS(m.GetOS()),
		bot:        m.IsBot(),
		mobile:     m.IsMobile(),
		tablet:     m.IsTablet(),
		desktop:    m.IsDesktop(),
		tv:         m.IsTV(),
	}
	switch {
	case m.IsBot():
		s.deviceType = DeviceBot
	case m.IsTV():
		s.deviceType = DeviceTV
	case m.IsTablet():
		s.deviceType = DeviceTablet
	case m.IsMobile():
		s.deviceType = DeviceMobile
	case m.IsDesktop():
		s.deviceType = DeviceDesktop
	default:
		s.deviceType = normalizeMedamaDevice(m.GetDevice())
	}
	return s
}

func adaptMssola(ua string) signal {
	m := user_agent.New(ua)
	browser, bver := m.Browser()
	eng, _ := m.Engine()
	s := signal{
		browser:    normalizeBrowser(browser),
		browserVer: bver,
		os:         normalizeOS(mssolaOSName(m.OS())),
		osVer:      mssolaOSVersion(m.OS()),
		engine:     mapRenderingEngine(eng, browser, ua),
		model:      m.Model(),
		bot:        m.Bot(),
		mobile:     m.Mobile(),
	}
	switch {
	case s.bot:
		s.deviceType = DeviceBot
	case strings.EqualFold(m.Platform(), "iPad"):
		s.deviceType = DeviceTablet
		s.tablet = true
	case s.mobile:
		s.deviceType = DeviceMobile
	default:
		s.deviceType = DeviceDesktop
		s.desktop = true
	}
	return s
}

func adaptWoothee(w *woothee.Result) signal {
	var s signal
	if w == nil {
		return s
	}
	s.browser = normalizeBrowser(w.Name)
	if w.Version != "" && w.Version != woothee.ValueUnknown {
		s.browserVer = w.Version
	}
	s.os = normalizeOS(w.Os)
	if w.OsVersion != "" && w.OsVersion != woothee.ValueUnknown {
		s.osVer = normalizeWootheeOSVersion(w.Os, w.OsVersion)
	}
	s.category = w.Category
	if w.Vendor != "" && w.Vendor != woothee.ValueUnknown {
		s.vendor = w.Vendor
	}
	s.deviceType = deviceTypeFromCategory(w.Category)
	s.bot = w.Category == woothee.CategoryCrawler || w.Type == "crawler"
	s.mobile = s.deviceType == DeviceMobile
	s.desktop = s.deviceType == DeviceDesktop
	return s
}

func deviceTypeFromUAP(s signal) DeviceType {
	if strings.EqualFold(s.family, "Spider") || strings.EqualFold(s.brand, "Spider") {
		return DeviceBot
	}
	fam := strings.ToLower(s.family)
	brand := strings.ToLower(s.brand)
	model := strings.ToLower(s.model)
	switch {
	case isConsoleDevice(s.family, s.brand, s.model):
		return DeviceConsole
	case strings.EqualFold(s.family, "iPad") || strings.Contains(brand, "tablet"):
		return DeviceTablet
	case strings.EqualFold(s.family, "iPhone"),
		strings.Contains(fam, "phone"),
		brand == "generic_android":
		return DeviceMobile
	case s.family == "" || strings.EqualFold(s.family, "Other") || strings.EqualFold(s.family, "Mac"):
		return DeviceDesktop
	default:
		if s.brand != "" || s.model != "" {
			// Named handsets from uap-go (Pixel 7, SM-..., etc.)
			if strings.Contains(model, "pad") || strings.Contains(fam, "pad") {
				return DeviceTablet
			}
			return DeviceMobile
		}
		return DeviceDesktop
	}
}

func deviceTypeFromCategory(cat string) DeviceType {
	switch cat {
	case woothee.CategoryPC:
		return DeviceDesktop
	case woothee.CategorySmartphone, woothee.CategoryMobilephone:
		return DeviceMobile
	case woothee.CategoryCrawler:
		return DeviceBot
	case woothee.CategoryAppliance:
		return DeviceAppliance
	default:
		return DeviceUnknown
	}
}
