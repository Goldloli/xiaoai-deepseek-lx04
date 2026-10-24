// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
package deepseekweb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchAuthenticatesAndBoundsUnicodeQuery(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-only" || r.Method != "POST" {
			t.Error("invalid authentication or method")
		}
		var in struct {
			Query   string `json:"query"`
			Count   int    `json:"count"`
			Content bool   `json:"need_content"`
		}
		json.NewDecoder(r.Body).Decode(&in)
		if len([]rune(in.Query)) != 128 || in.Count != 5 || in.Content {
			t.Error("invalid request bounds")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[{"title":"Source","url":"https://example.com","summary":"current facts","date":null}]}`))
	}))
	defer s.Close()
	c := NewClient("test-only")
	c.endpoint = s.URL
	r, err := c.Search(context.Background(), strings.Repeat("中", 200), "原问题")
	if err != nil || len(r.Results) != 1 {
		t.Fatal(r, err)
	}
}
func TestEmptySearchNeverBecomesEvidence(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"results":[]}`)) }))
	defer s.Close()
	c := NewClient("test-only")
	c.endpoint = s.URL
	if _, err := c.Search(context.Background(), "latest ranking", "question"); err == nil {
		t.Fatal("empty search accepted")
	}
}
