package network

import (
	"fmt"

	"github.com/prometheus/procfs"
)

// Ref: https://unix.stackexchange.com/a/470527
type tcpConnStatus uint64

const (
	tcpConnStatusEstablished tcpConnStatus = 1
	tcpConnStatusListening   tcpConnStatus = 10
)

type tcpPort uint64

const (
	tcpLocalhostIpv4Addr = "127.0.0.1"
)

func getOpenedTCPConns() (procfs.NetTCP, error) {
	proc, err := procfs.NewFS("/proc")
	if err != nil {
		return nil, fmt.Errorf("could not read /proc: %s", err)
	}

	tcpIPv4, err := proc.NetTCP()
	if err != nil {
		return nil, fmt.Errorf("could not read /proc/net/tcp: %s", err)
	}

	tcpIPv6, err := proc.NetTCP6()
	if err != nil {
		return nil, fmt.Errorf("could not read /proc/net/tcp6: %s", err)
	}

	return append(tcpIPv4, tcpIPv6...), nil
}
