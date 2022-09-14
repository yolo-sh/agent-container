package main

import (
	"log"
	"os"
	"time"

	"github.com/yolo-sh/agent-container/constants"
	"github.com/yolo-sh/agent-container/internal/grpcserver"
	"github.com/yolo-sh/agent-container/internal/network"
)

func main() {
	// Prevent "bind: address already in use" error
	err := ensureOldGRPCServerSocketRemoved(constants.GRPCServerAddr)

	if err != nil {
		log.Fatalf("%v", err)
	}

	go func() {
		log.Printf(
			"Polling proxies state...",
		)

		for {
			err := network.ReconcileProxiesState()

			if err != nil {
				log.Fatalf("%v", err)
			}

			time.Sleep(60 * time.Millisecond)
		}
	}()

	log.Printf(
		"GRPC server listening at %s",
		constants.GRPCServerUri,
	)

	err = grpcserver.ListenAndServe(
		constants.GRPCServerAddrProtocol,
		constants.GRPCServerAddr,
	)

	if err != nil {
		log.Fatalf("%v", err)
	}
}

func ensureOldGRPCServerSocketRemoved(socketPath string) error {
	return os.RemoveAll(socketPath)
}
