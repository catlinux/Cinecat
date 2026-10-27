package auth

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// defaultTrustedProxyCIDRs son los orígenes directos de los que se acepta
// X-Forwarded-For cuando no se configura otra cosa: loopback y redes privadas.
var defaultTrustedProxyCIDRs = []string{
	"127.0.0.0/8",
	"::1/128",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"fc00::/7",
}

// DefaultTrustedProxies devuelve una copia de los rangos de confianza por
// defecto. Se usa también desde la configuración de arranque.
func DefaultTrustedProxies() []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(defaultTrustedProxyCIDRs))
	for _, raw := range defaultTrustedProxyCIDRs {
		prefix, err := netip.ParsePrefix(raw)
		if err == nil {
			prefixes = append(prefixes, prefix.Masked())
		}
	}
	return prefixes
}

// ClientIP devuelve la IP real del cliente. Solo confía en X-Forwarded-For si
// la conexión directa llega desde un proxy de confianza; así un cliente
// externo no puede inventarse la cabecera. Recorre la lista de derecha a
// izquierda y se queda con la primera dirección que no es de confianza, que es
// el cliente original aunque haya varios proxies encadenados.
func ClientIP(r *http.Request, trusted []netip.Prefix) string {
	if r == nil {
		return ""
	}
	remote, ok := parseClientIP(r.RemoteAddr)
	if !ok {
		return ""
	}
	if !isTrustedProxy(remote, trusted) {
		return remote.String()
	}

	values := r.Header.Values("X-Forwarded-For")
	if len(values) == 0 {
		return remote.String()
	}
	candidates := make([]string, 0)
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if candidate := strings.TrimSpace(part); candidate != "" {
				candidates = append(candidates, candidate)
			}
		}
	}
	for index := len(candidates) - 1; index >= 0; index-- {
		address, ok := parseClientIP(candidates[index])
		if !ok {
			continue
		}
		if isTrustedProxy(address, trusted) {
			continue
		}
		return address.String()
	}
	return remote.String()
}

func parseClientIP(value string) (netip.Addr, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return netip.Addr{}, false
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.TrimPrefix(strings.TrimSuffix(value, "]"), "[")
	address, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Addr{}, false
	}
	return address.Unmap(), true
}

func isTrustedProxy(address netip.Addr, trusted []netip.Prefix) bool {
	address = address.Unmap()
	for _, prefix := range trusted {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}
