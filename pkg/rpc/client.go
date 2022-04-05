/*
Copyright © 2022 Platform9, Inc.
*/
package rpc

import (
	"net/rpc"
	"go.uber.org/zap"
)

func Connect(port string) (*rpc.Client, error) {
	zap.S().Infof("starting test client connecting localhost:8988")
	client, err := rpc.DialHTTP("tcp", "localhost:8988")
	if err != nil {
		zap.S().Infof("dialing: %v", err)
	}
	return client, err
}
