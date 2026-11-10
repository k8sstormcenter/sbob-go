package networkmatch

import (
	"fmt"
	"strings"
)

func ValidateIPEntry(entry string) error {
	if entry == "" {
		return fmt.Errorf("empty IP entry")
	}
	if entry == AnyIPSentinel {
		return nil
	}
	if strings.Contains(entry, "/") {
		if _, ok := parsePrefix(entry); !ok {
			return fmt.Errorf("malformed CIDR %q", entry)
		}
		return nil
	}
	if _, ok := parseAddr(entry); !ok {
		return fmt.Errorf("malformed IP %q (not a literal, not a CIDR)", entry)
	}
	return nil
}

func ValidateDNSEntry(entry string) error {
	if entry == "" {
		return fmt.Errorf("empty DNS entry")
	}
	canon := strings.TrimSuffix(entry, ".")
	if canon == "" {
		return fmt.Errorf("DNS entry %q is just a trailing dot", entry)
	}
	if canon == DNSWildcardLabel {
		return nil
	}
	labels := strings.Split(canon, ".")
	for i, l := range labels {
		switch {
		case l == "":
			return fmt.Errorf("DNS entry %q has empty label at position %d", entry, i)
		case l == "**":
			return fmt.Errorf("DNS entry %q contains reserved recursive wildcard %q", entry, "**")
		case l == DNSWildcardLabel && i != 0 && i != len(labels)-1:
			return fmt.Errorf(
				"DNS entry %q: bare %q is only allowed as the first label (RFC 4592) "+
					"or last label (project extension); use %q for mid positions",
				entry, DNSWildcardLabel, DNSDynamicLabel)
		}
	}
	return nil
}
