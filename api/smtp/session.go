package smtp

import (
	"context"
	"github.com/awakari/int-email/service"
	"github.com/emersion/go-smtp"
	"io"
)

type session struct {
	rcptsAllowed map[string]bool
	dataLimit    int64
	svc          service.Service
	//
	allowed bool
	from    string
}

func newSession(rcptsAllowed map[string]bool, dataLimit int64, svc service.Service) smtp.Session {
	return &session{
		rcptsAllowed: rcptsAllowed,
		dataLimit:    dataLimit,
	}
}

func (s *session) Reset() {
	s.allowed = false
	s.from = ""
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
		err = s.svc.Submit(context.TODO(), s.from, r)
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
