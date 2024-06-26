package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"golang.org/x/crypto/ssh"
	log "k8s.io/klog/v2"
)

type SshConfig struct {
	User     string
	Hostname string
	Port     int
	KeyPath  string
}

// RunCommand execute a command on the host
func RunCommand(cmd string) (string, error) {
	c := exec.Command("bash", "-c", cmd)
	out, err := c.CombinedOutput()
	return string(out), err
}

func CreateSShClient(conf SshConfig) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: conf.User,
		Auth: []ssh.AuthMethod{
			publicKey(conf.KeyPath),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", conf.Hostname+":"+fmt.Sprint(conf.Port), config)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func RunCommandRemotely(conf SshConfig, cmd string) (string, error) {
	conn, err := CreateSShClient(conf)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	res, err := session.CombinedOutput(cmd) // eg., /usr/bin/whoami
	if err != nil {
		return "", err
	}
	return string(res), err
}

func RunCommandRemotelyWithTimeout(ctx context.Context, timeout time.Duration, conf SshConfig, cmd string) error {
	conn, err := CreateSShClient(conf)
	if err != nil {
		return err
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	toctx, cancelFunc := context.WithTimeout(ctx, timeout)
	defer cancelFunc()
	log.Infof("Starting command [ %s ] on remote host [ %s ] with timeout [ %v ]", cmd, conf.Hostname, timeout)
	if err := session.Start(cmd); err != nil {
		return err
	}

	//wait for timeout or parent context to be over since the command is assumed to never end
	<-toctx.Done()
	log.Info("Context over sending SIGTERM to remote process")
	session.Signal(ssh.SIGTERM)
	session.Close()
	return toctx.Err()

}

func publicKey(path string) ssh.AuthMethod {
	key, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		panic(err)
	}
	return ssh.PublicKeys(signer)
}
