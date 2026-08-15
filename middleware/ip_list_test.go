package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIPList_Empty(t *testing.T) {
	l := ParseIPList("")
	assert.True(t, l.Empty())
}

func TestParseIPList_Any(t *testing.T) {
	l := ParseIPList("*")
	assert.True(t, l.Any)
	assert.False(t, l.Empty())
}

func TestParseIPList_SingleIP(t *testing.T) {
	l := ParseIPList("192.168.1.1")
	assert.False(t, l.Empty())
	assert.Len(t, l.IPs, 1)
	assert.True(t, l.Contains("192.168.1.1"))
	assert.False(t, l.Contains("10.0.0.1"))
}

func TestParseIPList_MultipleIPs(t *testing.T) {
	l := ParseIPList("192.168.1.1, 10.0.0.1, 172.16.0.1")
	assert.Len(t, l.IPs, 3)
	assert.True(t, l.Contains("192.168.1.1"))
	assert.True(t, l.Contains("10.0.0.1"))
	assert.True(t, l.Contains("172.16.0.1"))
}

func TestParseIPList_CIDR(t *testing.T) {
	l := ParseIPList("192.168.0.0/16")
	assert.Len(t, l.Nets, 1)
	assert.True(t, l.Contains("192.168.1.100"))
	assert.False(t, l.Contains("10.0.0.1"))
}

func TestParseIPList_InvalidIP(t *testing.T) {
	l := ParseIPList("not-an-ip")
	assert.True(t, l.Empty())
}

func TestParseIPList_InvalidCIDR(t *testing.T) {
	l := ParseIPList("192.168.0.0/99")
	assert.Empty(t, l.Nets)
}

func TestParseIPList_Mixed(t *testing.T) {
	l := ParseIPList("192.168.1.1, 10.0.0.0/8, *, invalid")
	assert.True(t, l.Any)
	assert.Len(t, l.IPs, 1)
	assert.Len(t, l.Nets, 1)
}

func TestIPList_Contains_InvalidIP(t *testing.T) {
	l := ParseIPList("192.168.1.1")
	assert.False(t, l.Contains("not-an-ip"))
}

func TestIPList_Contains_Any(t *testing.T) {
	l := ParseIPList("*")
	assert.True(t, l.Contains("1.2.3.4"))
	assert.True(t, l.Contains("anything"))
}

func TestIPACLDecision_Blocked(t *testing.T) {
	blocked := ParseIPList("10.0.0.1")
	allowed := IPList{}
	result, reason := IPACLDecision(blocked, allowed, IPList{}, IPList{}, "10.0.0.1")
	assert.False(t, result)
	assert.Equal(t, "blocked", reason)
}

func TestIPACLDecision_RouteBlocked(t *testing.T) {
	routeBlocked := ParseIPList("10.0.0.1")
	result, reason := IPACLDecision(IPList{}, IPList{}, routeBlocked, IPList{}, "10.0.0.1")
	assert.False(t, result)
	assert.Equal(t, "blocked", reason)
}

func TestIPACLDecision_RouteAllowlist(t *testing.T) {
	routeAllowed := ParseIPList("10.0.0.1")
	result, reason := IPACLDecision(IPList{}, IPList{}, IPList{}, routeAllowed, "10.0.0.1")
	assert.True(t, result)
	assert.Empty(t, reason)

	result2, reason2 := IPACLDecision(IPList{}, IPList{}, IPList{}, routeAllowed, "10.0.0.2")
	assert.False(t, result2)
	assert.Equal(t, "route_allowlist", reason2)
}

func TestIPACLDecision_GlobalAllowlist(t *testing.T) {
	globalAllowed := ParseIPList("10.0.0.1")
	result, reason := IPACLDecision(IPList{}, globalAllowed, IPList{}, IPList{}, "10.0.0.1")
	assert.True(t, result)
	assert.Empty(t, reason)

	result2, reason2 := IPACLDecision(IPList{}, globalAllowed, IPList{}, IPList{}, "10.0.0.2")
	assert.False(t, result2)
	assert.Equal(t, "global_allowlist", reason2)
}

func TestIPACLDecision_NoRestrictions(t *testing.T) {
	result, reason := IPACLDecision(IPList{}, IPList{}, IPList{}, IPList{}, "10.0.0.1")
	assert.True(t, result)
	assert.Empty(t, reason)
}
