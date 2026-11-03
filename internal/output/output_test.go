package output

import (
	"strings"
	"testing"
)

func TestFormatStatus(t *testing.T) {
	NoColor = true
	defer func() { NoColor = false }()

	tests := []struct {
		code int
		want string
	}{
		{200, "[200]"},
		{301, "[301]"},
		{404, "[404]"},
		{500, "[500]"},
	}

	for _, tc := range tests {
		got := FormatStatus(tc.code)
		if !strings.Contains(got, tc.want) {
			t.Errorf("FormatStatus(%d) = %s; want to contain %s", tc.code, got, tc.want)
		}
	}
}

func TestLoggingMethodsNoPanic(t *testing.T) {
	NoColor = true
	defer func() { NoColor = false }()

	// Verify functions execute safely without panic
	Info("test info")
	Infof("test infof %s", "arg")
	Success("test success")
	Successf("test successf %s", "arg")
	Warn("test warn")
	Warnf("test warnf %s", "arg")
	Errorf("test errorf %s", "arg")
	Dorkf("test dorkf %s", "arg")
	Grayf("test grayf %s", "arg")
}
