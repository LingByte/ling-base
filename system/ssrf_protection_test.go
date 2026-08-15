package system

import (
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	private := []string{"10.0.0.1", "172.16.0.1", "192.168.1.1", "127.0.0.1", "169.254.1.1"}
	for _, s := range private {
		if !isPrivateIP(net.ParseIP(s)) {
			t.Fatalf("expected %s to be private", s)
		}
	}
	public := []string{"8.8.8.8", "1.1.1.1", "114.114.114.114"}
	for _, s := range public {
		if isPrivateIP(net.ParseIP(s)) {
			t.Fatalf("expected %s to be public", s)
		}
	}
	// IPv6 private
	ipv6Private := []string{"::1", "fe80::1", "fc00::1", "fd00::1"}
	for _, s := range ipv6Private {
		if !isPrivateIP(net.ParseIP(s)) {
			t.Fatalf("expected IPv6 %s to be private", s)
		}
	}
	// IPv6 public
	if isPrivateIP(net.ParseIP("2606:4700:4700::1111")) {
		t.Fatal("expected IPv6 2606:4700:4700::1111 to be public")
	}
}

func TestIsIPInCIDRList(t *testing.T) {
	list := []string{"10.0.0.0/8", "192.168.1.1", "172.16.0.0/12"}
	cases := []struct {
		ip   string
		want bool
	}{
		{"10.1.2.3", true},
		{"10.255.255.255", true},
		{"192.168.1.1", true},
		{"172.16.5.5", true},
		{"8.8.8.8", false},
		{"11.0.0.1", false},
	}
	for _, c := range cases {
		got := isIPInCIDRList(net.ParseIP(c.ip), list)
		if got != c.want {
			t.Fatalf("isIPInCIDRList(%s)=%v want %v", c.ip, got, c.want)
		}
	}

	// invalid CIDR entries should be skipped
	weirdList := []string{"not-a-cidr", "10.0.0.0/8", "also-invalid"}
	if !isIPInCIDRList(net.ParseIP("10.1.1.1"), weirdList) {
		t.Fatal("should match despite invalid entries")
	}
	if isIPInCIDRList(net.ParseIP("8.8.8.8"), weirdList) {
		t.Fatal("should not match")
	}

	// empty list
	if isIPInCIDRList(net.ParseIP("10.0.0.1"), nil) {
		t.Fatal("empty list should not match")
	}
}

