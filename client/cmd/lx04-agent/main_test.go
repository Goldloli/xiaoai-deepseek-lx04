// SPDX-License-Identifier: Apache-2.0
package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"xiaobot/miservice"
)

func TestSafeErrorOmitsSignedLoginURL(t *testing.T) {
	err := fmt.Errorf("login: %w", &url.Error{Op: "Get", URL: "https://example.invalid/sts?clientSign=private", Err: errors.New("network is unreachable")})
	got := safeError(err)
	if strings.Contains(got, "private") || strings.Contains(got, "clientSign") || !strings.Contains(got, "network is unreachable") {
		t.Fatalf("unsafe or unhelpful diagnostic: %s", got)
	}
	err = &miservice.APIError{HTTPStatusCode: 429, Code: -127}
	if got = safeError(err); !strings.Contains(got, "HTTP 429, code -127") {
		t.Fatalf("lost service rejection code: %s", got)
	}
}
