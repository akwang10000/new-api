package common

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsPrivateIPRejectsSpecialUseRanges(t *testing.T) {
	privateIPs := []string{
		"0.0.0.0",
		"10.1.2.3",
		"100.64.0.1",
		"127.0.0.1",
		"169.254.169.254",
		"172.16.0.1",
		"192.0.0.1",
		"192.0.2.1",
		"192.168.1.1",
		"198.18.0.1",
		"198.51.100.1",
		"203.0.113.1",
		"224.0.0.1",
		"240.0.0.1",
		"255.255.255.255",
		"::1",
		"::ffff:192.0.2.1",
		"64:ff9b::1",
		"100::1",
		"2001:db8::1",
		"fc00::1",
		"fe80::1",
		"ff00::1",
	}

	for _, ipText := range privateIPs {
		t.Run(ipText, func(t *testing.T) {
			require.True(t, isPrivateIP(net.ParseIP(ipText)))
		})
	}
}

func TestIsPrivateIPAllowsPublicRanges(t *testing.T) {
	publicIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"2606:4700:4700::1111",
	}

	for _, ipText := range publicIPs {
		t.Run(ipText, func(t *testing.T) {
			require.False(t, isPrivateIP(net.ParseIP(ipText)))
		})
	}
}

func TestValidateURLRejectsScopedIPv6Literal(t *testing.T) {
	protection := &SSRFProtection{
		AllowPrivateIp:         false,
		DomainFilterMode:       false,
		IpFilterMode:           false,
		ApplyIPFilterForDomain: false,
	}

	require.Error(t, protection.ValidateURL("http://[fe80::1%25eth0]/"))
}

func TestValidateURLRejectsIPv4MappedIPv6Literal(t *testing.T) {
	protection := &SSRFProtection{
		AllowPrivateIp:         false,
		DomainFilterMode:       false,
		IpFilterMode:           false,
		ApplyIPFilterForDomain: false,
	}

	require.Error(t, protection.ValidateURL("http://[::ffff:8.8.8.8]/"))
	require.NoError(t, protection.ValidateURL("http://8.8.8.8/"))
}
