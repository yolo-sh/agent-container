package network

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/yolo-sh/agent-container/constants"
)

type localhostListenerID string

type localhostListener struct {
	listeningPort uint64
	listeningAddr string
}

type localhostListeners map[localhostListenerID]localhostListener

type localhostProxy struct {
	listeningPort uint64
	targetAddr    string
	doneChan      chan struct{}
}

var localhostProxies = map[localhostListenerID]localhostProxy{}

func ReconcileLocalhostProxiesState() error {
	tcpConns, err := getOpenedTCPConns()

	if err != nil {
		return err
	}

	listeners := localhostListeners{}

	for _, conn := range tcpConns {
		if conn.St != uint64(tcpConnStatusListening) {
			continue
		}

		if !conn.LocalAddr.IsLoopback() {
			continue
		}

		listeningAddr := conn.LocalAddr.String()

		if conn.LocalAddr.To4() == nil { // IPv6
			listeningAddr = "[" + listeningAddr + "]"
		}

		listenerAddrAndPort := fmt.Sprintf(
			"%s:%d",
			listeningAddr,
			conn.LocalPort,
		)

		listeners[localhostListenerID(listenerAddrAndPort)] = localhostListener{
			listeningAddr: listeningAddr,
			listeningPort: conn.LocalPort,
		}
	}

	reconcileLocalhostProxiesState(listeners)

	return nil
}

func reconcileLocalhostProxiesState(listeners localhostListeners) {
	for listenerID, proxy := range localhostProxies {
		if _, listenerExists := listeners[listenerID]; listenerExists {
			continue
		}

		close(proxy.doneChan)
		delete(localhostProxies, listenerID)
	}

	for listenerID, listener := range listeners {
		if _, proxyExists := localhostProxies[listenerID]; proxyExists {
			continue
		}

		proxy := localhostProxy{
			listeningPort: listener.listeningPort,
			targetAddr:    listener.listeningAddr,
			doneChan:      make(chan struct{}),
		}

		netProxy, err := startLocalhostProxy(proxy)

		if err != nil {
			log.Printf(
				"error when starting proxy for %s:%d: %v",
				listener.listeningAddr,
				listener.listeningPort,
				err,
			)

			continue
		}

		localhostProxies[listenerID] = proxy

		go handleLocalhostProxyConn(
			netProxy,
			proxy,
		)
	}
}

func startLocalhostProxy(proxy localhostProxy) (net.Listener, error) {
	return net.Listen(
		"tcp",
		fmt.Sprintf(
			"%s:%d",
			constants.DockerContainerIPAddress,
			proxy.listeningPort,
		),
	)
}

func handleLocalhostProxyConn(netProxy net.Listener, proxy localhostProxy) {
	go func() {
		<-proxy.doneChan

		if err := netProxy.Close(); err != nil {
			log.Printf(
				"error when closing proxy for %s:%d: %v",
				proxy.targetAddr,
				proxy.listeningPort,
				err,
			)
		}
	}()

	go func() {
		for {
			proxyConn, err := netProxy.Accept()

			if err != nil {
				select {
				case <-proxy.doneChan:
					return
				default:
					log.Printf(
						"error when accepting connection on proxy for %s:%d: %v",
						proxy.targetAddr,
						proxy.listeningPort,
						err,
					)

					continue
				}
			}

			localConn, err := connectToLocalhostAddr(proxy)

			if err != nil {
				log.Printf(
					"error when connecting to %s:%d: %v",
					proxy.targetAddr,
					proxy.listeningPort,
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

			go forwardProxyConnToLocalhost(
				proxyConn,
				localConn,
			)
		}
	}()
}

func connectToLocalhostAddr(proxy localhostProxy) (net.Conn, error) {
	return net.Dial(
		"tcp",
		fmt.Sprintf(
			"%s:%d",
			proxy.targetAddr,
			proxy.listeningPort,
		),
	)
}

func forwardProxyConnToLocalhost(
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
