package smtp

import (
	"github.com/awakari/int-email/service"
	"github.com/emersion/go-smtp"
)

type backend struct {
	rcptsPublish  map[string]bool
	rcptsInternal map[string]bool
	dataLimit     int64
	svc           service.Service
}

func NewBackend(rcptsPublish, rcptsInternal map[string]bool, dataLimit int64, svc service.Service) smtp.Backend {
	return backend{
		rcptsPublish:  rcptsPublish,
		rcptsInternal: rcptsInternal,
		dataLimit:     dataLimit,
		svc:           svc,
	}
}

func (b backend) NewSession(c *smtp.Conn) (s smtp.Session, err error) {
	s = newSession(b.rcptsPublish, b.rcptsInternal, b.dataLimit, b.svc)
	return
}
