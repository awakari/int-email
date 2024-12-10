package smtp

import (
	"context"
	"fmt"
	"github.com/awakari/int-email/util"
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
	bl.log.Log(context.TODO(), util.LogLevel(err), fmt.Sprintf("backend.NewSession(%s): %+v, %t, %s", c.Hostname(), tls, tlsOk, err))
	return
}
