package smtp

import (
    "github.com/awakari/int-email/service/writer"
    "github.com/emersion/go-smtp"
    "io"
)

type session struct {
    svcWriter    writer.Service
    rcptsAllowed map[string]bool
    dataLimit    int64
    //
    allowed bool
    from    string
    data    []byte
}

func newSession(svcWriter writer.Service, rcptsAllowed map[string]bool, dataLimit int64) smtp.Session {
    return &session{
        svcWriter:    svcWriter,
        rcptsAllowed: rcptsAllowed,
        dataLimit:    dataLimit,
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
        s.data, err = io.ReadAll(r)
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
