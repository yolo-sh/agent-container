package network

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/yolo-sh/agent-container/constants"
)

type proxyChan chan struct{}

var proxies = map[tcpPort]proxyChan{}

func ReconcileProxiesState() error {
	tcpConns, err := getOpenedTCPConns()

	if err != nil {
		return err
	}

	localPorts := map[tcpPort]bool{}

	for _, conn := range tcpConns {
		if conn.St != uint64(tcpConnStatusListening) {
			continue
		}

		if conn.LocalAddr.String() != tcpLocalhostIpv4Addr {
			continue
		}

		localPorts[tcpPort(conn.LocalPort)] = true
	}

	reconcileProxiesState(localPorts)

	return nil
}

func reconcileProxiesState(localPorts map[tcpPort]bool) {
	for proxyPort, proxyChan := range proxies {
		if localPorts[proxyPort] {
			continue
		}

		close(proxyChan)
		delete(proxies, proxyPort)
	}

	for localPort := range localPorts {
		if _, ok := proxies[localPort]; ok {
			continue
		}

		proxy, err := startProxy(tcpPort(localPort))

		if err != nil {
			log.Printf(
				"error when starting proxy for port %d: %v",
				localPort,
				err,
			)

			continue
		}

		proxies[localPort] = make(proxyChan)

		go handleProxyConn(
			proxy,
			proxies[localPort],
			localPort,
		)
	}
}

func startProxy(localPort tcpPort) (net.Listener, error) {
	return net.Listen(
		"tcp",
		fmt.Sprintf(
			"%s:%d",
			constants.DockerContainerIPAddress,
			localPort,
		),
	)
}

func handleProxyConn(
	proxy net.Listener,
	closeProxyChan proxyChan,
	localPort tcpPort,
) {

	go func() {
		<-closeProxyChan

		if err := proxy.Close(); err != nil {
			log.Printf(
				"error when closing proxy for port %d: %v",
				localPort,
				err,
			)
		}
	}()

	go func() {
		for {
			proxyConn, err := proxy.Accept()

			if err != nil {
				select {
				case <-closeProxyChan:
					return
				default:
					log.Printf(
						"error when accepting connection on proxy for port %d: %v",
						localPort,
						err,
					)

					continue
				}
			}

			localConn, err := connectToLocalAddr(localPort)

			if err != nil {
				log.Printf(
					"error when connecting to %s:%d: %v",
					tcpLocalhostIpv4Addr,
					localPort,
					err,
				)

				if err := proxyConn.Close(); err != nil {
					log.Printf(
						"error when closing proxy connection: %v",
						err,
					)
				}

				continue
			}

			go forwardProxyToLocalConn(
				proxyConn,
				localConn,
			)
		}
	}()
}

func connectToLocalAddr(localPort tcpPort) (net.Conn, error) {
	return net.Dial(
		"tcp",
		fmt.Sprintf(
			"%s:%d",
			tcpLocalhostIpv4Addr,
			localPort,
		),
	)
}

func forwardProxyToLocalConn(
	proxyConn net.Conn,
	localConn net.Conn,
) error {

	defer func() {
		proxyConn.Close()
		localConn.Close()
	}()

	proxyConnChan := make(chan error, 1)
	localConnChan := make(chan error, 1)

	// Forward proxy -> local
	go func() {
		_, err := io.Copy(proxyConn, localConn)
		proxyConnChan <- err
	}()

	// Forward local -> proxy
	go func() {
		_, err := io.Copy(localConn, proxyConn)
		localConnChan <- err
	}()

	select {
	case proxyConnErr := <-proxyConnChan:
		if proxyConnErr != nil {
			fmt.Printf(
				"error during proxy connection forwarding: %v",
				proxyConnErr,
			)
		}
		return proxyConnErr
	case localConnErr := <-localConnChan:
		if localConnErr != nil {
			fmt.Printf(
				"error during local connection forwarding: %v",
				localConnErr,
			)
		}
		return localConnErr
	}
}
