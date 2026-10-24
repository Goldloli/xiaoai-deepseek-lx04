// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
// Authenticate on the speaker, select only the user's known LX04, and test TTS.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xiaobot"
	"xiaobot/miservice"
	"xiaobot/runtimeenv"
)

func main() {
	path := flag.String("config", "", "private config file")
	devicePath := flag.String("device-file", "", "private JSON with serial_number and/or mac; exact match required")
	speak := flag.Bool("speak", false, "send a bounded speech test")
	resume := flag.Bool("resume", false, "reuse the private verification session")
	history := flag.Bool("check-history", false, "check conversation access without logging conversations")
	flag.Parse()
	if *path == "" || *devicePath == "" {
		fmt.Fprintln(os.Stderr, "-config and -device-file are required; refusing automatic device selection")
		os.Exit(2)
	}
	var selector deviceSelector
	data, err := os.ReadFile(*devicePath)
	if err != nil || json.Unmarshal(data, &selector) != nil || !selector.valid() {
		fmt.Fprintln(os.Stderr, "private device selector missing or invalid")
		os.Exit(2)
	}
	dir := filepath.Dir(*path)
	status := map[string]any{}
	finish := func(state string, code int) {
		status["status"] = state
		data, _ := json.MarshalIndent(status, "", "  ")
		os.WriteFile(filepath.Join(dir, "bootstrap-status.json"), append(data, '\n'), 0600)
		os.Exit(code)
	}
	if runtimeenv.ConfigureDNSFromEnv() != nil {
		finish("dns_config_invalid", 1)
	}
	config, err := xiaobot.NewConfigFromFile(*path)
	if err != nil || config.Account == "" || config.Password == "" {
		finish("credentials_missing", 1)
	}
	config.TokenPath = filepath.Join(dir, ".mi.token")
	account := miservice.NewAccount(config.Account, config.Password, miservice.NewTokenStore(config.TokenPath))
	if *resume {
		if err := account.RestoreVerificationSession(filepath.Join(dir, "otp-session.json")); err != nil {
			status["error"] = err.Error()
			finish("verification_resume_failed", 1)
		}
	}
	account.SetOTPProvider(func(method string) (string, error) {
		status["status"] = "verification_required"
		status["verification_method"] = method
		data, _ := json.MarshalIndent(status, "", "  ")
		os.WriteFile(filepath.Join(dir, "bootstrap-status.json"), data, 0600)
		codePath := filepath.Join(dir, "otp.txt")
		for deadline := time.Now().Add(10 * time.Minute); time.Now().Before(deadline); time.Sleep(time.Second) {
			data, err := os.ReadFile(codePath)
			if err == nil && strings.TrimSpace(string(data)) != "" {
				os.Remove(codePath)
				return strings.TrimSpace(string(data)), nil
			}
		}
		return "", fmt.Errorf("verification input timed out")
	})
	service := miservice.NewAIService(account)
	devices, err := service.DeviceList(0)
	if err != nil {
		// Library errors have been sanitized; never serialize raw API responses.
		status["error"] = err.Error()
		var login *miservice.LoginError
		if errors.As(err, &login) {
			status["login_code"] = login.Code
			if login.VerificationURL != "" || login.CaptchaURL != "" {
				data, _ := json.Marshal(login)
				os.WriteFile(filepath.Join(dir, "login-verification.json"), data, 0600)
				status["verification_file_created"] = true
			}
		}
		finish("xiaomi_login_or_device_list_failed", 1)
	}
	matches := []miservice.DeviceData{}
	for _, d := range devices {
		if selector.matches(d) {
			matches = append(matches, d)
		}
	}
	status["exact_matches"] = len(matches)
	if len(matches) != 1 {
		finish("speaker_match_failed", 1)
	}
	if matches[0].MiotDID == "" {
		finish("matched_speaker_has_no_miot_did", 1)
	}
	config.MiDID = matches[0].MiotDID
	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil || os.WriteFile(*path, append(data, '\n'), 0600) != nil {
		finish("config_save_failed", 1)
	}
	os.Chmod(*path, 0600)
	status["did_configured"] = true
	if *history {
		u := strings.ReplaceAll(xiaobot.LatestAskApi, "{hardware}", "LX04")
		u = strings.ReplaceAll(u, "{timestamp}", fmt.Sprint(time.Now().UnixMilli()))
		var result struct {
			Code int    `json:"code"`
			Data string `json:"data"`
		}
		err := account.Request("micoapi", u, nil, func(_ *miservice.Tokens, cookie map[string]string) url.Values {
			cookie["deviceId"] = matches[0].DeviceID
			return nil
		}, nil, true, &result)
		if err != nil {
			status["error"] = err.Error()
			finish("conversation_access_failed", 1)
		}
		status["conversation_access"] = true
	}
	if *speak {
		box := xiaobot.NewXiaoMi(config)
		if box.InitMiBox() != nil {
			finish("speaker_initialization_failed", 1)
		}
		if box.MiTTS("DeepSeek 接入测试，语音播报连接成功。") != nil {
			finish("speech_test_failed", 1)
		}
		status["speech_test_sent"] = true
	}
	finish("ready", 0)
}

type deviceSelector struct {
	SerialNumber string `json:"serial_number"`
	Mac          string `json:"mac"`
}

func normalize(v string) string {
	return strings.NewReplacer(":", "", "-", "", "/", "", " ", "").Replace(strings.ToUpper(v))
}

func (s deviceSelector) valid() bool {
	return normalize(s.SerialNumber) != "" || normalize(s.Mac) != ""
}

func (s deviceSelector) matches(d miservice.DeviceData) bool {
	return s.valid() && d.Hardware == "LX04" &&
		(normalize(s.SerialNumber) == "" || normalize(s.SerialNumber) == normalize(d.SerialNumber)) &&
		(normalize(s.Mac) == "" || normalize(s.Mac) == normalize(d.Mac))
}
