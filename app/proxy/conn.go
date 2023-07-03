package proxy

import (
	"errors"
	"io"
	config "mjump/app/data"
	"mjump/app/logger"
	"net"
	"sync"
	"sync/atomic"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

type ServerConnection interface {
	io.ReadWriteCloser
	SetWinSize(width, height int) error
	KeepAlive() error
}
type SSHClient struct {
	*gossh.Client
	// Cfg         *SSHClientOptions
	ProxyClient *SSHClient

	sync.Mutex

	traceSessionMap map[*gossh.Session]time.Time

	refCount int32
}

func (s *SSHClient) ReleaseSession(sess *gossh.Session) {
	atomic.AddInt32(&s.refCount, -1)
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	delete(s.traceSessionMap, sess)
	logger.Infof("SSHClient(%s) release one session remain %d", s, len(s.traceSessionMap))
}

func (s *SSHClient) AcquireSession() (*gossh.Session, error) {
	atomic.AddInt32(&s.refCount, 1)
	sess, err := s.Client.NewSession()
	if err != nil {
		atomic.AddInt32(&s.refCount, -1)
		return nil, err
	}
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.traceSessionMap[sess] = time.Now()
	return sess, nil
}

func CreateSSHClient(H config.Host) (*SSHClient, error) {
	sshcfg := gossh.ClientConfig{
		User:            H.Username,
		Auth:            []gossh.AuthMethod{gossh.Password(H.Password)},
		Timeout:         time.Duration(15) * time.Second, //超时15分钟后退出
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}

	gC, err := gossh.Dial("tcp", net.JoinHostPort(H.IP, H.Prot), &sshcfg)
	if err != nil {
		logger.Debug("gosshClientE:", err)
		return nil, err
	}
	return &SSHClient{Client: gC,
		traceSessionMap: make(map[*gossh.Session]time.Time)}, nil
}

func SSHConn(sess *gossh.Session) (*SSHConnection, error) {
	if sess == nil {
		return nil, errors.New("ssh session is nil")
	}
	modes := gossh.TerminalModes{
		gossh.ECHO:          1,     // enable echoing
		gossh.TTY_OP_ISPEED: 14400, // input speed = 14.4 kbaud
		gossh.TTY_OP_OSPEED: 14400, // output speed = 14.4 kbaud
	}
	err := sess.RequestPty("xterm", 120, 80, modes)
	if err != nil {
		return nil, err
	}
	stdin, err := sess.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		return nil, err
	}
	conn := &SSHConnection{
		session: sess,
		stdin:   stdin,
		stdout:  stdout,
		// options: options,
	}
	err = sess.Shell()
	if err != nil {
		_ = sess.Close()
		return nil, err
	}
	return conn, nil

}

type SSHConnection struct {
	session *gossh.Session
	stdin   io.Writer
	stdout  io.Reader
	// options *SSHOptions
}

func (sc *SSHConnection) SetWinSize(w, h int) error {
	return sc.session.WindowChange(h, w)
}

func (sc *SSHConnection) Read(p []byte) (n int, err error) {
	return sc.stdout.Read(p)
}

func (sc *SSHConnection) Write(p []byte) (n int, err error) {
	return sc.stdin.Write(p)
}

func (sc *SSHConnection) Close() (err error) {
	return sc.session.Close()
}

func (sc *SSHConnection) KeepAlive() error {
	_, err := sc.session.SendRequest("keepalive@openssh.com", false, nil)
	return err
}
