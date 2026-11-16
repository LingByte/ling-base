// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package uaparse_test

import (
	"strings"
	"testing"

	"github.com/LingByte/ling-base/common/uaparse"
)

func TestParse_Chrome_Windows(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	r := uaparse.Parse(ua)
	if r.OS.Name != "Windows" {
		t.Errorf("OS = %q, want Windows", r.OS.Name)
	}
	if r.OS.Version != "10" {
		t.Errorf("OS.Version = %q, want 10", r.OS.Version)
	}
	if r.Browser.Name != "Chrome" {
		t.Errorf("Browser = %q, want Chrome", r.Browser.Name)
	}
	if !strings.HasPrefix(r.Browser.Version, "120") {
		t.Errorf("Browser.Version = %q, want 120.x", r.Browser.Version)
	}
	if r.Device.Type != uaparse.DeviceDesktop {
		t.Errorf("Device.Type = %q, want desktop", r.Device.Type)
	}
	if r.Engine != "Blink" {
		t.Errorf("Engine = %q, want Blink", r.Engine)
	}
	if r.Bot() || !r.Desktop() {
		t.Errorf("flags: bot=%v desktop=%v", r.Bot(), r.Desktop())
	}
}

func TestParse_Firefox_macOS(t *testing.T) {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0"
	r := uaparse.Parse(ua)
	if r.OS.Name != "macOS" {
		t.Errorf("OS = %q, want macOS", r.OS.Name)
	}
	if r.OS.Version != "10.15" {
		t.Errorf("OS.Version = %q, want 10.15", r.OS.Version)
	}
	if r.Browser.Name != "Firefox" {
		t.Errorf("Browser = %q, want Firefox", r.Browser.Name)
	}
	if r.Browser.Version != "121.0" {
		t.Errorf("Browser.Version = %q, want 121.0", r.Browser.Version)
	}
	if r.Engine != "Gecko" {
		t.Errorf("Engine = %q, want Gecko", r.Engine)
	}
}

func TestParse_Safari_iPhone(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1"
	r := uaparse.Parse(ua)
	if r.OS.Name != "iOS" {
		t.Errorf("OS = %q, want iOS", r.OS.Name)
	}
	if r.Browser.Name != "Safari" {
		t.Errorf("Browser = %q, want Safari", r.Browser.Name)
	}
	if r.Browser.Version != "17.2" {
		t.Errorf("Browser.Version = %q, want 17.2", r.Browser.Version)
	}
	if r.Device.Type != uaparse.DeviceMobile {
		t.Errorf("Device.Type = %q, want mobile", r.Device.Type)
	}
	if r.Engine != "WebKit" {
		t.Errorf("Engine = %q, want WebKit", r.Engine)
	}
	if r.Device.Brand != "Apple" {
		t.Errorf("Brand = %q, want Apple", r.Device.Brand)
	}
}

func TestParse_Edge_Windows(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0"
	r := uaparse.Parse(ua)
	if r.Browser.Name != "Edge" {
		t.Errorf("Browser = %q, want Edge", r.Browser.Name)
	}
	if r.Engine != "Blink" {
		t.Errorf("Engine = %q, want Blink", r.Engine)
	}
}

func TestParse_Opera(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 OPR/106.0.0.0"
	r := uaparse.Parse(ua)
	if r.Browser.Name != "Opera" {
		t.Errorf("Browser = %q, want Opera", r.Browser.Name)
	}
}

func TestParse_Android_Chrome_Mobile(t *testing.T) {
	ua := "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"
	r := uaparse.Parse(ua)
	if r.OS.Name != "Android" {
		t.Errorf("OS = %q, want Android", r.OS.Name)
	}
	if r.Device.Type != uaparse.DeviceMobile {
		t.Errorf("Device.Type = %q, want mobile", r.Device.Type)
	}
	if r.Browser.Name != "Chrome" {
		t.Errorf("Browser = %q, want Chrome", r.Browser.Name)
	}
	if r.Device.Model != "Pixel 7" {
		t.Errorf("Model = %q, want Pixel 7", r.Device.Model)
	}
}

func TestParse_Android_Tablet(t *testing.T) {
	ua := "Mozilla/5.0 (Linux; Android 12; SM-X906) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	r := uaparse.Parse(ua)
	if r.Device.Type != uaparse.DeviceTablet {
		t.Errorf("Device.Type = %q, want tablet", r.Device.Type)
	}
}

func TestParse_iPad(t *testing.T) {
	ua := "Mozilla/5.0 (iPad; CPU OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1"
	r := uaparse.Parse(ua)
	if r.OS.Name != "iOS" {
		t.Errorf("OS = %q, want iOS", r.OS.Name)
	}
	if r.Device.Type != uaparse.DeviceTablet {
		t.Errorf("Device.Type = %q, want tablet", r.Device.Type)
	}
}

