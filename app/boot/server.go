package boot

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"icetea/client"
	"icetea/service"
	server2 "icetea/service/server"
	"log"
	"strings"
)

var (
	listeners = []server2.Listener{
		server2.NewTCPListener("127.0.0.1", 1883),
	}
)

type Server struct {
}

func newServer() *Server {
	return &Server{}
}

func (s *Server) Name() string {
	return `server`
}

func (s *Server) Run(ctx context.Context) error {
	for _, v := range listeners {
		go func(listener server2.Listener) {
			if err := listener.Listen(); err != nil {
				log.Fatalln(err)
			}
			for {
				select {
				case <-ctx.Done():
					return
				default:
					conn, err := listener.Accept()
					if err == nil {
						// TODO: add to service manager
						cli := client.NewClient(conn, service.Handler)
						if err := cli.Run(ctx); err != nil {
							cli.Close()
							break
						}
						service.DefaultClientService.CreateClient(cli)
					} else {
						logrus.Error(err)
						conn.Close()
						continue
					}
				}
			}
		}(v)
	}
	return nil
}

func (s *Server) Stop() error {
	var (
		errs []string
	)
	for _, v := range listeners {
		if err := v.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) != 0 {
		return errors.New(strings.Join(errs, ","))
	}
	return nil
}
