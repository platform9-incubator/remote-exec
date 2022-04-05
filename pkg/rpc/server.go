/*
Copyright © 2022 Platform9, Inc.
*/
package rpc

import (
	"fmt"
	"net/http"
	"net/rpc"
	"os"

	"go.uber.org/zap"
)

func Register(data interface{}) {
	rpc.Register(data)
}

func Start(port string) error {
	zap.S().Infof("Starting RPC Server on port %s", port)
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error getting executable path: %s", err)
	}
	// delete itself to avoid leakage of the binary lying on remote hosts
	zap.S().Infof("deleting itself %s", execPath)
	err = os.Remove(execPath)
	if err != nil {
		zap.S().Warnf("error removing executable %s: %s, ignoring the error", execPath, err)
	}

	rpc.HandleHTTP()
	zap.S().Infof("starting http server on port %s", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		return fmt.Errorf("error starting rpc server: %s", err)
	}
	zap.S().Infof("rpc server exiting %s", port)
	return nil
}
