/**
* Example Client
 */

package cmd

import (
	"fmt"
	"net/rpc"
	"reflect"
	"runtime"

	"go.uber.org/zap"
)

func MagicCall(client *rpc.Client, m interface{}, args interface{}, reply interface{}) error {

	methodName := runtime.FuncForPC(reflect.ValueOf(m).Pointer()).Name()
	err := client.Call(methodName, args, reply)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}

func testClient(client *rpc.Client) {
	// Synchronous call
	args := &Args{7, 8}
	var reply int
	zap.S().Infof("calling multiply")

	err := client.Call("Arith.Multiply", args, &reply)
	if err != nil {
		zap.S().Infof("arith error: %s", err)
	}
	zap.S().Infof("Arith: %d*%d=%d", args.A, args.B, reply)

}
