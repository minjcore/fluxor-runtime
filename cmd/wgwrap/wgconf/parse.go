// Package wgconf converts wg-quick style INI into WireGuard userspace UAPI key=value lines.
package wgconf

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// Parsed holds UAPI blob and TUN MTU hint from [Interface].
type Parsed struct {
	UAPI string
	MTU  int
	// AddressLines are raw Address = values from [Interface] (for OS setup; not applied by wireguard-go).
	AddressLines []string
}

// ParseQuick converts a wg-quick / wg.conf file body to UAPI format for device.Device.IpcSet.
func ParseQuick(ini string) (*Parsed, error) {
	var iface map[string]string
	var peers []map[string]string
	var peer map[string]string
	var section string
	out := &Parsed{MTU: 0}

	flushPeer := func() {
		if peer != nil && len(peer) > 0 {
			peers = append(peers, peer)
		}
		peer = nil
	}

	for _, raw := range strings.Split(ini, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			switch section {
			case "interface":
				flushPeer()
				iface = make(map[string]string)
			case "peer":
				flushPeer()
				peer = make(map[string]string)
			default:
				return nil, fmt.Errorf("wgconf: unknown section [%s]", section)
			}
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("wgconf: invalid line %q", line)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch section {
		case "interface":
			if iface == nil {
				return nil, fmt.Errorf("wgconf: [Interface] must appear before keys")
			}
			iface[key] = val
		case "peer":
			if peer == nil {
				return nil, fmt.Errorf("wgconf: [Peer] must appear before peer keys")
			}
			peer[key] = val
		default:
			return nil, fmt.Errorf("wgconf: key outside section: %s", key)
		}
	}
	flushPeer()

	if iface == nil {
		return nil, fmt.Errorf("wgconf: missing [Interface] section")
	}

	var b strings.Builder
	pk, ok := iface["PrivateKey"]
	if !ok || pk == "" {
		return nil, fmt.Errorf("wgconf: [Interface] PrivateKey is required")
	}
	skHex, err := keyBase64ToHex(pk)
	if err != nil {
		return nil, fmt.Errorf("wgconf: PrivateKey: %w", err)
	}
	fmt.Fprintf(&b, "private_key=%s\n", skHex)

	if lp, ok := iface["ListenPort"]; ok && lp != "" {
		port, err := strconv.ParseUint(lp, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("wgconf: ListenPort: %w", err)
		}
		fmt.Fprintf(&b, "listen_port=%d\n", port)
	}

	if mtuStr, ok := iface["MTU"]; ok && mtuStr != "" {
		mtu, err := strconv.Atoi(mtuStr)
		if err != nil || mtu < 576 || mtu > 65535 {
			return nil, fmt.Errorf("wgconf: invalid MTU %q", mtuStr)
		}
		out.MTU = mtu
	}

	fmt.Fprintf(&b, "replace_peers=true\n\n")

	for _, p := range peers {
		if err := appendPeer(&b, p); err != nil {
			return nil, err
		}
	}

	out.UAPI = b.String()

	if addr, ok := iface["Address"]; ok && addr != "" {
		for _, p := range strings.Split(addr, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				out.AddressLines = append(out.AddressLines, p)
			}
		}
	}

	return out, nil
}

func appendPeer(b *strings.Builder, peer map[string]string) error {
	pub, ok := peer["PublicKey"]
	if !ok || pub == "" {
		return fmt.Errorf("wgconf: [Peer] PublicKey is required")
	}
	pubHex, err := keyBase64ToHex(pub)
	if err != nil {
		return fmt.Errorf("wgconf: PublicKey: %w", err)
	}
	fmt.Fprintf(b, "public_key=%s\n", pubHex)

	if ps, ok := peer["PresharedKey"]; ok && ps != "" {
		h, err := keyBase64ToHex(ps)
		if err != nil {
			return fmt.Errorf("wgconf: PresharedKey: %w", err)
		}
		fmt.Fprintf(b, "preshared_key=%s\n", h)
	}

	if ep, ok := peer["Endpoint"]; ok && ep != "" {
		fmt.Fprintf(b, "endpoint=%s\n", ep)
	}

	if err := writeAllowedIPs(b, peer["AllowedIPs"]); err != nil {
		return err
	}

	if ka, ok := peer["PersistentKeepalive"]; ok && ka != "" {
		secs, err := strconv.ParseUint(ka, 10, 16)
		if err != nil {
			return fmt.Errorf("wgconf: PersistentKeepalive: %w", err)
		}
		if secs > 0 {
			fmt.Fprintf(b, "persistent_keepalive_interval=%d\n", secs)
		}
	}

	b.WriteByte('\n')
	return nil
}

func writeAllowedIPs(b *strings.Builder, raw string) error {
	if raw == "" {
		return fmt.Errorf("wgconf: [Peer] AllowedIPs is required (use 0.0.0.0/0 for full tunnel)")
	}
	fmt.Fprintf(b, "replace_allowed_ips=true\n")
	for _, cidr := range strings.Split(raw, ",") {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		fmt.Fprintf(b, "allowed_ip=%s\n", cidr)
	}
	return nil
}

func keyBase64ToHex(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(b64)
		if err != nil {
			return "", err
		}
	}
	if len(raw) != 32 {
		return "", fmt.Errorf("expected 32-byte key, got %d", len(raw))
	}
	return hex.EncodeToString(raw), nil
}
