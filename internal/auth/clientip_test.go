package auth

import (
	"net/http"
	"net/netip"
	"testing"
)

func clientIPTestPrefixes(t *testing.T, values ...string) []netip.Prefix {
	t.Helper()
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			t.Fatalf("ParsePrefix(%q): %v", value, err)
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}

func clientIPTestRequest(remote, forwarded string) *http.Request {
	request := &http.Request{Header: http.Header{}, RemoteAddr: remote}
	if forwarded != "" {
		request.Header.Set("X-Forwarded-For", forwarded)
	}
	return request
}

func TestClientIPResuelveCadenaDeProxies(t *testing.T) {
	trusted := clientIPTestPrefixes(t, "127.0.0.0/8", "10.0.0.0/8", "192.168.0.0/16", "::1/128", "fc00::/7")

	cases := []struct {
		name      string
		remote    string
		forwarded string
		trusted   []netip.Prefix
		want      string
	}{
		{"sin cabecera", "203.0.113.7:4321", "", trusted, "203.0.113.7"},
		{"origen no confiable ignora cabecera", "203.0.113.7:4321", "10.0.0.2", trusted, "203.0.113.7"},
		{"origen confiable", "127.0.0.1:4321", "198.51.100.20", trusted, "198.51.100.20"},
		{"cadena con proxies de confianza", "127.0.0.1:4321", "198.51.100.20, 10.0.0.2", trusted, "198.51.100.20"},
		{"varias cabeceras", "10.0.0.1:4321", "198.51.100.20, 10.0.0.2", trusted, "198.51.100.20"},
		{"basura a la derecha", "127.0.0.1:4321", "198.51.100.20, no-es-ip", trusted, "198.51.100.20"},
		{"solo basura", "127.0.0.1:4321", "no-es-ip, otra", trusted, "127.0.0.1"},
		{"ipv6", "[::1]:4321", "2001:db8::20, fd00::1", trusted, "2001:db8::20"},
		{"sin puerto", "203.0.113.7", "", trusted, "203.0.113.7"},
		{"ipv4 mapeada", "[::ffff:203.0.113.7]:4321", "", trusted, "203.0.113.7"},
		{"remoto vacio", "", "198.51.100.20", trusted, ""},
		{"sin confianza", "127.0.0.1:4321", "198.51.100.20", nil, "127.0.0.1"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			request := clientIPTestRequest(test.remote, test.forwarded)
			if got := ClientIP(request, test.trusted); got != test.want {
				t.Fatalf("ClientIP = %q, se esperaba %q", got, test.want)
			}
		})
	}
}

func TestClientIPConfiaEnTodosLosProxiesDeLaCadena(t *testing.T) {
	trusted := clientIPTestPrefixes(t, "10.0.0.0/8")
	request := clientIPTestRequest("10.0.0.1:1234", "203.0.113.9, 10.0.0.8, 10.0.0.7")
	if got := ClientIP(request, trusted); got != "203.0.113.9" {
		t.Fatalf("ClientIP = %q, se esperaba el cliente original", got)
	}
}

func TestClientIPConRangosPorDefecto(t *testing.T) {
	prefixes := DefaultTrustedProxies()
	if len(prefixes) == 0 {
		t.Fatal("debe haber rangos de confianza por defecto")
	}
	request := clientIPTestRequest("172.18.0.2:4000", "198.51.100.4")
	if got := ClientIP(request, prefixes); got != "198.51.100.4" {
		t.Fatalf("ClientIP = %q, se esperaba la IP reenviada", got)
	}
}
