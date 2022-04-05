package rpc


import (
	"fmt"
	"os"
	"path"
	"github.com/platform9-incubator/remote-exec/pkg/tunnel"
	"go.uber.org/zap"
)

type Conn interface {
	GetLocalAddress() string
	Open() error
	UploadFile(local string, dest string, mode os.FileMode) error
	RunCommand(cmd string) ([]byte, []byte, error)
}

type RemoteRPCServer struct {
	binPath string
	binName string
	conn    Conn
	remoteTmp string
}

func SshRemoteRPC(server, user, sshKey string) (*RemoteRPCServer, error) {
	// replace hardcoded ports with dynamic ones or use linux socket
	tun := tunnel.New("tcp4", server,
						"localhost:8988", 
						"localhost:8989", 
						user, 
						sshKey)
	return NewRemoteRPC(tun)
}

func NewRemoteRPC(conn Conn) (*RemoteRPCServer, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	baseName := path.Base(execPath)
	zap.S().Info("cloner path: ", execPath, baseName)

	return &RemoteRPCServer{binPath: execPath, conn: conn, binName: baseName, remoteTmp: "/tmp"}, nil
}

// GetAddress returns the address to connect o
func (rc *RemoteRPCServer) GetAddress() string {
	return rc.conn.GetLocalAddress()
}

func (rc *RemoteRPCServer) Start() error {
	err := rc.conn.Open()
	if err != nil {
		return fmt.Errorf("error opening tunnel: %s", err)
	}
	zap.S().Infof("connected to %s", clientOpts.server)

	// use the dfault port
	remoteFile := fmt.Sprintf("%s/%s", rc.remoteTmp, rc.binName)
	zap.S().Infof("copying binary %s to remote %s", rc.binPath, remoteFile)
	err = rc.conn.UploadFile(rc.binPath, remoteFile, os.FileMode(0700))
	if err != nil {
		return fmt.Errorf("error uploading file: %s", err)
	}
	remoteCmd := fmt.Sprintf("%s %s", remoteFile, "minion")
	zap.S().Infof("running remote command in another goroutine %s", remoteCmd)
	stdout, stderr, err := rc.conn.RunCommand(remoteCmd)
	zap.S().Infof("Stdout %s, StdErr %s", stdout, stderr)
	if err != nil {
		return fmt.Errorf("error executing command %s: %s", remoteCmd, err)
	}
	zap.S().Infof("command execution done %s", remoteCmd)
	return nil
}

func (rc *RemoteRPCServer) Stop() error {
	return nil
}