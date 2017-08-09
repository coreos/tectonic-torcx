package internal

import "testing"

func TestParseOSVersion(t *testing.T) {
	inp := `NAME="Container Linux by CoreOS"
ID=coreos
VERSION=1465.0.0
VERSION_ID=1465.0.0
BUILD_ID=2017-07-06-0206
PRETTY_NAME="Container Linux by CoreOS 1465.0.0 (Ladybug)"
ANSI_COLOR="38;5;75"
HOME_URL="https://coreos.com/"
BUG_REPORT_URL="https://issues.coreos.com"
COREOS_BOARD="amd64-usr"`

	vers := parseOSVersion(inp)
	expected := "1465.0.0"

	if vers != expected {
		t.Fatalf("parseOSVersion expected %q, got %q", expected, vers)
	}
}
