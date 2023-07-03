package proxy

import (
	"context"
	"mjump/app/logger"
	"time"
)

type SwitchSession struct {
	ID string

	MaxIdleTime   int
	keepAliveTime int

	ctx    context.Context
	cancel context.CancelFunc

	p *Server
}

func (s *SwitchSession) Bridge(userConn UserConnection, srvConn ServerConnection) (err error) {
	logger.Infof("Conn[%s] create ParseEngine success", userConn.ID())
	srvInChan := make(chan []byte, 1)
	usrInChan := make(chan []byte, 1)
	done := make(chan struct{})
	winCh := userConn.WinCh()
	maxIdleTime := time.Duration(120) * time.Minute
	lastActiveTime := time.Now()
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	exitSignal := make(chan struct{}, 2)
	go func() {
		var (
			exitFlag bool
			err      error
			nr       int
		)
		for {
			buf := make([]byte, 1024)
			nr, err = srvConn.Read(buf)
			if nr > 0 {
				select {
				case srvInChan <- buf[:nr]:
				case <-done:
					exitFlag = true
					logger.Infof("Session[%s] done", s.ID)
				}
				if exitFlag {
					break
				}
			}
			if err != nil {
				logger.Errorf("Session[%s] srv read err: %s", s.ID, err)
				break
			}
		}
		logger.Infof("Session[%s] srv read end", s.ID)
		exitSignal <- struct{}{}
		close(srvInChan)
	}()
	go func() {
		var (
			exitFlag bool
		)
		for {
			buf := make([]byte, 1024)
			nr, err := userConn.Read(buf)
			if nr > 0 {
				select {
				case usrInChan <- buf[:nr]:
				case <-done:
					exitFlag = true
					logger.Infof("Session[%s] done", s.ID)
				}
				if exitFlag {
					break
				}
				if err != nil {
					logger.Errorf("Session[%s] srv read err: %s", s.ID, err)
					break
				}
			}
			if err != nil {
				logger.Errorf("Session[%s] user read err: %s", s.ID, err)
				break
			}
		}
		logger.Infof("Session[%s] user read end", s.ID)
		exitSignal <- struct{}{}
	}()
	keepAliveTime := time.Duration(s.keepAliveTime) * time.Second
	keepAliveTick := time.NewTicker(keepAliveTime)
	defer keepAliveTick.Stop()
	for {
		select {
		// 检测是否超过最大空闲时间
		case now := <-tick.C:
			outTime := lastActiveTime.Add(maxIdleTime)
			if now.After(outTime) {
				logger.Infof("Session[%s] idle more than %d minutes, disconnect", s.ID, 60)
				return
			}
			continue
		case <-s.ctx.Done():
			logger.Infof("Session[%s]: close", s.ID)
			return
			// 监控窗口大小变化
		case win, ok := <-winCh:
			if !ok {
				return
			}
			_ = srvConn.SetWinSize(win.Width, win.Height)
			logger.Infof("Session[%s] Window server change: %d*%d",
				s.ID, win.Width, win.Height)
		case p, ok := <-srvInChan:
			if !ok {
				return
			}
			if _, err := userConn.Write(p); err != nil {
				logger.Errorf("Session[%s] userConn write err: %s", s.ID, err)
			}
		case p, ok := <-usrInChan:
			if !ok {
				return
			}
			if _, err := srvConn.Write(p); err != nil {
				logger.Errorf("Session[%s] srvConn write err: %s", s.ID, err)
			}

		case now := <-keepAliveTick.C:
			if now.After(lastActiveTime.Add(keepAliveTime)) {
				if err := srvConn.KeepAlive(); err != nil {
					logger.Errorf("Session[%s] srvCon keep alive err: %s", s.ID, err)
				}
			}
			continue
		case <-userConn.Context().Done():
			logger.Infof("Session[%s]: user conn context done", s.ID)
			return nil
		case <-exitSignal:
			logger.Debugf("Session[%s] end by exit signal", s.ID)
			return
		}
		lastActiveTime = time.Now()
	}
}
