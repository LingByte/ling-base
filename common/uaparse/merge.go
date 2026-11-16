// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package uaparse

import "strings"

// mergeStandard uses uap-go as identity source of truth, then enriches:
//   - mssola  → rendering engine + mobile/bot hints
//   - woothee → category / vendor / crawler
//   - medama  → device class / TV / bot confirmation
func mergeStandard(ua string, p *Parser) Result {
	uap := adaptUAP(p.uap.Parse(ua))
	ms := adaptMssola(ua)
	wt, _ := p.woothee.Parse(ua)
	wo := adaptWoothee(wt)
	md := adaptMedama(p.medama.Parse(ua))

	s := signal{
		browser:    pick(uap.browser, md.browser, ms.browser, wo.browser),
		browserVer: pick(uap.browserVer, md.browserVer, ms.browserVer, wo.browserVer),
		os:         pick(uap.os, md.os, wo.os, ms.os),
		osVer:      pick(uap.osVer, wo.osVer, ms.osVer),
		engine:     pick(ms.engine, inferEngine(uap.browser, ua)),
		family:     uap.family,
		brand:      uap.brand,
		model:      pick(uap.model, ms.model),
		category:   wo.category,
		vendor:     pick(wo.vendor, uap.brand),
		bot:        uap.bot || ms.bot || wo.bot || md.bot,
		mobile:     uap.mobile || ms.mobile || md.mobile || wo.mobile,
		tablet:     uap.tablet || ms.tablet || md.tablet,
		desktop:    uap.desktop || md.desktop || wo.desktop,
		tv:         md.tv,
	}
	s.deviceType = mergeDeviceType(ua, uap, ms, wo, md, s.bot)
	return resultFromSignal(ua, ModeStandard, s)
}

func mergeDeviceType(ua string, uap, ms, wo, md signal, bot bool) DeviceType {
	if bot {
		return DeviceBot
	}
	if isConsoleUA(ua) || isConsoleDevice(uap.family, uap.brand, uap.model) {
		return DeviceConsole
	}
	if uap.deviceType == DeviceTablet || ms.deviceType == DeviceTablet || md.tablet ||
		strings.Contains(strings.ToLower(uap.brand), "tablet") {
		return DeviceTablet
	}
	if wo.deviceType == DeviceAppliance {
		return DeviceAppliance
	}
	if md.deviceType == DeviceTV || md.tv {
		return DeviceTV
	}
	if uap.deviceType == DeviceMobile || ms.mobile || md.mobile ||
		wo.deviceType == DeviceMobile {
		return DeviceMobile
	}
	if uap.deviceType == DeviceDesktop || md.desktop || wo.deviceType == DeviceDesktop {
		return DeviceDesktop
	}
	return firstDeviceType(md.deviceType, wo.deviceType, ms.deviceType, uap.deviceType)
}

func pick(vals ...string) string {
	for _, v := range vals {
		if v != "" && !strings.EqualFold(v, "unknown") {
			return v
		}
	}
	return ""
}

func firstDeviceType(vals ...DeviceType) DeviceType {
	for _, v := range vals {
		if v != "" && v != DeviceUnknown {
			return v
		}
	}
	return DeviceUnknown
}
