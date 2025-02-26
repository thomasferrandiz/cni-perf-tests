package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"golang.org/x/crypto/ssh"
	yamlv2 "gopkg.in/yaml.v2"
	log "k8s.io/klog/v2"
)

type Servers struct {
	MasterNode  SshConfig
	WorkerNode1 SshConfig
	WorkerNode2 SshConfig
}
type SshConfig struct {
	User       string
	SshIpAddr  string // ip to use for ssh
	TestIpAddr string // ip to use for the tests (i.e iperf3)
	Port       int
	KeyPath    string
	Nodename   string // name as k8s node
}

func ParseConfig(conf []byte) (*Servers, error) {
	testConf := new(Servers)
	err := yamlv2.Unmarshal(conf, testConf)
	if err != nil {
		return nil, err
	}

	return testConf, nil
}

func ReadConfigFile(filename string) (*Servers, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	testConf, err := ParseConfig(content)
	if err != nil {
		return nil, err
	}
	return testConf, nil
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

	conn, err := ssh.Dial("tcp", conf.SshIpAddr+":"+fmt.Sprint(conf.Port), config)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func RunCommandRemotely(conf SshConfig, cmd string) ([]byte, error) {
	conn, err := CreateSShClient(conf)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	session.Setenv("KUBECONFIG", "/etc/rancher/rke2/rke2.yaml")
	log.Infof("Running command [ %s ] on host [ %s ]...", cmd, conf.SshIpAddr)
	res, err := session.CombinedOutput(cmd) // eg., /usr/bin/whoami
	if err != nil {
		log.Errorf("command result:\n %s", res)
		return nil, err
	}
	return res, err
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
	log.Infof("Starting command [ %s ] on remote host [ %s ] with timeout [ %v ]", cmd, conf.SshIpAddr, timeout)
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
