package smtp

import (
	"context"
	"github.com/awakari/int-email/service/converter"
	"github.com/awakari/int-email/service/writer"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/emersion/go-smtp"
	"github.com/segmentio/ksuid"
	"io"
)

type session struct {
	svcWriter    writer.Service
	rcptsAllowed map[string]bool
	dataLimit    int64
	evtType      string
	conv         converter.Service
	//
	allowed bool
	from    string
	data    []byte
}

func newSession(svcWriter writer.Service, rcptsAllowed map[string]bool, dataLimit int64, evtType string, conv converter.Service) smtp.Session {
	return &session{
		svcWriter:    svcWriter,
		rcptsAllowed: rcptsAllowed,
		dataLimit:    dataLimit,
		evtType:      evtType,
		conv:         conv,
	}
}

func (s *session) Reset() {
	s.allowed = false
	s.from = ""
	s.data = nil
	return
}

func (s *session) Logout() (err error) {
	return
}

func (s *session) Mail(from string, opts *smtp.MailOptions) (err error) {
	s.from = from
	return
}

func (s *session) Rcpt(to string, opts *smtp.RcptOptions) (err error) {
	if s.rcptsAllowed[to] {
		s.allowed = true
	}
	return
}

func (s *session) Data(r io.Reader) (err error) {
	switch s.allowed {
	case true:
		r = io.LimitReader(r, s.dataLimit)
		evt := &pb.CloudEvent{
			Id:          ksuid.New().String(),
			Source:      s.from,
			SpecVersion: "1.0",
			Type:        s.evtType,
			Attributes:  make(map[string]*pb.CloudEventAttributeValue),
		}
		err = s.conv.Convert(r, evt)
		switch err {
		case nil:
			err = s.svcWriter.Write(context.TODO(), evt, "default", s.from)
		default:
			err = &smtp.SMTPError{
				Code: 554,
				EnhancedCode: smtp.EnhancedCode{
					5, 3, 0,
				},
				Message: err.Error(),
			}
		}
	default:
		err = &smtp.SMTPError{
			Code: 550,
			EnhancedCode: smtp.EnhancedCode{
				5, 1, 1,
			},
			Message: "recipient rejected",
		}
	}
	return
}
