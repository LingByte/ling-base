// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package uaparse

import "strings"

func normalizeOS(name string) string {
	n := strings.TrimSpace(name)
	if n == "" || strings.EqualFold(n, "unknown") || strings.EqualFold(n, "other") {
		return ""
	}
	lower := strings.ToLower(n)
	switch {
	case lower == "mac os x" || lower == "mac osx" || lower == "macos" || lower == "mac os":
		return "macOS"
	case strings.HasPrefix(lower, "mac os"):
		return "macOS"
	case lower == "chrome os" || lower == "chromeos" || lower == "cros":
		return "ChromeOS"
	case strings.HasPrefix(lower, "windows"):
		return "Windows"
	case strings.HasPrefix(lower, "android"):
		return "Android"
	case lower == "ios" || lower == "iphone" || lower == "ipad":
		return "iOS"
	case strings.HasPrefix(lower, "linux") || lower == "ubuntu" || lower == "fedora" || lower == "debian":
		return "Linux"
	default:
		return n
	}
}

func normalizeBrowser(name string) string {
	n := strings.TrimSpace(name)
	if n == "" || strings.EqualFold(n, "unknown") {
		return ""
	}
	lower := strings.ToLower(n)
	switch {
	case lower == "mobile safari" || lower == "safari":
		return "Safari"
	case lower == "chrome mobile" || lower == "chrome mobile ios" || lower == "chrome":
		return "Chrome"
	case lower == "firefox ios" || lower == "firefox mobile" || lower == "firefox":
		return "Firefox"
	case lower == "edge" || lower == "edge mobile":
		return "Edge"
	case lower == "opera" || lower == "opera mobile" || lower == "opera mini":
		return "Opera"
	case lower == "ie" || lower == "internet explorer":
		return "IE"
	case lower == "samsung internet" || lower == "samsung browser":
		return "Samsung Browser"
	case lower == "googlebot":
		return "Googlebot"
	default:
		return n
	}
}

func normalizeMedamaDevice(d string) DeviceType {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case "desktop":
		return DeviceDesktop
	case "mobile":
		return DeviceMobile
	case "tablet":
		return DeviceTablet
	case "bot":
		return DeviceBot
	case "tv":
		return DeviceTV
	default:
		return DeviceUnknown
	}
}

func mapRenderingEngine(engine, browser, ua string) string {
	e := strings.ToLower(strings.TrimSpace(engine))
	b := strings.ToLower(strings.TrimSpace(browser))
	ual := strings.ToLower(ua)
	switch {
	case e == "trident" || b == "internet explorer" || b == "ie":
		return "Trident"
	case e == "gecko" || b == "firefox":
		return "Gecko"
	case e == "applewebkit":
		if strings.Contains(ual, "chrome") || strings.Contains(ual, "crios") ||
			b == "chrome" || b == "edge" || b == "opera" || strings.Contains(b, "samsung") {
			return "Blink"
		}
		return "WebKit"
	case e == "blink":
		return "Blink"
	default:
		return inferEngine(browser, ua)
	}
}

func inferEngine(browser, ua string) string {
	b := strings.ToLower(browser)
	ual := strings.ToLower(ua)
	switch {
	case b == "ie" || b == "internet explorer" || strings.Contains(ual, "trident/"):
		return "Trident"
	case b == "firefox" || (strings.Contains(ual, "gecko/") && !strings.Contains(ual, "chrome")):
		return "Gecko"
	case b == "chrome" || b == "edge" || b == "opera" || strings.Contains(b, "samsung"):
		return "Blink"
	case b == "safari" || strings.Contains(ual, "applewebkit/"):
		return "WebKit"
	default:
		return ""
	}
}

func mssolaOSName(os string) string {
	os = strings.TrimSpace(os)
	if os == "" {
		return ""
	}
	lower := strings.ToLower(os)
	switch {
	case strings.Contains(lower, "windows"):
		return "Windows"
	case strings.Contains(lower, "android"):
		return "Android"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad") || strings.Contains(lower, "cpu os"):
		return "iOS"
	case strings.Contains(lower, "mac os"):
		return "macOS"
	case strings.Contains(lower, "cros"):
		return "ChromeOS"
	case strings.Contains(lower, "linux"):
		return "Linux"
	default:
		return os
	}
}

func mssolaOSVersion(os string) string {
	os = strings.TrimSpace(os)
	lower := strings.ToLower(os)
	switch {
	case strings.HasPrefix(lower, "windows "):
		return strings.TrimSpace(os[len("Windows "):])
	case strings.HasPrefix(lower, "android "):
		return strings.TrimSpace(os[len("Android "):])
	case strings.Contains(lower, "mac os x "):
		idx := strings.Index(lower, "mac os x ")
		return strings.ReplaceAll(extractVersion(os[idx+9:]), "_", ".")
	case strings.Contains(lower, "iphone os "):
		idx := strings.Index(lower, "iphone os ")
		return strings.ReplaceAll(extractVersion(os[idx+10:]), "_", ".")
	case strings.Contains(lower, "cpu os "):
		idx := strings.Index(lower, "cpu os ")
		return strings.ReplaceAll(extractVersion(os[idx+7:]), "_", ".")
	default:
		return ""
	}
}

func normalizeWootheeOSVersion(os, ver string) string {
	ver = strings.TrimSpace(ver)
	if strings.HasPrefix(strings.ToLower(ver), "nt ") {
		return strings.TrimSpace(ver[3:])
	}
	if strings.HasPrefix(strings.ToLower(os), "windows") {
		return ""
	}
	return ver
}

func isConsoleUA(ua string) bool {
	lower := strings.ToLower(ua)
	return strings.Contains(lower, "playstation") ||
		strings.Contains(lower, "xbox") ||
		strings.Contains(lower, "nintendo")
}

func isConsoleDevice(family, brand, model string) bool {
	blob := strings.ToLower(family + " " + brand + " " + model)
	return strings.Contains(blob, "playstation") ||
		strings.Contains(blob, "xbox") ||
		strings.Contains(blob, "nintendo")
}

func extractVersion(s string) string {
	var b strings.Builder
	dots := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			b.WriteByte(c)
			continue
		}
		if c == '.' || c == '_' {
			if i+1 >= len(s) || s[i+1] < '0' || s[i+1] > '9' {
				break
			}
			b.WriteByte('.')
			dots++
			if dots >= 4 {
				break
			}
			continue
		}
		break
	}
	return strings.TrimSuffix(b.String(), ".")
}
