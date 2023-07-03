package proxy

import (
	config "mjump/app/data"
)

type Server struct {
	ID                       string
	UserConn                 UserConnection
	H                        config.Host
	flag                     bool
	CreateSessionCallback    func() error
	ConnectedSuccessCallback func() error
	ConnectedFailedCallback  func(err error) error
	DisConnectedCallback     func() error
}
