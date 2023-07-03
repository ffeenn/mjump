package sshd

import (
	"context"
	"errors"
	"io"
	config "mjump/app/data"
	"mjump/app/logger"
	"mjump/app/utils"
	"net"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/pires/go-proxyproto"
	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
)

type Server struct {
	Srv  *ssh.Server
	Conf config.Config
}

func (s *Server) Start() {
	logger.Infof("Start SSH server %s", s.Srv.Addr)
	ln, err := net.Listen("tcp", s.Srv.Addr)
	if err != nil {
		logger.Fatal(err)
	}
	// proxyListener :=
	logger.Fatal(s.Srv.Serve(&proxyproto.Listener{Listener: ln}))
}

func (s *Server) Stop() {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	logger.Fatal(s.Srv.Shutdown(ctx))
}

func NewSSHServer(config config.Config) *Server {
	ListenAddr := net.JoinHostPort(config.Listen.Host, config.Listen.Port)
	srv := &ssh.Server{
		// 允许本地端口转发的回调，如果为nil则拒绝所有
		// LocalPortForwardingCallback: false,
		Addr: ListenAddr,
		// 键盘交互身份验证处理程序
		// KeyboardInteractiveHandler: func(ctx ssh.Context, challenger gossh.KeyboardInteractiveChallenge) bool {
		// 	return KeyboardInteractiveAuth(ctx, challenger)
		// },
		PasswordHandler: PasswordAuth,
		// 公钥认证处理程序
		PublicKeyHandler: PublicKeyAuth,
		// 下一个验证方法处理程序为2步验证
		// NextAuthMethodsHandler: func(ctx ssh.Context) []string {
		// 	return []string{"keyboard-interactive"}
		// },
		// HostSigners: []ssh.Signer{handler.GetSSHSigner()}, // 主机密钥的私钥，必须至少有一个
		Handler: SessionHandler,
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": SFTPHandler,
		},
		// ChannelHandlers允许重写内置会话处理程序或提供协议的扩展，如tcpip转发。默认情况下，只有启用“session”处理程序。
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"session": ssh.DefaultSessionHandler,
		},
	}
	return &Server{srv, config}
}

func PublicKeyAuth(ctx ssh.Context, key ssh.PublicKey) bool {
	u, err := GetUser(ctx.User())
	if err != nil {
		return false
	}
	if u.Public == "" || len(u.Public) < 20 {
		return false
	}
	pub, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(u.Public))
	return ssh.KeysEqual(key, pub)
}
func KeyboardInteractiveAuth(ctx ssh.Context,
	challenger gossh.KeyboardInteractiveChallenge) bool {
	logger.Debug("KeyboardInteractiveAuth:秘钥认证")
	return false
}

func GetUser(us string) (u config.User, err error) {
	cnf, err := config.Loadcnf()
	if err != nil {
		logger.Error("解析Json 文件错误.", err)
		return u, err
	}
	for i := range cnf.Users {
		if cnf.Users[i].Username == us {
			u = cnf.Users[i]
			break
		}
	}
	logger.Debug(u.Username)
	if u.Username == "" {
		err = errors.New("用户不存在")
		return

	}
	return
}
func PasswordAuth(ctx ssh.Context, password string) bool {
	// logger.Debug("PasswordAuth", ctx.SessionID())
	// ctx.SetValue(ctxID, ctx.SessionID())
	if u, err := GetUser(ctx.User()); err == nil {
		if password == u.Password {
			return true
		}
	}
	return false
}

func SessionHandler(sess ssh.Session) {
	if pty, winChan, isPty := sess.Pty(); isPty {
		// if pty, winChan, isPty := sess.Pty(); isPty {
		logger.Infof("User %s request pty %s", sess.User(), pty.Term)
		inth := InitactiveH(sess)
		go inth.WatcheChangeWin(winChan)
		inth.Displayterm()
		return
	}
}

func SFTPHandler(sess ssh.Session) {
	logger.Infof("User %s request sftp. ", sess.User())

	cnf, err := config.Loadcnf()
	if err != nil {
		logger.Error("解析Json 文件错误.", err)
		return
	}
	var uHost []config.Host
	users, err := GetUser(sess.User())
	if err != nil {
		return
	}
	for i := 0; i < len(cnf.Hosts); i++ {
		if utils.In(cnf.Hosts[i].ID, users.Assets) {
			uHost = append(uHost, cnf.Hosts[i])
		}
	}
	host, _, _ := net.SplitHostPort(sess.RemoteAddr().String())
	uc := uConn{
		Addr:     host,
		modeTime: time.Now().UTC(),
		fc:       map[string]*fCon{},
	}
	UC := &sftpHandler{uConn: &uc, Hosts: uHost}
	hands := sftp.Handlers{
		FileGet:  UC,
		FilePut:  UC,
		FileCmd:  UC,
		FileList: UC,
	}
	Rs := sftp.NewRequestServer(sess, hands)
	if err := Rs.Serve(); err == io.EOF {
		logger.Debugf("SFTP request: Exited session.")
	} else if err != nil {
		logger.Errorf("SFTP request: Server completed with error %s", err)
	}
	_ = Rs.Close()
	UC.Close()
	logger.Infof("SFTP request: Handler exit.")
}
