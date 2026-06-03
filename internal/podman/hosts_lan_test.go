package podman

import (
	"net"
	"testing"
)

func cidr(t *testing.T, s string) net.Addr {
	t.Helper()
	ip, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		t.Fatal(err)
	}
	ipnet.IP = ip
	return ipnet
}

func TestPickLANIP_SkipsLoopbackAndDown(t *testing.T) {
	ifaces := []lanIface{
		{up: true, loopback: true, addrs: []net.Addr{cidr(t, "127.0.0.1/8")}},
		{up: false, loopback: false, addrs: []net.Addr{cidr(t, "10.0.0.5/24")}}, // down
		{up: true, loopback: false, addrs: []net.Addr{cidr(t, "172.20.0.3/20")}}, // eth0 WSL
	}
	if got := pickLANIP(ifaces); got != "172.20.0.3" {
		t.Errorf("pickLANIP = %q, want 172.20.0.3", got)
	}
}

func TestPickLANIP_LoopbackOnlyReturnsEmpty(t *testing.T) {
	ifaces := []lanIface{
		{up: true, loopback: true, addrs: []net.Addr{cidr(t, "127.0.0.1/8")}},
	}
	if got := pickLANIP(ifaces); got != "" {
		t.Errorf("pickLANIP = %q, want vazio", got)
	}
}

func TestPickLANIP_NoInterfacesReturnsEmpty(t *testing.T) {
	if got := pickLANIP(nil); got != "" {
		t.Errorf("pickLANIP = %q, want vazio", got)
	}
}

func TestHostCandidates_OrderAndDedup(t *testing.T) {
	got := hostCandidates("172.20.0.1", "172.20.0.1") // getent == lan: dedup
	want := []string{"172.20.0.1", "10.0.2.2"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("hostCandidates = %v, want %v", got, want)
	}
}
