// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package uaparse_test

import (
	"testing"

	"github.com/LingByte/ling-base/common/uaparse"
)

func TestParse_Chrome_Windows(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	info := uaparse.Parse(ua)
	if info.OS != "Windows" {
		t.Errorf("OS = %q, want Windows", info.OS)
	}
	if info.OSVersion != "10" {
		t.Errorf("OSVersion = %q, want 10", info.OSVersion)
	}
	if info.Browser != "Chrome" {
		t.Errorf("Browser = %q, want Chrome", info.Browser)
	}
	if info.BrowserVersion != "120.0.0.0" {
		t.Errorf("BrowserVersion = %q, want 120.0.0.0", info.BrowserVersion)
	}
	if info.Device != "Desktop" {
		t.Errorf("Device = %q, want Desktop", info.Device)
	}
	if info.Engine != "Blink" {
		t.Errorf("Engine = %q, want Blink", info.Engine)
	}
	if info.IsBot {
		t.Error("IsBot = true, want false")
	}
}

func TestParse_Firefox_macOS(t *testing.T) {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0"
	info := uaparse.Parse(ua)
	if info.OS != "macOS" {
		t.Errorf("OS = %q, want macOS", info.OS)
	}
	if info.OSVersion != "10.15" {
		t.Errorf("OSVersion = %q, want 10.15", info.OSVersion)
	}
	if info.Browser != "Firefox" {
		t.Errorf("Browser = %q, want Firefox", info.Browser)
	}
	if info.BrowserVersion != "121.0" {
		t.Errorf("BrowserVersion = %q, want 121.0", info.BrowserVersion)
	}
	if info.Device != "Desktop" {
		t.Errorf("Device = %q, want Desktop", info.Device)
	}
	if info.Engine != "Gecko" {
		t.Errorf("Engine = %q, want Gecko", info.Engine)
	}
}

func TestParse_Safari_iPhone(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1"
	info := uaparse.Parse(ua)
	if info.OS != "iOS" {
		t.Errorf("OS = %q, want iOS", info.OS)
	}
	if info.Browser != "Safari" {
		t.Errorf("Browser = %q, want Safari", info.Browser)
	}
	if info.BrowserVersion != "17.2" {
		t.Errorf("BrowserVersion = %q, want 17.2", info.BrowserVersion)
	}
	if info.Device != "Mobile" {
		t.Errorf("Device = %q, want Mobile", info.Device)
	}
	if info.Engine != "WebKit" {
		t.Errorf("Engine = %q, want WebKit", info.Engine)
	}
}

func TestParse_Edge_Windows(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0"
	info := uaparse.Parse(ua)
	if info.Browser != "Edge" {
		t.Errorf("Browser = %q, want Edge", info.Browser)
	}
	if info.BrowserVersion != "120.0.0.0" {
		t.Errorf("BrowserVersion = %q, want 120.0.0.0", info.BrowserVersion)
	}
	if info.Engine != "Blink" {
		t.Errorf("Engine = %q, want Blink", info.Engine)
	}
}

func TestParse_Opera(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 OPR/106.0.0.0"
	info := uaparse.Parse(ua)
	if info.Browser != "Opera" {
		t.Errorf("Browser = %q, want Opera", info.Browser)
	}
	if info.BrowserVersion != "106.0.0.0" {
		t.Errorf("BrowserVersion = %q, want 106.0.0.0", info.BrowserVersion)
	}
}

func TestParse_Android_Chrome_Mobile(t *testing.T) {
	ua := "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"
	info := uaparse.Parse(ua)
	if info.OS != "Android" {
		t.Errorf("OS = %q, want Android", info.OS)
	}
	if info.Device != "Mobile" {
		t.Errorf("Device = %q, want Mobile", info.Device)
	}
	if info.Browser != "Chrome" {
		t.Errorf("Browser = %q, want Chrome", info.Browser)
	}
}

func TestParse_Android_Tablet(t *testing.T) {
	ua := "Mozilla/5.0 (Linux; Android 12; SM-X906) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	info := uaparse.Parse(ua)
	if info.OS != "Android" {
		t.Errorf("OS = %q, want Android", info.OS)
	}
	if info.Device != "Tablet" {
		t.Errorf("Device = %q, want Tablet", info.Device)
	}
}

