// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: MIT
package miservice

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type otpTransport func(*http.Request) (*http.Response, error)

func (f otpTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAccountURLRejectsOtherHosts(t *testing.T) {
	for _, raw := range []string{"http://account.xiaomi.com/x", "https://evil.example/x", "//evil.example/x", "https://user@account.xiaomi.com/x"} {
		if _, err := accountURL(raw); err == nil {
			t.Errorf("accepted unsafe URL %q", raw)
		}
	}
	if got, err := accountURL("/identity/list"); err != nil || got != "https://account.xiaomi.com/identity/list" {
		t.Fatal(got, err)
	}
}

func TestSMSVerificationResumesSession(t *testing.T) {
	paths := []string{"/fe/service/identity/authStart", "/identity/list", "/identity/auth/verifyPhone", "/identity/auth/sendPhoneTicket", "/identity/auth/verifyPhone", "/pass/resume", "/pass/serviceLogin"}
	i := 0
	a := NewAccount("unused", "unused", nil)
	a.client.Transport = otpTransport(func(r *http.Request) (*http.Response, error) {
		if i >= len(paths) || r.URL.Path != paths[i] {
			t.Fatalf("unexpected step %d: %s", i, r.URL.Path)
		}
		i++
		body := `{"code":0}`
		switch i {
		case 2:
			if r.URL.Query().Get("context") != "session" {
				t.Fatal("lost verification context")
			}
			body = `&&&START&&&{"code":2,"flag":4}`
		case 4:
			if r.Method != http.MethodPost {
				t.Fatal("SMS not sent by POST")
			}
		case 5:
			r.ParseForm()
			if r.Form.Get("ticket") != "123456" || r.Form.Get("trust") != "false" {
				t.Fatal("invalid verification submission")
			}
			body = `{"code":0,"location":"/pass/resume"}`
		case 7:
			body = `&&&START&&&{"code":0,"userId":1,"location":"https://api2.mina.mi.com/sts?x=1","ssecurity":"test","passToken":"test"}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	a.SetOTPProvider(func(method string) (string, error) {
		if method != "Phone" || i != 4 {
			t.Fatal("OTP requested before SMS was sent")
		}
		return "123456", nil
	})
	r, err := a.verifyOTP("micoapi", "/fe/service/identity/authStart?context=session", nil)
	if err != nil || r.UserID != 1 || i != len(paths) {
		t.Fatalf("flow failed: %v, steps=%d", err, i)
	}
}
