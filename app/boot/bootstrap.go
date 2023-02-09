package boot

import (
	"context"
	"icetea/service"
	"log"
)

type Service interface {
	Run(ctx context.Context) error
	Stop() error
	Name() string
}
type App struct {
}

func (*App) Run(ctx context.Context) error {
	var (
		services = []Service{
			newServer(),
			service.NewLogger(),
		}
	)

	for _, s := range services {
		if err := s.Run(ctx); err != nil {
			return err
		}
		log.Println(`service: ` + s.Name() + ` started`)
	}
	return nil
}
