package smtp

import (
	"fmt"
	"github.com/emersion/go-smtp"
	"log/slog"
)

type backendLogging struct {
	b   smtp.Backend
	log *slog.Logger
}

func NewBackendLogging(b smtp.Backend, log *slog.Logger) smtp.Backend {
	return backendLogging{
		b:   b,
		log: log,
	}
}

func (bl backendLogging) NewSession(c *smtp.Conn) (s smtp.Session, err error) {
	tls, tlsOk := c.TLSConnectionState()
	s, err = bl.b.NewSession(c)
	switch err {
	case nil:
		bl.log.Debug(fmt.Sprintf("backend.NewSession(%s, %+v, %t)", c.Hostname(), tls, tlsOk))
		s = NewSessionLogging(s, bl.log)
	default:
		bl.log.Error(fmt.Sprintf("backend.NewSession(%s, %+v, %t): err=%s", c.Hostname(), tls, tlsOk, err))
	}
	return
}
