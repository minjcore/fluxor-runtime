package wgconf

import (
	"strings"
	"testing"
)

func TestParseQuick_minimal(t *testing.T) {
	// Keys are zero keys — invalid for real WG but valid base64 length for parser smoke test.
	const ini = `
[Interface]
PrivateKey = AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEA=
ListenPort = 51820
MTU = 1380

[Peer]
PublicKey = YAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
AllowedIPs = 10.66.66.0/24
Endpoint = 198.51.100.1:51820
PersistentKeepalive = 25
`
	p, err := ParseQuick(ini)
	if err != nil {
		t.Fatal(err)
	}
	if p.MTU != 1380 {
		t.Fatalf("MTU: got %d", p.MTU)
	}
	if !strings.Contains(p.UAPI, "private_key=") || !strings.Contains(p.UAPI, "public_key=") {
		t.Fatalf("UAPI: %q", p.UAPI)
	}
	if !strings.Contains(p.UAPI, "allowed_ip=10.66.66.0/24") {
		t.Fatalf("allowed_ip missing: %q", p.UAPI)
	}
}
