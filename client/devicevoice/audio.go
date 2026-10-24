// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
package devicevoice

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// These transaction IDs were checked against this LX04's Android 8.1 service.
// Only master playback mute is used; microphone state and volume are untouched.
type AudioGate struct {
	mu       sync.Mutex
	Path     string
	active   bool
	original bool
}

var parcel = regexp.MustCompile(`Parcel\(\s*([0-9a-fA-F]{8})(?:\s+([0-9a-fA-F]{8}))?`)

func audioCall(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/system/bin/service", append([]string{"call", "audio"}, args...)...).CombinedOutput()
	m := parcel.FindStringSubmatch(string(out))
	if err != nil || len(m) < 2 || m[1] != "00000000" {
		return "", errors.New("audio service operation failed")
	}
	return string(out), nil
}
func masterMute() (bool, error) {
	s, err := audioCall("6")
	if err != nil {
		return false, err
	}
	m := parcel.FindStringSubmatch(s)
	if len(m) < 3 || (m[2] != "00000000" && m[2] != "00000001") {
		return false, errors.New("unexpected audio service reply")
	}
	return m[2] == "00000001", nil
}
func setMasterMute(mute bool) error {
	v := "0"
	if mute {
		v = "1"
	}
	_, err := audioCall("7", "i32", v, "i32", "0", "s16", "com.android.shell", "i32", "0")
	return err
}

func (g *AudioGate) Mute() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.active {
		return nil
	}
	was, err := masterMute()
	if err != nil {
		return err
	}
	g.original = was
	b, _ := json.Marshal(struct {
		Muted bool `json:"muted"`
	}{was})
	if err = os.WriteFile(g.Path, b, 0600); err != nil {
		return err
	}
	g.active = true
	return setMasterMute(true)
}
func (g *AudioGate) Restore() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.active {
		b, err := os.ReadFile(g.Path)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		var saved struct {
			Muted *bool `json:"muted"`
		}
		if json.Unmarshal(b, &saved) != nil || saved.Muted == nil {
			return errors.New("invalid audio recovery file")
		}
		g.original = *saved.Muted
	}
	if err := setMasterMute(g.original); err != nil {
		return err
	}
	g.active = false
	return os.Remove(g.Path)
}
func SupportedAudioService() bool {
	b, err := exec.Command("/system/bin/getprop", "ro.build.version.sdk").Output()
	if err != nil || strings.TrimSpace(string(b)) != "27" {
		return false
	}
	_, err = masterMute()
	return err == nil
}
