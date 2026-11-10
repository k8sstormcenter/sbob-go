package match

import (
	"net/netip"
	"strings"
)

const AnyIPSentinel = "*"

type IPMatcher struct {
	any      bool
	literals []netip.Addr
	prefixes []netip.Prefix
}

func parseAddr(s string) (netip.Addr, bool) {
	a, err := netip.ParseAddr(s)
	if err != nil || a.Zone() != "" {
		return netip.Addr{}, false
	}
	return a.Unmap(), true
}

func parsePrefix(s string) (netip.Prefix, bool) {
	p, err := netip.ParsePrefix(s)
	if err != nil || p.Addr().Zone() != "" {
		return netip.Prefix{}, false
	}
	if p.Addr().Is4In6() {
		bits := p.Bits() - 96
		if bits < 0 {
			bits = 0
		}
		p = netip.PrefixFrom(p.Addr().Unmap(), bits)
	}
	return p.Masked(), true
}

func CompileIP(profileEntries []string) *IPMatcher {
	m := &IPMatcher{}
	for _, entry := range profileEntries {
		if entry == "" {
			continue
		}
		if entry == AnyIPSentinel {
			m.any = true
			continue
		}
		if strings.Contains(entry, "/") {
			if p, ok := parsePrefix(entry); ok {
				m.prefixes = append(m.prefixes, p)
			}
			continue
		}
		if a, ok := parseAddr(entry); ok {
			m.literals = append(m.literals, a)
		}
	}
	return m
}

func (m *IPMatcher) Match(observedIP string) bool {
	if observedIP == "" {
		return false
	}
	observed, ok := parseAddr(observedIP)
	if !ok {
		return false
	}
	if m.any {
		return true
	}
	for _, lit := range m.literals {
		if lit == observed {
			return true
		}
	}
	for _, p := range m.prefixes {
		if p.Contains(observed) {
			return true
		}
	}
	return false
}

func MatchIP(profileEntries []string, observedIP string) bool {
	if observedIP == "" || len(profileEntries) == 0 {
		return false
	}
	return CompileIP(profileEntries).Match(observedIP)
}
