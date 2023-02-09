package main

import (
	"context"
	"icetea/app/boot"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var (
		rootCtx, cancel = context.WithCancel(context.TODO())
		sign            = make(chan os.Signal, 1)
		err             error
		app             = &boot.App{}
	)

	err = app.Run(rootCtx)
	if err != nil {
		log.Fatalln(err)
	}
	signal.Notify(sign, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGUSR1)
	<-sign
	cancel()
}
