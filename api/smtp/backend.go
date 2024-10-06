package smtp

import (
    "github.com/awakari/int-email/service/writer"
    "github.com/emersion/go-smtp"
)

type backend struct {
    svcWriter writer.Service
    rcpts     map[string]bool
    dataLimit int64
}

func NewBackend(svcWriter writer.Service, rcpts map[string]bool, dataLimit int64) smtp.Backend {
    return backend{
        svcWriter: svcWriter,
        rcpts:     rcpts,
        dataLimit: dataLimit,
    }
}

func (b backend) NewSession(c *smtp.Conn) (s smtp.Session, err error) {
    s = newSession(b.svcWriter, b.rcpts, b.dataLimit)
    return
}
