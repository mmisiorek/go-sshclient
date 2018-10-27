package sshwrapper

import (
	"errors"
	"fmt"
	"github.com/eugenmayer/go-sshclient/scpwrapper"
	"golang.org/x/crypto/ssh"
	"net"
	"time"
)

// connect the ssh client and create a session, ready to go with commands ro scp
func (sshApi *SshApi) ConnectAndSession() (err error) {
	if client, err := sshApi.Connect(); err != nil {
		return err
	} else {
		sshApi.Client = client
	}

	return sshApi.SessionDefault()
}

// creates a default session with usual parameters
func (sshApi *SshApi) SessionDefault() (err error) {
	if session, err := sshApi.Client.NewSession(); err != nil {
		return err
	} else {
		sshApi.Session = session
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := sshApi.Session.RequestPty("xterm", 80, 40, modes); err != nil {
		sshApi.Session.Close()
		return err
	}

	sshApi.Session.Stdout = &sshApi.StdOut
	sshApi.Session.Stderr = &sshApi.StdErr
	return nil
}

// connect the ssh client - use ConnectAndSession if you have no reason to create
// the session manually why ever
// we do support proper timeouts here, thats why it looks a little more complicated then the usual ssh connect
func (sshApi *SshApi) Connect() (*ssh.Client, error) {
	var addr = fmt.Sprintf("%s:%d", sshApi.Host, sshApi.Port)
	conn, err := net.DialTimeout("tcp", addr, sshApi.SshConfig.Timeout)
	if err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, sshApi.SshConfig)
	if err != nil {
		return nil, err
	}

	err = conn.SetReadDeadline(time.Time{})
	return ssh.NewClient(c, chans, reqs), err
}

// get the stdout from your last command
func (sshApi *SshApi) GetStdOut() string {
	return sshApi.StdOut.String()
}

// get the stderr from your last command
func (sshApi *SshApi) GetStdErr() string {
	return sshApi.StdErr.String()
}

// run a ssh command. Auto-creates session if you did yet not connect
// just wrapping ssh.Session.Run with connect / run and then disconnect
func (sshApi *SshApi) Run(cmd string) (stdout string, stderr string, err error) {
	if sshApi.Session == nil {
		if err = sshApi.ConnectAndSession(); err != nil {
			return "","", err
		}

		// this can actually still happen. TODO: document why
		if sshApi.Session == nil {
			return "","", errors.New("could not start ssh session")
		}
	}

	err = sshApi.Session.Run(cmd)
	sshApi.Session.Close()
	return  sshApi.GetStdOut(),sshApi.GetStdErr(), err
}

// scp a local file to a remote host
func (sshApi *SshApi) CopyToRemote(source string, dest string) (err error) {
	sshApi.ConnectAndSession()
	err = scpwrapper.CopyToRemote(source, dest, sshApi.Session)
	sshApi.Session.Close()
	return err
}

// scp a file from a remote host
func (sshApi *SshApi) CopyFromRemote(source string, dest string) (err error) {
	sshApi.ConnectAndSession()
	err = scpwrapper.CopyFromRemote(source, dest, sshApi.Session)
	sshApi.Session.Close()
	return err
}