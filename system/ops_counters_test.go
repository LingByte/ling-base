package system

import "testing"

func TestOpsCounters(t *testing.T) {
	IncPanic()
	IncHTTP5xx()
	IncHTTP4xx()
	IncJWTVerifyFail()
	IncDBReconnect()
	IncPreflightFail()
	IncDependencyFail()
	IncFileIOError()
	IncSSLCertLoadFail()

	c := GetOpsCounters()
	if c.PanicTotal == 0 {
		t.Fatal("panic counter not incremented")
	}
	if c.HTTP5xxTotal == 0 {
		t.Fatal("5xx counter not incremented")
	}
	if c.HTTP4xxTotal == 0 {
		t.Fatal("4xx counter not incremented")
	}
	if c.JWTVerifyFailTotal == 0 {
		t.Fatal("JWTVerifyFail counter not incremented")
	}
	if c.DBReconnectTotal == 0 {
		t.Fatal("DBReconnect counter not incremented")
	}
	if c.PreflightFailTotal == 0 {
		t.Fatal("PreflightFail counter not incremented")
	}
	if c.DependencyFailTotal == 0 {
		t.Fatal("DependencyFail counter not incremented")
	}
	if c.FileIOErrorTotal == 0 {
		t.Fatal("FileIOError counter not incremented")
	}
	if c.SSLCertLoadFailTotal == 0 {
		t.Fatal("SSLCertLoadFail counter not incremented")
	}
}
