package actions

import (
	"net"
	"os/exec"
	"strings"
	"os"
)

type NetworkInfo struct {
	Hostname string   `json:"hostname"`
	IPs      []string `json:"ips"`
	DNSServers []string `json:"dns_servers,omitempty"`
	DefaultGateway string `json:"default_gateway,omitempty"`
}

func GetNetworkInfo() (*NetworkInfo, error) {
	hn, _ := os.Hostname()
	if hn == "" {
		hn = "unknown"
	}

	var ips []string
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			ips = append(ips, ipnet.IP.String())
		}
	}

	dns, _ := getDNSServers()
	gw, _ := getDefaultGateway()

	return &NetworkInfo{
		Hostname:      hn,
		IPs:           ips,
		DNSServers:    dns,
		DefaultGateway: gw,
	}, nil
}

func getDNSServers() ([]string, error) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		`(Get-DnsClientServerAddress -AddressFamily IPv4).ServerAddresses -join ','`).Output()
	if err != nil {
		return nil, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil, nil
	}
	return strings.Split(s, ","), nil
}

func getDefaultGateway() (string, error) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		`(Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Select-Object -First 1).NextHop`).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func PingHost(host string) (bool, error) {
	cmd := exec.Command("ping", "-n", "1", "-w", "3000", host)
	err := cmd.Run()
	return err == nil, nil
}
