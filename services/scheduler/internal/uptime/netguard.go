// Package uptime probes HTTP monitors, tracks their state and delivers
// notifications. It runs inside the scheduler service next to the cron Runner.
package uptime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ErrPrivateAddress is returned when a destination resolves to a loopback,
// private, link-local or otherwise non-public address and private targets are
// not allowed. It is the SSRF guard for user-supplied URLs.
var ErrPrivateAddress = errors.New("uptime: destination resolves to a private or local address")

// Resolver is the subset of net.Resolver used to look up probe targets.
// Tests inject a fake to steer names at local servers.
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// IsDisallowedIP reports whether ip must not be contacted by a probe:
// loopback, RFC 1918 and IPv6 ULA, link-local, unspecified, multicast and
// carrier-grade NAT (100.64.0.0/10).
func IsDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		return v4[0] == 100 && v4[1]&0xc0 == 64 // 100.64.0.0/10
	}
	return false
}

// newTransport returns a Transport that resolves the host first and refuses
// non-public addresses unless allowPrivate is set. Connections are made to the
// resolved IP (not the name again) so a DNS rebinding cannot bypass the check.
func newTransport(allowPrivate bool, resolver Resolver) *http.Transport {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			addrs, err := resolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(addrs) == 0 {
				return nil, fmt.Errorf("uptime: no addresses for %q", host)
			}
			if !allowPrivate {
				for _, a := range addrs {
					if IsDisallowedIP(a.IP) {
						return nil, ErrPrivateAddress
					}
				}
			}
			var lastErr error
			for _, a := range addrs {
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(a.IP.String(), port))
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			return nil, lastErr
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 0,
		DisableKeepAlives:     true,
		ForceAttemptHTTP2:     true,
	}
}

// NewHTTPClient returns a client guarded by newTransport with no overall
// timeout; callers bound requests with a context.
func NewHTTPClient(allowPrivate bool) *http.Client {
	return &http.Client{Transport: newTransport(allowPrivate, nil)}
}