func TestParse_IE11(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; rv:11.0) like Gecko"
	r := uaparse.Parse(ua)
	if r.Browser.Name != "IE" {
		t.Errorf("Browser = %q, want IE", r.Browser.Name)
	}
	if r.OS.Name != "Windows" || r.OS.Version != "7" {
		t.Errorf("OS = %s/%s, want Windows/7", r.OS.Name, r.OS.Version)
	}
	if r.Engine != "Trident" {
		t.Errorf("Engine = %q, want Trident", r.Engine)
	}
}

func TestParse_GoogleBot(t *testing.T) {
	ua := "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
	r := uaparse.Parse(ua)
	if !r.Bot() {
		t.Error("Bot() = false, want true")
	}
	if r.Device.Type != uaparse.DeviceBot {
		t.Errorf("Device.Type = %q, want bot", r.Device.Type)
	}
}

func TestParse_BingBot(t *testing.T) {
	ua := "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)"
	r := uaparse.Parse(ua)
	if !r.Bot() {
		t.Error("Bot() = false, want true")
	}
}

func TestParse_Empty(t *testing.T) {
	r := uaparse.Parse("")
	if !r.Empty() {
		t.Errorf("Parse(\"\") = %+v, want empty", r)
	}
}

func TestParse_Linux_Desktop(t *testing.T) {
	ua := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	r := uaparse.Parse(ua)
	if r.OS.Name != "Linux" {
		t.Errorf("OS = %q, want Linux", r.OS.Name)
	}
	if r.Device.Type != uaparse.DeviceDesktop {
		t.Errorf("Device.Type = %q, want desktop", r.Device.Type)
	}
}

func TestParse_ChromeOS(t *testing.T) {
	ua := "Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	r := uaparse.Parse(ua)
	if r.OS.Name != "ChromeOS" {
		t.Errorf("OS = %q, want ChromeOS", r.OS.Name)
	}
}

func TestParse_SamsungBrowser(t *testing.T) {
	ua := "Mozilla/5.0 (Linux; Android 13; SAMSUNG SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/22.0 Chrome/108.0.5359.128 Mobile Safari/537.36"
	r := uaparse.Parse(ua)
	if r.Browser.Name != "Samsung Browser" {
		t.Errorf("Browser = %q, want Samsung Browser", r.Browser.Name)
	}
	if r.Device.Brand != "Samsung" {
		t.Errorf("Brand = %q, want Samsung", r.Device.Brand)
	}
}

func TestParse_iOSChrome(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0.6099.119 Mobile/15E148 Safari/604.1"
	r := uaparse.Parse(ua)
	if r.Browser.Name != "Chrome" {
		t.Errorf("Browser = %q, want Chrome", r.Browser.Name)
	}
	if r.OS.Name != "iOS" {
		t.Errorf("OS = %q, want iOS", r.OS.Name)
	}
}

func TestParse_iOSFirefox(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/121.0 Mobile/15E148 Safari/605.1.15"
	r := uaparse.Parse(ua)
	if r.Browser.Name != "Firefox" {
		t.Errorf("Browser = %q, want Firefox", r.Browser.Name)
	}
}

func TestParse_PlayStation(t *testing.T) {
	ua := "Mozilla/5.0 (PlayStation; PlayStation 5/5.0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/5.0 Safari/605.1.15"
	r := uaparse.Parse(ua)
	if r.Device.Type != uaparse.DeviceConsole {
		t.Errorf("Device.Type = %q, want console", r.Device.Type)
	}
}

func TestParse_MacOS_UnderscoreVersion(t *testing.T) {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	r := uaparse.Parse(ua)
	if r.OS.Name != "macOS" {
		t.Errorf("OS = %q, want macOS", r.OS.Name)
	}
	if r.OS.Version != "10.15.7" {
		t.Errorf("OS.Version = %q, want 10.15.7", r.OS.Version)
	}
}

func TestModes_Smoke(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	for _, m := range []uaparse.Mode{
		uaparse.ModeStandard,
		uaparse.ModeUAP,
		uaparse.ModeFast,
		uaparse.ModeLite,
		uaparse.ModeWoothee,
	} {
		r := uaparse.ParseMode(ua, m)
		if r.Browser.Name != "Chrome" {
			t.Errorf("%s: Browser = %q, want Chrome", m, r.Browser.Name)
		}
		if r.Mode != m {
			t.Errorf("%s: Result.Mode = %v", m, r.Mode)
		}
	}
}

func TestFor_Shared(t *testing.T) {
	a := uaparse.For(uaparse.ModeFast)
	b := uaparse.For(uaparse.ModeFast)
	if a != b {
		t.Fatal("For(ModeFast) should return the same instance")
	}
}

func TestResult_String(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	s := uaparse.Parse(ua).String()
	if !strings.Contains(s, "Chrome") || !strings.Contains(s, "Windows") {
		t.Fatalf("String() = %q", s)
	}
}
