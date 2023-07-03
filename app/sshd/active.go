package sshd

import (
	"context"
	"fmt"
	"io"
	"mjump/app/logger"
	"mjump/app/proxy"
	"mjump/app/utils"
	"strings"
	"sync"
	"time"

	config "mjump/app/data"

	"github.com/gliderlabs/ssh"
	"github.com/jedib0t/go-pretty/table"
	uuid "github.com/satori/go.uuid"
)

func InitactiveH(sess ssh.Session) *DispPanel {
	cnf, err := config.Loadcnf()
	if err != nil {
		logger.Error("解析Json 文件错误.", err)
		return nil
	}
	users, err := GetUser(sess.User())
	if err != nil {
		return nil
	}
	var uHost []config.Host
	for i := 0; i < len(cnf.Hosts); i++ {
		if utils.In(cnf.Hosts[i].ID, users.Assets) {
			uHost = append(uHost, cnf.Hosts[i])
		}
	}
	wrSess := &WrapperSession{
		Sess:  sess,
		mux:   new(sync.RWMutex),
		winch: make(chan ssh.Window),
	}
	term := utils.NewTerminal(wrSess, "OPTIONS>")
	wrSess.initial()
	h := &DispPanel{
		wsession: wrSess,
		term:     term,
		hosts:    uHost,
		page:     1,
		pagemax:  10,
	}

	h.InitPanel()
	return h
}

type DispPanel struct {
	wsession *WrapperSession
	term     *utils.Terminal
	hosts    []config.Host
	page     int
	pagemax  int
	// selectHandler *UserSelectHandler
}

func (d *DispPanel) Home() {
	d.term.SetPrompt("Options> ")
	_, err := io.WriteString(d.wsession.Sess, "\x1b[H\x1b[3J\x1b[2J\r\n\t\t欢迎使用 Mjump 迷你堡垒机\r\n")
	if err != nil {
		logger.Errorf("Send to client error, %s", err)
		return
	}
	io.WriteString(d.wsession.Sess, "\t\t          _                          \r\n")
	io.WriteString(d.wsession.Sess, "\t\t  /\\/\\   (_) _   _  _ __ ___   _ __  \r\n")
	io.WriteString(d.wsession.Sess, "\t\t /    \\  | || | | || '_ \\` _ \\| '_ \\ \r\n")
	io.WriteString(d.wsession.Sess, "\t\t/ /\\/\\ \\ | || |_| || | | | | || |_) |\r\n")
	io.WriteString(d.wsession.Sess, "\t\t\\/    \\/_/ | \\__,_||_| |_| |_|| .__/ \r\n")
	io.WriteString(d.wsession.Sess, "\t\t       |__/                   |_|    \r\n\r\n")
	io.WriteString(d.wsession.Sess, "\t\t输入 \033[1;36m回车\033[0m  显示可连接的主机列表\r\n")
	io.WriteString(d.wsession.Sess, "\t\t输入 \033[1;36mh\033[0m     进入首页\r\n")
	io.WriteString(d.wsession.Sess, "\t\t输入 \033[1;36mn\033[0m     下一页\r\n")
	io.WriteString(d.wsession.Sess, "\t\t输入 \033[1;36mb\033[0m     上一页\r\n")
	io.WriteString(d.wsession.Sess, "\t\t输入 \033[1;36mr\033[0m     刷新新增主机\r\n")
	io.WriteString(d.wsession.Sess, "\t\t输入 \033[1;36mq\033[0m     退出终端\r\n\r\n")

}

func (d *DispPanel) HostsPage(nex string) {
	// io.WriteString(d.wsession.Sess, "\x1b[H\x1b[2J")
	// io.WriteString(d.wsession.Sess, "\x1b[2J")
	io.WriteString(d.wsession.Sess, "\x1b[3J\x1b[H\x1b[2J")
	d.term.SetPrompt("IP/ID> ")
	w, h := d.term.GetSize()
	logger.Debug("宽度", w, "高度", h)
	title := "可用主机列表"
	tc := (w / 2) - (len(title) / 2)
	tr := ""
	for i := 0; i < tc; i++ {
		tr += " "
	}
	t := table.NewWriter()
	t.SetStyle(table.Style{
		Name: "myNewStyle",
		Box: table.BoxStyle{
			MiddleHorizontal: "-",
			MiddleSeparator:  "-",
			MiddleVertical:   " ",
		},
		Options: table.Options{
			SeparateColumns: true,
			SeparateHeader:  true,
		},
	})
	t.SetTitle("可用主机列表")
	t.AppendHeader(table.Row{"ID", "名称", "IP地址"})
	t.SetColumnConfigs([]table.ColumnConfig{{
		Name:     "IP地址",
		WidthMin: w - 45,
		// WidthMin: 15,
	}})
	total := len(d.hosts) / d.pagemax
	if len(d.hosts)%d.pagemax != 0 {
		total += 1
	}
	switch nex {
	case "n":
		if d.page < total {
			d.page += 1
		}
	case "b":
		if d.page != 1 {
			d.page -= 1
		}
	default:
		d.page = 1
	}
	end := d.page * d.pagemax
	art := end - d.pagemax
	if d.page == total {
		end = art + (len(d.hosts) % d.pagemax)
	}
	for _, row := range d.hosts[art:end] {
		t.AppendRows([]table.Row{
			{row.ID, row.Name, row.IP},
		})
	}
	tab_b := fmt.Sprintf("\r\n每页: %d 总数: %d  当前: (%d/%d)  翻页: (n/b)\r\n", d.pagemax, len(d.hosts), d.page, total)
	_, _ = d.term.Write([]byte(t.Render() + "\r\n"))
	_, _ = d.term.Write([]byte(tab_b))

}
func (d *DispPanel) FilterH(key string) (H *config.Host) {
	for _, row := range d.hosts {
		if key == row.ID {
			return &row
		}
	}
	for _, row := range d.hosts {
		if key == row.IP {
			return &row
		}
	}
	return nil
}
func (d *DispPanel) SelectHost(line string) {
	host := d.FilterH(line)
	if host == nil {
		_, _ = d.term.Write([]byte("\033[1;32m没有找到可用的主机\033[0m\r\n"))
		return
	}
	proxy.Proxy(d.wsession, *host)
}

