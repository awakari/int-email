package smtp

import (
	"github.com/awakari/int-email/service"
	"github.com/emersion/go-smtp"
)

type backend struct {
	rcpts     map[string]bool
	dataLimit int64
	svc       service.Service
}

func NewBackend(rcpts map[string]bool, dataLimit int64, svc service.Service) smtp.Backend {
	return backend{
		rcpts:     rcpts,
		dataLimit: dataLimit,
		svc:       svc,
	}
}

func (b backend) NewSession(c *smtp.Conn) (s smtp.Session, err error) {
	s = newSession(b.rcpts, b.dataLimit, b.svc)
	return
}
