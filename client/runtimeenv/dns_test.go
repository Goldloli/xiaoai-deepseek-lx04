// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
package runtimeenv

import (
	"context"
	"net"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

func TestDNSUsesConfiguredServerWithoutResolvConf(t *testing.T) {
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	go func() {
		buf := make([]byte, 4096)
		for {
			n, from, err := server.ReadFrom(buf)
			if err != nil {
				return
			}
			var query dnsmessage.Message
			if query.Unpack(buf[:n]) != nil {
				continue
			}
			response := dnsmessage.Message{Header: dnsmessage.Header{ID: query.ID, Response: true}, Questions: query.Questions}
			for _, q := range query.Questions {
				if q.Type == dnsmessage.TypeA {
					response.Answers = append(response.Answers, dnsmessage.Resource{
						Header: dnsmessage.ResourceHeader{Name: q.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 1},
						Body:   &dnsmessage.AResource{A: [4]byte{203, 0, 113, 7}},
					})
				}
			}
			data, err := response.Pack()
			if err == nil {
				server.WriteTo(data, from)
			}
		}
	}()
	previous := net.DefaultResolver
	defer func() { net.DefaultResolver = previous }()
	t.Setenv("XIAOBOT_DNS_SERVER", server.LocalAddr().String())
	if err := ConfigureDNSFromEnv(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, "speaker-test.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 1 || ips[0].IP.String() != "203.0.113.7" {
		t.Fatalf("unexpected result: %v", ips)
	}
}

func TestDNSRejectsHostnamesAndInvalidPorts(t *testing.T) {
	for _, value := range []string{"router.local:53", "192.168.0.1:0", "192.168.0.1:70000", "192.168.0.1"} {
		t.Setenv("XIAOBOT_DNS_SERVER", value)
		if ConfigureDNSFromEnv() == nil {
			t.Fatalf("accepted invalid DNS server %q", value)
		}
	}
}
