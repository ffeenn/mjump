package proxy

import (
	"context"
	"fmt"
	"io"
	"mjump/app/logger"
	"time"

	config "mjump/app/data"

	"github.com/gliderlabs/ssh"
)

type UserConnection interface {
	io.ReadWriteCloser
	ID() string
	WinCh() <-chan ssh.Window
	Pty() ssh.Pty
	Context() context.Context
}

func Proxy(conn UserConnection, H config.Host) {
	src := Server{
		ID:       "1",
		UserConn: conn,
		H:        H,
	}
	src.SSHconn()
	logger.Infof("Request : asset  proxy end")
}
func (s *Server) SSHconn() {
	ctx, cancel := context.WithCancel(context.Background())
	sw := SwitchSession{
		ID:            s.ID,
		MaxIdleTime:   5000,
		keepAliveTime: 60,
		ctx:           ctx,
		cancel:        cancel,
		p:             s,
	}
	AddCommonSwitch(&sw)
	defer RemoveCommonSwitch(&sw)
	ch := make(chan struct{})
	defer func() {
		close(ch)
	}()
	go s.TemWaitStr(ch)
	sshC, err := CreateSSHClient(s.H)
	if err != nil {
		logger.Errorf("ssh client err: %s", err)
		s.UserConn.Write([]byte(fmt.Sprintf("\x08\r\n连接 %v 错误：%v\r\n", s.H.IP, err)))
		return
	}
	s.flag = true
	s.UserConn.Write([]byte("\x08\x1b[3J\x1b[H\x1b[2J"))
	sess, err := sshC.AcquireSession()
	if err != nil {
		logger.Errorf("SSH client(%s) start session err %s", sshC, err)
		return
	}
	sshConn, err := SSHConn(sess)
	if err != nil {
		_ = sess.Close()
		sshC.ReleaseSession(sess)
		return
	}
	go func() {
		_ = sess.Wait()
		sshC.ReleaseSession(sess)
		logger.Infof("SSH client(%s) shell connection release", sshC)
	}()
	defer sshConn.Close()
	logger.Infof("Conn[%s] create session %s success", s.UserConn.ID(), s.ID)
	// utils.IgnoreErrWriteWindowTitle(s.UserConn, s.connOpts.TerminalTitle())
	if err = sw.Bridge(s.UserConn, sshConn); err != nil {
		logger.Error(err)
	}

}
func (s *Server) TemWaitStr(c chan struct{}) {
	ds := []string{"|", "/", "—", "\\"}
	ic := 0
	s.UserConn.Write([]byte(fmt.Sprintf("Connecting to %v:%v ...  ", s.H.IP, s.H.Prot)))
	for {
		select {
		case <-c:
			return
		default:
			if s.flag {
				break
			}
			s.UserConn.Write([]byte(fmt.Sprintf("\x08%v", ds[ic])))
		}
		if ic >= 3 {
			ic = 0
		} else {
			ic += 1
		}
		time.Sleep(100 * time.Millisecond)
	}
}
