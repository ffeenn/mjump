package main

import (
	config "mjump/app/data"
	"mjump/app/logger"
	"mjump/app/sshd"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger.Initlog()
	cnf, err := config.Loadcnf()
	if err != nil {
		logger.Error("解析Json 文件错误.", err)
		return
	}
	SignalStop := make(chan os.Signal, 1)
	signal.Notify(SignalStop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	sshs := sshd.NewSSHServer(cnf)
	go sshs.Start()

	<-SignalStop
}

//echo "          _                          ";
//echo "  /\/\   (_) _   _  _ __ ___   _ __  ";
//echo " /    \  | || | | || '_ \` _ \ | '_ \ ";
//echo "/ /\/\ \ | || |_| || | | | | || |_) |";
//echo "\/    \/_/ | \__,_||_| |_| |_|| .__/ ";
//echo "       |__/                   |_|    ";
