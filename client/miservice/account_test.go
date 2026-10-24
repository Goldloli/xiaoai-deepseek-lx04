// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: MIT
package miservice

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestAPICallUsesOnlyRequestedServiceToken(t *testing.T) {
	a := NewAccount("unused", "unused", nil)
	a.token = NewTokens()
	a.token.UserId = "test"
	a.token.Sids["micoapi"] = SidToken{ServiceToken: "correct"}
	u, _ := url.Parse("https://userprofile.mina.mi.com/")
	a.client.Jar.SetCookies(u, []*http.Cookie{{Name: "serviceToken", Value: "another-service"}})
	a.client.Transport = otpTransport(func(r *http.Request) (*http.Response, error) {
		count := 0
		for _, c := range r.Cookies() {
			if c.Name == "serviceToken" {
				count++
				if c.Value != "correct" {
					t.Fatal("another SID's cookie was sent")
				}
			}
		}
		if count != 1 {
			t.Fatalf("got %d serviceToken cookies", count)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"code":0}`)), Request: r}, nil
	})
	var out map[string]any
	if err := a.Request("micoapi", u.String(), nil, nil, nil, false, &out); err != nil {
		t.Fatal(err)
	}
}
