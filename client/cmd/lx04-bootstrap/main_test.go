// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"testing"
	"xiaobot/miservice"
)

func TestDeviceSelectorRequiresAllSpecifiedIdentifiers(t *testing.T) {
	d := miservice.DeviceData{Hardware: "LX04", SerialNumber: "EXAMPLE/123", Mac: "02:00:00:00:00:01"}
	for _, tc := range []struct {
		selector deviceSelector
		want     bool
	}{
		{deviceSelector{}, false},
		{deviceSelector{SerialNumber: "example123"}, true},
		{deviceSelector{Mac: "020000000001"}, true},
		{deviceSelector{SerialNumber: "wrong", Mac: d.Mac}, false},
		{deviceSelector{SerialNumber: d.SerialNumber, Mac: "02:00:00:00:00:02"}, false},
	} {
		if got := tc.selector.matches(d); got != tc.want {
			t.Fatalf("selector match = %v, want %v", got, tc.want)
		}
	}
	d.Hardware = "OTHER"
	if (deviceSelector{Mac: d.Mac}).matches(d) {
		t.Fatal("wrong hardware accepted")
	}
}
