package smtp

import (
	"github.com/awakari/int-email/service/converter"
	"github.com/awakari/int-email/service/writer"
	"github.com/emersion/go-smtp"
)

type backend struct {
	svcWriter writer.Service
	rcpts     map[string]bool
	dataLimit int64
	evtType   string
	conv      converter.Service
}

func NewBackend(svcWriter writer.Service, rcpts map[string]bool, dataLimit int64, evtType string, conv converter.Service) smtp.Backend {
	return backend{
		svcWriter: svcWriter,
		rcpts:     rcpts,
		dataLimit: dataLimit,
		evtType:   evtType,
		conv:      conv,
	}
}

func (b backend) NewSession(c *smtp.Conn) (s smtp.Session, err error) {
	s = newSession(b.svcWriter, b.rcpts, b.dataLimit, b.evtType, b.conv)
	return
}