func TestParsePortRanges(t *testing.T) {
	ports, err := parsePortRanges([]string{"80", "443", "8000-8002"})
	if err != nil {
		t.Fatal(err)
	}
	want := []int{80, 443, 8000, 8001, 8002}
	if len(ports) != len(want) {
		t.Fatalf("got %v want %v", ports, want)
	}
	for i := range want {
		if ports[i] != want[i] {
			t.Fatalf("ports[%d]=%d want %d", i, ports[i], want[i])
		}
	}

	// empty entries skipped
	ports, err = parsePortRanges([]string{"", "  ", "80"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ports) != 1 || ports[0] != 80 {
		t.Fatalf("got %v want [80]", ports)
	}

	// empty input
	ports, err = parsePortRanges(nil)
	if err != nil || len(ports) != 0 {
		t.Fatalf("nil input: ports=%v err=%v", ports, err)
	}

	// out-of-range port
	if _, err := parsePortRanges([]string{"99999"}); err == nil {
		t.Fatal("expected error for out-of-range port")
	}
	if _, err := parsePortRanges([]string{"0"}); err == nil {
		t.Fatal("expected error for port 0")
	}

	// reversed range
	if _, err := parsePortRanges([]string{"8000-7000"}); err == nil {
		t.Fatal("expected error for reversed range")
	}

	// invalid range format (too many parts)
	if _, err := parsePortRanges([]string{"8000-9000-10000"}); err == nil {
		t.Fatal("expected error for invalid range format")
	}

	// invalid port in range
	if _, err := parsePortRanges([]string{"abc-9000"}); err == nil {
		t.Fatal("expected error for non-numeric start port")
	}
	if _, err := parsePortRanges([]string{"8000-abc"}); err == nil {
		t.Fatal("expected error for non-numeric end port")
	}
	if _, err := parsePortRanges([]string{"8000-99999"}); err == nil {
		t.Fatal("expected error for out-of-range end port")
	}

	// non-numeric single port
	if _, err := parsePortRanges([]string{"abc"}); err == nil {
		t.Fatal("expected error for non-numeric port")
	}
}

func TestIsDomainListed(t *testing.T) {
	list := []string{"example.com", "*.test.com"}
	if !isDomainListed("example.com", list) {
		t.Fatal("exact match failed")
	}
	if !isDomainListed("sub.test.com", list) {
		t.Fatal("wildcard match failed")
	}
	if !isDomainListed("test.com", list) {
		t.Fatal("wildcard bare domain match failed")
	}
	if isDomainListed("evil.com", list) {
		t.Fatal("non-listed domain should not match")
	}

	// empty list
	if isDomainListed("example.com", nil) {
		t.Fatal("empty list should not match")
	}

	// list with empty entries
	if !isDomainListed("example.com", []string{"", "example.com"}) {
		t.Fatal("should skip empty entries and match")
	}

	// case insensitive
	if !isDomainListed("EXAMPLE.COM", list) {
		t.Fatal("case insensitive match failed")
	}
	if !isDomainListed("SUB.TEST.COM", list) {
		t.Fatal("case insensitive wildcard match failed")
	}
}

func TestSSRFProtection_ValidateURL(t *testing.T) {
	p := &SSRFProtection{
		AllowPrivateIp: false,
	}
	// private IP should be rejected
	if err := p.ValidateURL("http://10.0.0.1/x"); err == nil {
		t.Fatal("expected private IP rejection")
	}
	// public IP should pass
	if err := p.ValidateURL("http://8.8.8.8/x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// non-http protocol should be rejected
	if err := p.ValidateURL("ftp://8.8.8.8/x"); err == nil {
		t.Fatal("expected protocol rejection")
	}
	// invalid URL
	if err := p.ValidateURL("://invalid"); err == nil {
		t.Fatal("expected error for invalid URL")
	}
	// port restriction
	p.AllowedPorts = []int{80, 443}
	if err := p.ValidateURL("http://8.8.8.8:8080/x"); err == nil {
		t.Fatal("expected port rejection")
	}
	if err := p.ValidateURL("http://8.8.8.8:80/x"); err != nil {
		t.Fatalf("unexpected error for allowed port: %v", err)
	}
	// HTTPS default port
	if err := p.ValidateURL("https://8.8.8.8/x"); err != nil {
		t.Fatalf("unexpected error for HTTPS default port: %v", err)
	}
	// invalid port in URL
	if err := p.ValidateURL("http://8.8.8.8:abc/x"); err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestSSRFProtection_DomainWhitelist(t *testing.T) {
	p := &SSRFProtection{
		DomainFilterMode: true,
		DomainList:       []string{"safe.com"},
	}
	if err := p.ValidateURL("http://evil.com/x"); err == nil {
		t.Fatal("expected domain whitelist rejection")
	}
	if !p.isDomainAllowed("safe.com") {
		t.Fatal("safe.com should be allowed")
	}
	if p.isDomainAllowed("evil.com") {
		t.Fatal("evil.com should not be allowed")
	}
}

func TestSSRFProtection_DomainBlacklist(t *testing.T) {
	p := &SSRFProtection{
		DomainFilterMode: false, // blacklist mode
		DomainList:       []string{"evil.com"},
	}
	if !p.isDomainAllowed("good.com") {
		t.Fatal("good.com should be allowed in blacklist mode")
	}
	if p.isDomainAllowed("evil.com") {
		t.Fatal("evil.com should be blocked in blacklist mode")
	}
}

func TestSSRFProtection_IPWhitelist(t *testing.T) {
	p := &SSRFProtection{
		IpFilterMode: true,
		IpList:       []string{"8.8.8.8"},
	}
	if !p.IsIPAccessAllowed(net.ParseIP("8.8.8.8")) {
		t.Fatal("8.8.8.8 should be allowed in whitelist")
	}
	if p.IsIPAccessAllowed(net.ParseIP("1.1.1.1")) {
		t.Fatal("1.1.1.1 should not be allowed in whitelist")
	}
}

func TestSSRFProtection_IPBlacklist(t *testing.T) {
	p := &SSRFProtection{
		IpFilterMode: false,
		IpList:       []string{"8.8.8.8"},
	}
	if p.IsIPAccessAllowed(net.ParseIP("8.8.8.8")) {
		t.Fatal("8.8.8.8 should be blocked in blacklist")
	}
	if !p.IsIPAccessAllowed(net.ParseIP("1.1.1.1")) {
		t.Fatal("1.1.1.1 should be allowed in blacklist")
	}
}

func TestSSRFProtection_AllowPrivateIP(t *testing.T) {
	p := &SSRFProtection{
		AllowPrivateIp: true,
	}
	if !p.IsIPAccessAllowed(net.ParseIP("10.0.0.1")) {
		t.Fatal("private IP should be allowed when AllowPrivateIp=true")
	}

	p.AllowPrivateIp = false
	if p.IsIPAccessAllowed(net.ParseIP("10.0.0.1")) {
		t.Fatal("private IP should be blocked when AllowPrivateIp=false")
	}
}

func TestSSRFProtection_ValidateURLWithIPWhitelistRejection(t *testing.T) {
	p := &SSRFProtection{
		IpFilterMode: true,
		IpList:       []string{"1.1.1.1"},
	}
	// 8.8.8.8 is not in whitelist → should be rejected
	err := p.ValidateURL("http://8.8.8.8/x")
	if err == nil {
		t.Fatal("expected IP whitelist rejection")
	}
}

func TestSSRFProtection_ValidateURLWithIPBlacklistRejection(t *testing.T) {
	p := &SSRFProtection{
		IpFilterMode: false,
		IpList:       []string{"8.8.8.8"},
	}
	err := p.ValidateURL("http://8.8.8.8/x")
	if err == nil {
		t.Fatal("expected IP blacklist rejection")
	}
}

func TestValidateURLWithFetchSetting_Disabled(t *testing.T) {
	if err := ValidateURLWithFetchSetting("http://10.0.0.1/x", false, false, false, false, nil, nil, nil, false); err != nil {
		t.Fatalf("unexpected error when disabled: %v", err)
	}
}

func TestValidateURLWithFetchSetting_Enabled(t *testing.T) {
	// enabled with private IP → should reject
	err := ValidateURLWithFetchSetting("http://10.0.0.1/x", true, false, false, false, nil, nil, nil, false)
	if err == nil {
		t.Fatal("expected rejection for private IP")
	}

	// enabled with public IP → should pass
	err = ValidateURLWithFetchSetting("http://8.8.8.8/x", true, false, false, false, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// invalid port config
	err = ValidateURLWithFetchSetting("http://8.8.8.8/x", true, false, false, false, nil, nil, []string{"99999"}, false)
	if err == nil {
		t.Fatal("expected error for invalid port config")
	}

	// with allowed ports
	err = ValidateURLWithFetchSetting("http://8.8.8.8:80/x", true, false, false, false, nil, nil, []string{"80"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSRFProtection_ValidateURLDomainWithIPFilter(t *testing.T) {
	p := &SSRFProtection{
		ApplyIPFilterForDomain: true,
		AllowPrivateIp:         false,
	}
	// localhost should resolve to 127.0.0.1 which is private → rejected
	err := p.ValidateURL("http://localhost/x")
	if err == nil {
		t.Fatal("expected rejection for localhost resolving to private IP")
	}
}

func TestSSRFProtection_ValidateURLDomainBlacklistPass(t *testing.T) {
	p := &SSRFProtection{
		DomainFilterMode:       false, // blacklist
		DomainList:             []string{"evil.com"},
		ApplyIPFilterForDomain: false,
	}
	// "example.com" is not in blacklist → should pass domain check,
	// and ApplyIPFilterForDomain=false → no DNS lookup → pass
	err := p.ValidateURL("http://example.com/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSRFProtection_ValidateURLDomainWhitelistPass(t *testing.T) {
	p := &SSRFProtection{
		DomainFilterMode:       true, // whitelist
		DomainList:             []string{"example.com"},
		ApplyIPFilterForDomain: false,
	}
	err := p.ValidateURL("http://example.com/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSRFProtection_ValidateURLDNSFail(t *testing.T) {
	p := &SSRFProtection{
		DomainFilterMode:       false, // blacklist (empty → allow all)
		ApplyIPFilterForDomain: true,
	}
	// Use a domain that definitely won't resolve
	err := p.ValidateURL("http://this-domain-definitely-does-not-exist-xyz123.invalid/x")
	if err == nil {
		t.Fatal("expected DNS resolution failure")
	}
}

func TestSSRFProtection_ValidateURLDomainResolvesPrivate(t *testing.T) {
	p := &SSRFProtection{
		DomainFilterMode:       false,
		ApplyIPFilterForDomain: true,
		AllowPrivateIp:         false,
	}
	// localhost should resolve to 127.0.0.1
	err := p.ValidateURL("http://localhost/x")
	if err == nil {
		t.Fatal("expected rejection for domain resolving to private IP")
	}
}

func TestValidateURLWithFetchSetting_DomainWhitelistReject(t *testing.T) {
	err := ValidateURLWithFetchSetting(
		"http://evil.com/x",
		true,                 // enableSSRFProtection
		false,                // allowPrivateIp
		true,                 // domainFilterMode (whitelist)
		false,                // ipFilterMode
		[]string{"safe.com"}, // domainList
		nil,                  // ipList
		nil,                  // allowedPorts
		false,                // applyIPFilterForDomain
	)
	if err == nil {
		t.Fatal("expected domain whitelist rejection")
	}
}

func TestValidateURLWithFetchSetting_DomainBlacklistPass(t *testing.T) {
	err := ValidateURLWithFetchSetting(
		"http://example.com/x",
		true,                 // enableSSRFProtection
		false,                // allowPrivateIp
		false,                // domainFilterMode (blacklist)
		false,                // ipFilterMode
		[]string{"evil.com"}, // domainList
		nil,                  // ipList
		nil,                  // allowedPorts
		false,                // applyIPFilterForDomain
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateURLWithFetchSetting_IPWhitelist(t *testing.T) {
	err := ValidateURLWithFetchSetting(
		"http://8.8.8.8/x",
		true,                // enableSSRFProtection
		false,               // allowPrivateIp
		false,               // domainFilterMode
		true,                // ipFilterMode (whitelist)
		nil,                 // domainList
		[]string{"1.1.1.1"}, // ipList
		nil,                 // allowedPorts
		false,               // applyIPFilterForDomain
	)
	if err == nil {
		t.Fatal("expected IP whitelist rejection for 8.8.8.8")
	}
}

func TestIsIPListedEmpty(t *testing.T) {
	if isIPListed(net.ParseIP("8.8.8.8"), nil) {
		t.Fatal("empty list should return false")
	}
}

func TestIsAllowedPortNoRestriction(t *testing.T) {
	p := &SSRFProtection{}
	if !p.isAllowedPort(8080) {
		t.Fatal("should allow any port when no restriction")
	}
}
