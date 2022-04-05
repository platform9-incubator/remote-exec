/*
Copyright © 2020 Platform9, Inc.
*/
package cmd

import (
	"fmt"

	"github.com/platform9-incubator/remote-exec/pkg/rpc"
	"go.uber.org/zap"

	"github.com/spf13/cobra"
)

type clientCmdOpts struct {
	server  string
	user    string
	sshKey  string
	sshPort string
}

var clientOpts = clientCmdOpts{}
var mainCmd = &cobra.Command{
	Use:   "client",
	Short: "start client",
	Long:  `start client`,
	RunE: func(cmd *cobra.Command, args []string) error {
		zap.S().Infof("Starting main server, connecting to %s", clientOpts.server)
		remoteRPCServer, err := rpc.SshRemoteRPC(clientOpts.server, clientOpts.user, clientOpts.sshKey)
		if err != nil {
			return fmt.Errorf("error cloning through tunnel: %s", err)
		}
		remoteRPCServer.Start()
		zap.S().Infof("starting test client connecting %s", remoteRPCServer.GetAddress())
		client, err := rpc.Connect(remoteRPCServer.GetAddress())
		if err != nil {
			zap.S().Infof("dialing: %v", err)
			return err
		}

		testClient(client)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(mainCmd)
	mainCmd.Flags().StringVar(&clientOpts.server, "server", "", "The server to SSH to")
	mainCmd.Flags().StringVar(&clientOpts.user, "user", "", "The user to SSH as")
	mainCmd.Flags().StringVar(&clientOpts.sshKey, "sshKey", "", "The SSH key to use")
}
