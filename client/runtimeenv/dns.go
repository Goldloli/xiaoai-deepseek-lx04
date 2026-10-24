// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
// Package runtimeenv provides process-local networking for Android's userspace.
package runtimeenv

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

// ConfigureDNSFromEnv avoids needing to modify Android's read-only /etc.
// Android has no resolv.conf; the launcher supplies its existing Wi-Fi DNS server.
func ConfigureDNSFromEnv() error {
	server := os.Getenv("XIAOBOT_DNS_SERVER")
	if server == "" {
		return nil
	}
	host, port, err := net.SplitHostPort(server)
	if err != nil || net.ParseIP(host) == nil {
		return fmt.Errorf("XIAOBOT_DNS_SERVER must be a literal IP and port")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("XIAOBOT_DNS_SERVER has an invalid port")
	}
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, server)
		},
	}
	return nil
}