func (d *DispPanel) Displayterm() {
	d.Home()
	defer logger.Infof("Request %s: User %s stop interactive", d.wsession.ID(), "root")
	// var Inception bool
	for {
		line, err := d.term.ReadLine()
		if err != nil {
			logger.Debugf("User %s close connect", "root")
			break
		}
		line = strings.TrimSpace(line)

		if len(line) == 0 {
			d.HostsPage("e")
			continue
		}
		switch len(line) {
		case 1:
			switch strings.ToLower(line) {
			case "n":
				d.HostsPage("n")
				continue
			case "b":
				d.HostsPage("b")
				continue
			case "h":
				d.Home()
				continue
			case "r":
				cnf, err := config.Loadcnf()
				if err != nil {
					_, _ = d.term.Write([]byte("\033[1;31m加载主机信息失败\033[0m\r\n"))
				} else {
					var uHost []config.Host
					users, err := GetUser(d.wsession.Sess.User())
					if err != nil {
						return
					}
					for i := 0; i < len(cnf.Hosts); i++ {
						if utils.In(cnf.Hosts[i].ID, users.Assets) {
							uHost = append(uHost, cnf.Hosts[i])
						}
					}
					d.hosts = uHost
					_, _ = d.term.Write([]byte("\033[1;32m刷新主机列表完成\033[0m\r\n"))
				}

				continue
			case "q":
				return
			}
		default:
			if line == "exit" || line == "quit" {
				return
			}
		}
		d.SelectHost(line)
	}
}

func (d *DispPanel) WatcheChangeWin(winChan <-chan ssh.Window) {
	defer logger.Infof("Request %s: Windows change watch close", d.wsession.Uuid)
	for {
		select {
		case <-d.wsession.Sess.Context().Done():
			return
		case win, ok := <-winChan:
			if !ok {
				return
			}
			d.wsession.SetWin(win)
			logger.Debugf("Term window size change: %d*%d", win.Height, win.Width)
			_ = d.term.SetSize(win.Width, win.Height)
		}
	}
}

func UUID() string {
	return uuid.NewV4().String()
}

func (h *DispPanel) InitPanel() {
	go h.keepSessionAlive(time.Duration(30) * time.Second)
}
func (h *DispPanel) keepSessionAlive(keepAliveTime time.Duration) {
	t := time.NewTicker(keepAliveTime)
	defer t.Stop()
	for {
		select {
		case <-h.wsession.Sess.Context().Done():
			return
		case <-t.C:
			_, err := h.wsession.Sess.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				logger.Errorf("Request %s: Send user root keepalive packet failed: %s",
					h.wsession.Uuid, err)
				continue
			}
			logger.Debugf("Request %s: Send userroot keepalive packet success", h.wsession.Uuid)
		}
	}
}

type WrapperSession struct {
	Uuid      string
	Sess      ssh.Session
	inWriter  io.WriteCloser
	outReader io.ReadCloser
	mux       *sync.RWMutex

	closed chan struct{}

	winch      chan ssh.Window
	currentWin ssh.Window
}

func (w *WrapperSession) ID() string {
	return w.Uuid
}
func (w *WrapperSession) initial() {
	w.Uuid = UUID()
	w.closed = make(chan struct{})
	w.initReadPip()
	go w.readLoop()
}

func (w *WrapperSession) initReadPip() {
	w.mux.Lock()
	defer w.mux.Unlock()
	w.outReader, w.inWriter = io.Pipe()
}

func (w *WrapperSession) readLoop() {
	buf := make([]byte, 1024*8)
	for {
		nr, err := w.Sess.Read(buf)

		if nr > 0 {
			w.mux.RLock()
			_, _ = w.inWriter.Write(buf[:nr])
			w.mux.RUnlock()
		}
		if err != nil {
			break
		}
	}
	w.mux.RLock()
	_ = w.inWriter.Close()
	_ = w.outReader.Close()
	w.mux.RUnlock()
	close(w.closed)
	logger.Infof("Request %s: Read loop break", w.Uuid)
}

func (w *WrapperSession) SetWin(win ssh.Window) {
	select {
	case w.winch <- win:
	default:
	}
	w.currentWin = win
}

func (w *WrapperSession) Read(p []byte) (int, error) {
	select {
	case <-w.closed:
		return 0, io.EOF
	default:
	}
	w.mux.RLock()
	defer w.mux.RUnlock()
	return w.outReader.Read(p)
}
func (w *WrapperSession) Close() error {
	select {
	case <-w.closed:
		return nil
	default:
	}
	_ = w.inWriter.Close()
	err := w.outReader.Close()
	w.initReadPip()
	return err
}

func (w *WrapperSession) Write(p []byte) (int, error) {
	return w.Sess.Write(p)
}

func (w *WrapperSession) Context() context.Context {
	return w.Sess.Context()
}

func (w *WrapperSession) WinCh() (winch <-chan ssh.Window) {
	return w.winch
}
func (w *WrapperSession) Pty() ssh.Pty {
	pty, _, _ := w.Sess.Pty()
	termWin := w.currentWin
	if w.currentWin.Width == 0 || w.currentWin.Height == 0 {
		termWin = pty.Window
	}
	return ssh.Pty{
		Term:   pty.Term,
		Window: termWin,
	}
}
