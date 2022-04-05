/*
Copyright © 2020 Platform9, Inc.
*/
package cmd

import (
	"github.com/platform9-incubator/remote-exec/pkg/rpc"
	"github.com/spf13/cobra"
)

type minionCmdOpts struct {
	listenPort string
}

var minionOpts = minionCmdOpts{}
var minionCmd = &cobra.Command{
	Use:   "minion",
	Short: "start minion on a server",
	Long:  `start minion on a server`,
	RunE: func(cmd *cobra.Command, args []string) error {
		arith := new(Arith)
		rpc.Register(arith)
		rpc.Start(minionOpts.listenPort)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(minionCmd)
	minionCmd.Flags().StringVar(&minionOpts.listenPort, "port", "8989", "The port to listen to")
}