func TestParse_iPad(t *testing.T) {
	ua := "Mozilla/5.0 (iPad; CPU OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1"
	info := uaparse.Parse(ua)
	if info.OS != "iOS" {
		t.Errorf("OS = %q, want iOS", info.OS)
	}
	if info.Device != "Tablet" {
		t.Errorf("Device = %q, want Tablet", info.Device)
	}
}

func TestParse_IE11(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; rv:11.0) like Gecko"
	info := uaparse.Parse(ua)
	if info.Browser != "IE" {
		t.Errorf("Browser = %q, want IE", info.Browser)
	}
	if info.OS != "Windows" {
		t.Errorf("OS = %q, want Windows", info.OS)
	}
	if info.OSVersion != "7" {
		t.Errorf("OSVersion = %q, want 7", info.OSVersion)
	}
	if info.Engine != "Trident" {
		t.Errorf("Engine = %q, want Trident", info.Engine)
	}
}

func TestParse_GoogleBot(t *testing.T) {
	ua := "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
	info := uaparse.Parse(ua)
	if !info.IsBot {
		t.Error("IsBot = false, want true")
	}
	if info.Device != "Bot" {
		t.Errorf("Device = %q, want Bot", info.Device)
	}
}

func TestParse_BingBot(t *testing.T) {
	ua := "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)"
	info := uaparse.Parse(ua)
	if !info.IsBot {
		t.Error("IsBot = false, want true")
	}
}

func TestParse_Empty(t *testing.T) {
	info := uaparse.Parse("")
	if info.OS != "" || info.Browser != "" || info.Device != "" {
		t.Errorf("Parse(\"\") = %+v, want zero value", info)
	}
}

func TestParse_Linux_Desktop(t *testing.T) {
	ua := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	info := uaparse.Parse(ua)
	if info.OS != "Linux" {
		t.Errorf("OS = %q, want Linux", info.OS)
	}
	if info.Device != "Desktop" {
		t.Errorf("Device = %q, want Desktop", info.Device)
	}
}

func TestParse_ChromeOS(t *testing.T) {
	ua := "Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	info := uaparse.Parse(ua)
	if info.OS != "ChromeOS" {
		t.Errorf("OS = %q, want ChromeOS", info.OS)
	}
}

func TestParse_SamsungBrowser(t *testing.T) {
	ua := "Mozilla/5.0 (Linux; Android 13; SAMSUNG SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/22.0 Chrome/108.0.5359.128 Mobile Safari/537.36"
	info := uaparse.Parse(ua)
	if info.Browser != "Samsung Browser" {
		t.Errorf("Browser = %q, want Samsung Browser", info.Browser)
	}
	if info.OS != "Android" {
		t.Errorf("OS = %q, want Android", info.OS)
	}
}

func TestParse_iOSChrome(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0.6099.119 Mobile/15E148 Safari/604.1"
	info := uaparse.Parse(ua)
	if info.Browser != "Chrome" {
		t.Errorf("Browser = %q, want Chrome", info.Browser)
	}
	if info.OS != "iOS" {
		t.Errorf("OS = %q, want iOS", info.OS)
	}
}

func TestParse_iOSFirefox(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/121.0 Mobile/15E148 Safari/605.1.15"
	info := uaparse.Parse(ua)
	if info.Browser != "Firefox" {
		t.Errorf("Browser = %q, want Firefox", info.Browser)
	}
}

func TestParse_PlayStation(t *testing.T) {
	ua := "Mozilla/5.0 (PlayStation; PlayStation 5/5.0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/5.0 Safari/605.1.15"
	info := uaparse.Parse(ua)
	if info.Device != "Console" {
		t.Errorf("Device = %q, want Console", info.Device)
	}
}

func TestParse_MacOS_UnderscoreVersion(t *testing.T) {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	info := uaparse.Parse(ua)
	if info.OS != "macOS" {
		t.Errorf("OS = %q, want macOS", info.OS)
	}
	if info.OSVersion != "10.15.7" {
		t.Errorf("OSVersion = %q, want 10.15.7", info.OSVersion)
	}
}
