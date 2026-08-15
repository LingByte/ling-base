package system

import "testing"

func TestSetGaugeSetter(t *testing.T) {
	var capturedName string
	var capturedVal float64
	SetGaugeSetter(func(name, help string, labels map[string]string, v float64) {
		capturedName = name
		capturedVal = v
	})
	SyncPrometheusRuntimeGauges()
	if capturedName == "" {
		t.Fatal("gauge setter was not called")
	}
	if capturedVal < 0 {
		t.Fatal("gauge value should be non-negative")
	}

	// restore no-op
	SetGaugeSetter(noopGaugeSetter)
}

func TestNoopGaugeSetter(t *testing.T) {
	// just verify it doesn't panic
	noopGaugeSetter("test", "help", nil, 42.0)
}

func TestNoopGaugeSetterDirect(t *testing.T) {
	noopGaugeSetter("test", "help", map[string]string{"a": "b"}, 123.45)
}
