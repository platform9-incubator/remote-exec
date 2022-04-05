/*
	Copyright (c) 2021 Platform9, Inc.
	All rights reserved.
*/

package tunnel

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"github.com/pkg/sftp"
)

type SSHConn struct {
	NetworkType       string
	ServerAddr        string
	LocalAddr         string
	RemoteAddr        string
	Username          string
	SSHPrivateKeyFile string
	exit              chan bool
	localListener     net.Listener
	sshClient     	  *ssh.Client
	sftpClient        *sftp.Client
}

func New(netType string, serverAddr string, localAddr string, remoteAddr string, username string, sshPrivateKeyFile string) *SSHConn {
	cfg := SSHConn{
		NetworkType:       netType,
		ServerAddr:        serverAddr,
		LocalAddr:         localAddr,
		RemoteAddr:        remoteAddr,
		Username:          username,
		SSHPrivateKeyFile: sshPrivateKeyFile,
	}
	cfg.exit = make(chan bool)

	return &cfg
}

func (sc *SSHConn) Open() error {
	privateKey, err := ioutil.ReadFile(sc.SSHPrivateKeyFile)
	if err != nil {
		return fmt.Errorf("error reading private key file %s %s", sc.SSHPrivateKeyFile, err)
	}
	authMethods := make([]ssh.AuthMethod, 1)
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return fmt.Errorf("error parsing private key: %s", err)
	}
	authMethods[0] = ssh.PublicKeys(signer)
	sshConfig := &ssh.ClientConfig{
		User: string(sc.Username),
		Auth: authMethods,
		// by default ignore host key checks
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}


	// Setup localListener (type net.Listener)
	sc.localListener, err = net.Listen(sc.NetworkType, sc.LocalAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", sc.LocalAddr, err)
	}

	// Setup sshClientConn (type *ssh.ClientConn)
	sc.sshClient, err = ssh.Dial(sc.NetworkType, sc.ServerAddr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to dial serverAddr %s: %s", sc.ServerAddr, err)
	}
	sc.sftpClient, err = sftp.NewClient(sc.sshClient)
	if err != nil {
		return fmt.Errorf("failed to create sftpClient %s: %s", sc.ServerAddr, err)
	}
	go sc.startListening(sshConfig)

	return nil
}

func (sc *SSHConn) startListening(sshConfig *ssh.ClientConfig) {

	defer sc.localListener.Close()
	defer sc.sshClient.Close()
	for {
		select {
		case <-sc.exit:
			return
		default:
			// Setup localConn (type net.Conn)
			localConn, err := sc.localListener.Accept()
			if err != nil {
				zap.S().Fatalf("failed to accept connections: %v", err)
			}
			go sc.forward(localConn, sshConfig)
		}
	}
}

func (sc *SSHConn) Close() {
	zap.S().Infof("closing tunnel")
	select {
	case sc.exit <- true:
	default:
		// HACK - the other goroutine will still be blocked on localListener.Accept()
		// sending a dummy request to return from it
		client := http.Client{
			Timeout: 1 * time.Second,
		}
		_, _ = client.Get(fmt.Sprintf("http://%s/v2/_catalog", sc.LocalAddr))
	}

}

func (sc *SSHConn) forward(localConn net.Conn, config *ssh.ClientConfig) {
	done := make(chan bool)
	// Setup sshConn (type net.Conn)
	sshConn, err := sc.sshClient.Dial(sc.NetworkType, sc.RemoteAddr)
	if err != nil {
		zap.S().Errorf("failed to dial remoteAddr %s: %s", sc.RemoteAddr, err)
		sc.sshClient.Close()
		return
	}

	// Copy localConn.Reader to sshConn.Writer
	go func() {
		_, err = io.Copy(sshConn, localConn)
		if err != nil {
			zap.S().Errorf("tunnel: local to remote copy failed: %v", err)
			done <- true
			return
		}
		done <- true
	}()

	// Copy sshConn.Reader to localConn.Writer
	go func() {
		_, err = io.Copy(localConn, sshConn)
		if err != nil {
			zap.S().Errorf("tunnel: remote to local copy failed: %v", err)
			done <- true
		}
		done <- true
	}()

	<-done
	<-done
	sshConn.Close()
	localConn.Close()
}

// RunCommand runs a command on the machine and returns stdout and stderr
// separately
func (sc *SSHConn) RunCommand(cmd string) ([]byte, []byte, error) {

	session, err := sc.sshClient.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create session: %s", err)
	}
	stdOutPipe, err := session.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to pipe stdout: %s", err)
	}
	stdErrPipe, err := session.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to pipe stderr: %s", err)
	}
	err = session.Start(cmd)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to run command: %s", err)
	}
	stdOut, err := ioutil.ReadAll(stdOutPipe)
	stdErr, err := ioutil.ReadAll(stdErrPipe)
	err = session.Wait()
	if err != nil {
		retError := err
		switch err.(type) {
		case *ssh.ExitError:
			retError = fmt.Errorf("command %s failed: %s", cmd, err)
		case *ssh.ExitMissingError:
			retError = fmt.Errorf("command %s failed (no exit status): %s", cmd, err)
		default:
			retError = fmt.Errorf("command %s failed: %s", cmd, err)
		}

		zap.L().Debug("Error ", zap.String("stdout", string(stdOut)), zap.String("stderr", string(stdErr)))

		return stdOut, stdErr, retError
	}
	return stdOut, stdErr, nil
}

// Upload writes a file to the machine
func (sc *SSHConn) UploadFile(localFile string, remoteFilePath string, mode os.FileMode) error {
	// first check if the local file exists or not
	localFp, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("unable to read localFile: %s", err)
	}
	defer localFp.Close()
	_, err = localFp.Stat()
	if err != nil {
		return fmt.Errorf("Unable to find size of the file %s", localFile)
	}

	localFileReader := bufio.NewReader(localFp)

	remoteFile, err := sc.sftpClient.Create(remoteFilePath)
	if err != nil {
		return fmt.Errorf("unable to create file: %s", err)
	}
	defer remoteFile.Close()
	// IMHO this function is misnomer, it actually writes to the remoteFile
	_, err = remoteFile.ReadFrom(localFileReader)
	if err != nil {
		// rmove the remote file since write failed and ignore the errors
		// we can't do much about it anyways.
		sc.sftpClient.Remove(remoteFilePath)
		return fmt.Errorf("write failed: %s, ", err)
	}
	err = remoteFile.Chmod(mode)
	if err != nil {
		return fmt.Errorf("chmod failed: %s", err)
	}
	return nil
}

func (sc *SSHConn) GetLocalAddress() string {
	return sc.LocalAddr
}