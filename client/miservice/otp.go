// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: MIT
package miservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// SetOTPProvider is used only by the interactive bootstrap, never the daemon.
func (ma *Account) SetOTPProvider(provider func(string) (string, error)) { ma.otpProvider = provider }

func accountURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("invalid Xiaomi verification URL")
	}
	u = (&url.URL{Scheme: "https", Host: "account.xiaomi.com"}).ResolveReference(u)
	if u.Scheme != "https" || u.Host != "account.xiaomi.com" || u.User != nil {
		return "", errors.New("unexpected Xiaomi verification host")
	}
	return u.String(), nil
}

type otpResponse struct {
	Code     *int   `json:"code"`
	Flag     int    `json:"flag"`
	Location string `json:"location"`
}

type otpSession struct {
	Cookies      []*http.Cookie `json:"cookies"`
	DeviceID     string         `json:"device_id"`
	Continuation string         `json:"continuation"`
}

// RestoreVerificationSession reuses the owner's pending verification cookies.
func (ma *Account) RestoreVerificationSession(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var s otpSession
	if json.Unmarshal(b, &s) != nil || s.DeviceID == "" {
		return errors.New("invalid private verification session")
	}
	ma.token = NewTokens()
	ma.token.UserName = ma.username
	ma.token.DeviceId = s.DeviceID
	u, _ := url.Parse("https://account.xiaomi.com/")
	ma.client.Jar.SetCookies(u, s.Cookies)
	if s.Continuation != "" {
		_, final, err := ma.otpRequest(s.Continuation, nil, nil, false)
		if err != nil {
			return err
		}
		if confirm := final.Query().Get("confirmUrl"); confirm != "" {
			_, _, err = ma.otpRequest(confirm, nil, nil, false)
			return err
		}
	}
	return nil
}

func (ma *Account) otpRequest(raw string, data url.Values, cookies []*http.Cookie, decode bool) (*otpResponse, *url.URL, error) {
	u, err := accountURL(raw)
	if err != nil {
		return nil, nil, err
	}
	method := http.MethodGet
	var body io.Reader
	if data != nil {
		method = http.MethodPost
		body = strings.NewReader(data.Encode())
	}
	req, _ := http.NewRequest(method, u, body)
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 18_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 iOS/26.3.1 MiHome/11.3.203 "+UA)
	if data != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	resp, err := ma.client.Do(req)
	if err != nil {
		return nil, nil, errors.New("Xiaomi verification network request failed")
	}
	defer resp.Body.Close()
	if os.Getenv("XIAOBOT_OTP_DIAGNOSTICS") == "1" {
		u, _ := url.Parse("https://account.xiaomi.com/")
		state := otpSession{Cookies: ma.client.Jar.Cookies(u), DeviceID: ma.token.DeviceId}
		if req.URL.Path == "/pass/serviceLoginAuth2/end" {
			state.Continuation = req.URL.String()
		}
		b, _ := json.Marshal(state)
		os.WriteFile(filepath.Join(os.Getenv("HOME"), "otp-session.json"), b, 0600)
	}
	// The continuation can finish at a service /sts endpoint with HTTP 401
	// before clientSign is added. Its account cookies are still valid; resume
	// serviceLogin next, as the upstream MiService implementation does.
	if resp.StatusCode != http.StatusOK && decode {
		return nil, nil, fmt.Errorf("Xiaomi verification HTTP %d at %s (final %s)", resp.StatusCode, req.URL.Path, resp.Request.URL.Path)
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, nil, errors.New("Xiaomi verification response read failed")
	}
	result := &otpResponse{}
	if os.Getenv("XIAOBOT_OTP_DIAGNOSTICS") == "1" {
		// Temporary owner-only diagnostics, never stdout or the normal daemon log.
		name := "otp-" + strings.ReplaceAll(strings.Trim(req.URL.Path, "/"), "/", "-") + ".response"
		os.WriteFile(filepath.Join(os.Getenv("HOME"), name), payload, 0600)
	}
	if decode {
		if json.Unmarshal([]byte(strings.TrimPrefix(string(payload), "&&&START&&&")), result) != nil {
			return nil, nil, errors.New("Xiaomi verification response is not JSON")
		}
		// /identity/list uses code=2 for a valid verification-method listing.
		listing := req.URL.Path == "/identity/list" && (result.Flag == 4 || result.Flag == 8)
		if result.Code != nil && *result.Code != 0 && !listing {
			return nil, nil, fmt.Errorf("Xiaomi verification rejected at %s (code %d)", req.URL.Path, *result.Code)
		}
	}
	return result, resp.Request.URL, nil
}

func (ma *Account) verifyOTP(sid, notification string, cookies []*http.Cookie) (*loginResp, error) {
	ntf, err := accountURL(notification)
	if err != nil {
		return nil, err
	}
	if _, _, err = ma.otpRequest(ntf, nil, cookies, false); err != nil {
		return nil, err
	}
	u, _ := url.Parse(ntf)
	query := url.Values{"sid": {sid}, "supportedMask": {"0"}, "_locale": {"zh_CN"}, "context": {u.Query().Get("context")}}
	methods, _, err := ma.otpRequest("/identity/list?"+query.Encode(), nil, cookies, true)
	if err != nil {
		return nil, err
	}
	flag, method := methods.Flag, "Phone"
	if flag == 0 {
		flag = 4
	}
	if flag == 8 {
		method = "Email"
	} else if flag != 4 {
		return nil, errors.New("Xiaomi verification method is unsupported")
	}
	if _, _, err = ma.otpRequest(fmt.Sprintf("/identity/auth/verify%s?_flag=%d&_json=true", method, flag), nil, cookies, true); err != nil {
		return nil, err
	}
	if method == "Phone" {
		if _, _, err = ma.otpRequest("/identity/auth/sendPhoneTicket", url.Values{"retry": {"0"}, "icode": {""}, "_json": {"true"}}, cookies, true); err != nil {
			return nil, err
		}
	}
	code, err := ma.otpProvider(method)
	if err != nil {
		return nil, err
	}
	verified, _, err := ma.otpRequest("/identity/auth/verify"+method, url.Values{"_flag": {fmt.Sprint(flag)}, "ticket": {strings.TrimSpace(code)}, "trust": {"false"}, "_json": {"true"}}, cookies, true)
	if err != nil {
		return nil, err
	}
	if verified.Location == "" {
		return nil, errors.New("Xiaomi verification did not return a continuation")
	}
	_, finalURL, err := ma.otpRequest(verified.Location, nil, cookies, false)
	if err != nil {
		return nil, err
	}
	if confirm := finalURL.Query().Get("confirmUrl"); confirm != "" {
		if _, _, err = ma.otpRequest(confirm, nil, cookies, false); err != nil {
			return nil, err
		}
	}
	resumed, err := ma.serviceLogin("serviceLogin?sid="+url.QueryEscape(sid)+"&_json=true", nil, cookies)
	if err != nil {
		return nil, err
	}
	if resumed.Code != 0 || resumed.NotificationURL != "" {
		return nil, errors.New("Xiaomi login did not resume after verification")
	}
	return resumed, nil
}
