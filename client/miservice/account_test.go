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

func TestMiIORenewsExpiredTokenWithCurrentClient(t *testing.T) {
	store := &DummyTokenStore{}
	a := NewAccount("owner", "unused", store)
	a.token = NewTokens()
	a.token.UserName = "owner"
	a.token.UserId = "1"
	a.token.PassToken = "verified-identity"
	a.token.DeviceId = "same-device"
	a.token.Sids[MIIO_SID] = SidToken{Ssecurity: "b2xk", ServiceToken: "expired"}
	a.token.Sids["micoapi"] = SidToken{ServiceToken: "other-service"}
	apiCalls, logins := 0, 0
	a.client.Transport = otpTransport(func(r *http.Request) (*http.Response, error) {
		status, body := http.StatusOK, `{"code":0,"result":{"code":0}}`
		headers := make(http.Header)
		switch r.URL.Hostname() {
		case "api.io.mi.com":
			apiCalls++
			// The server withdrew support for the legacy client identifier.
			if strings.Contains(r.UserAgent(), "6.0.103") {
				status, body = http.StatusTooManyRequests, `{"code":-127,"message":"client withdraw"}`
				break
			}
			cookie, err := r.Cookie("serviceToken")
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, c := range r.Cookies() {
				if c.Name == "serviceToken" {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("renewal mixed expired and current cookies: %d tokens", count)
			}
			if apiCalls == 1 {
				if cookie.Value != "expired" {
					t.Fatal("unexpected initial session")
				}
				status, body = http.StatusUnauthorized, `{"code":3,"message":"auth error"}`
			} else {
				if apiCalls != 2 || cookie.Value != "renewed" {
					t.Fatalf("request did not use the renewed session exactly once: calls=%d cookie=%q", apiCalls, cookie.Value)
				}
				r.ParseForm()
				if !strings.Contains(r.Form.Get("data"), `"did":"configured-speaker"`) {
					t.Fatal("renewal lost the original speech request")
				}
			}
		case "account.xiaomi.com":
			logins++
			cookie, err := r.Cookie("passToken")
			if err != nil || cookie.Value != "verified-identity" || r.Method != http.MethodGet {
				t.Fatal("service renewal must retain the verified identity")
			}
			body = `{"code":0,"userId":1,"passToken":"verified-identity","ssecurity":"bmV3","location":"https://sts.api.io.mi.com/sts?sid=xiaomiio","nonce":0}`
		case "sts.api.io.mi.com":
			headers.Add("Set-Cookie", "serviceToken=renewed; Path=/")
			body = ""
		default:
			t.Fatalf("unexpected host %s", r.URL.Hostname())
		}
		return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	code, err := NewIOService(a, nil).MiotAction("configured-speaker", []int{5, 1}, []any{"test"})
	if err != nil || code != 0 {
		t.Fatalf("speech did not recover: code=%v err=%v", code, err)
	}
	if apiCalls != 2 || logins != 1 || store.tokens == nil || store.tokens.Sids["micoapi"].ServiceToken != "other-service" {
		t.Fatal("renewal must be bounded, persistent, and isolated to the expired service")
	}
}

func TestHTTPFailureRetainsXiaomiServiceCode(t *testing.T) {
	a := NewAccount("unused", "unused", nil)
	a.token = NewTokens()
	a.token.Sids["micoapi"] = SidToken{ServiceToken: "test"}
	a.client.Transport = otpTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"code":-127,"message":"client withdraw"}`)), Request: r}, nil
	})
	var out map[string]any
	err := a.Request("micoapi", "https://api2.mina.mi.com/test", nil, nil, nil, true, &out)
	if err == nil || !strings.Contains(err.Error(), "HTTP 429, code -127") {
		t.Fatalf("lost server rejection code: %v", err)
	}
}
