package openai

import (
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientRejectsUntrustedTLSCertificate(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := NewClient("", server.URL, "")
	if response, err := client.client.Get(server.URL); err == nil {
		response.Body.Close()
		t.Fatal("accepted an untrusted TLS certificate")
	}
	// The same local endpoint must work once its certificate is explicitly trusted.
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	client.client.Transport.(*http.Transport).TLSClientConfig.RootCAs = roots
	response, err := client.client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
}
