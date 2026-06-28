package actions

import (
	"net"
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	iphlpapi         = windows.NewLazySystemDLL("iphlpapi.dll")
	getNetworkParams = iphlpapi.NewProc("GetNetworkParams")
	getAdaptersInfo  = iphlpapi.NewProc("GetAdaptersInfo")
)

type NetworkInfo struct {
	Hostname       string   `json:"hostname"`
	IPs            []string `json:"ips"`
	DNSServers     []string `json:"dns_servers,omitempty"`
	DefaultGateway string   `json:"default_gateway,omitempty"`
}

type FIXED_INFO struct {
	HostName         [132]byte
	DomainName       [132]byte
	CurrentDNSServer *byte
	DNSServerList    IP_ADDR_STRING
	NodeType         uint32
	ScopeId          [260]byte
	EnableRouting    uint32
	EnableProxy      uint32
	EnableDNS        uint32
}

type IP_ADDR_STRING struct {
	Next      *IP_ADDR_STRING
	IPAddress [16]byte
	IPMask    [16]byte
	Context   uint32
	_         [4]byte
}

type IP_ADAPTER_INFO struct {
	Next                *IP_ADAPTER_INFO
	ComboIndex          uint32
	AdapterName         [260]byte
	Description         [260]byte
	AddressLength       uint32
	Address             [8]byte
	Index               uint32
	Type                uint32
	DhcpEnabled         uint32
	_                   [4]byte
	CurrentIPAddress    *byte
	IPAddressList       IP_ADDR_STRING
	GatewayList         IP_ADDR_STRING
	DhcpServer          IP_ADDR_STRING
	HaveWins            uint32
	PrimaryWinsServer   IP_ADDR_STRING
	SecondaryWinsServer IP_ADDR_STRING
	LeaseObtained       uint32
	LeaseExpires        uint32
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

	dns := getDNSServersWin32()
	gw := getDefaultGatewayWin32()

	return &NetworkInfo{
		Hostname:       hn,
		IPs:            ips,
		DNSServers:    dns,
		DefaultGateway: gw,
	}, nil
}

func getDNSServersWin32() []string {
	bufSize := uint32(0)
	getNetworkParams.Call(uintptr(0), uintptr(unsafe.Pointer(&bufSize)))
	if bufSize == 0 {
		return nil
	}
	buf := make([]byte, bufSize)
	ret, _, _ := getNetworkParams.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bufSize)))
	if ret != 0 {
		return nil
	}
	fi := (*FIXED_INFO)(unsafe.Pointer(&buf[0]))
	if fi.DNSServerList.IPAddress[0] == 0 {
		return nil
	}
	var servers []string
	for ds := &fi.DNSServerList; ds != nil; ds = ds.Next {
		ip := string(ds.IPAddress[:])
		if idx := strings.IndexByte(ip, 0); idx >= 0 {
			ip = ip[:idx]
		}
		if ip != "" {
			servers = append(servers, ip)
		}
	}
	return servers
}

func getDefaultGatewayWin32() string {
	bufSize := uint32(0)
	getAdaptersInfo.Call(uintptr(0), uintptr(unsafe.Pointer(&bufSize)))
	if bufSize == 0 {
		return ""
	}
	buf := make([]byte, bufSize)
	ret, _, _ := getAdaptersInfo.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bufSize)))
	if ret != 0 {
		return ""
	}
	ai := (*IP_ADAPTER_INFO)(unsafe.Pointer(&buf[0]))
	for ai != nil {
		if ai.GatewayList.IPAddress[0] != 0 {
			gw := string(ai.GatewayList.IPAddress[:])
			if idx := strings.IndexByte(gw, 0); idx >= 0 {
				gw = gw[:idx]
			}
			if gw != "" && gw != "0.0.0.0" {
				return gw
			}
		}
		ai = ai.Next
	}
	return ""
}

func PingHost(host string) (bool, error) {
	cmd := exec.Command("ping", "-n", "1", "-w", "3000", host)
	err := cmd.Run()
	return err == nil, nil
}
