package smtp

import (
	"context"
	"github.com/awakari/int-email/service"
	"github.com/emersion/go-smtp"
	"io"
	"strings"
)

type session struct {
	rcptsPublish  map[string]bool
	rcptsInternal map[string]bool
	dataLimit     int64
	svc           service.Service
	//
	publish  bool
	internal bool
	from, to string
}

func newSession(rcptsPublish, rcptsInternal map[string]bool, dataLimit int64, svc service.Service) smtp.Session {
	s := &session{
		rcptsPublish:  make(map[string]bool),
		rcptsInternal: make(map[string]bool),
		dataLimit:     dataLimit,
		svc:           svc,
	}
	for r := range rcptsPublish {
		s.rcptsPublish[strings.ToLower(r)] = true
	}
	for r := range rcptsInternal {
		s.rcptsInternal[strings.ToLower(r)] = true
	}
	return s
}

func (s *session) Reset() {
	s.publish = false
	s.internal = false
	s.from, s.to = "", ""
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
	sepIdx := strings.LastIndex(to, "@")
	if sepIdx > 0 {
		name := strings.ToLower(to[:sepIdx])
		if s.rcptsPublish[name] {
			s.publish = true
			s.to = to
		}
		if s.rcptsInternal[name] {
			s.internal = true
			s.to = to
		}
	}
	return
}

func (s *session) Data(r io.Reader) (err error) {
	switch {
	case s.publish, s.internal:
		r = io.LimitReader(r, s.dataLimit)
		err = s.svc.Submit(context.TODO(), s.from, s.internal, r)
		if err != nil {
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
