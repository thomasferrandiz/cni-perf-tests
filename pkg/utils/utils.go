package utils

import (
	"os"
	"os/exec"

	"golang.org/x/crypto/ssh"
)

type SshConfig struct {
	User     string
	Hostname string
	KeyPath  string
}

// RunCommand execute a command on the host
func RunCommand(cmd string) (string, error) {
	c := exec.Command("bash", "-c", cmd)
	out, err := c.CombinedOutput()
	return string(out), err
}

func RunCommandRemotely(conf SshConfig, cmd string) (string, error) {
	config := &ssh.ClientConfig{
		User: conf.User,
		Auth: []ssh.AuthMethod{
			publicKey(conf.KeyPath),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", conf.Hostname, config)
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
