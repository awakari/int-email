package smtp

import (
    "context"
    "fmt"
    "github.com/emersion/go-smtp"
    "io"
    "log/slog"
)

type sessionLogging struct {
    s   smtp.Session
    log *slog.Logger
}

func NewSessionLogging(s smtp.Session, log *slog.Logger) smtp.Session {
    return sessionLogging{
        s:   s,
        log: log,
    }
}

func (sl sessionLogging) Reset() {
    sl.s.Reset()
    sl.log.Debug("session.Reset()")
    return
}

func (sl sessionLogging) Logout() (err error) {
    err = sl.s.Logout()
    sl.log.Log(context.TODO(), logLevel(err), fmt.Sprintf("session.Logout(): err=%s", err))
    return
}

func (sl sessionLogging) Mail(from string, opts *smtp.MailOptions) (err error) {
    err = sl.s.Mail(from, opts)
    sl.log.Log(context.TODO(), logLevel(err), fmt.Sprintf("session.Mail(from=%s, opts=%+v): err=%s", from, opts, err))
    return
}

func (sl sessionLogging) Rcpt(to string, opts *smtp.RcptOptions) (err error) {
    err = sl.s.Rcpt(to, opts)
    sl.log.Log(context.TODO(), logLevel(err), fmt.Sprintf("session.Rcpt(to=%s, opts=%+v): err=%s", to, opts, err))
    return
}

func (sl sessionLogging) Data(r io.Reader) (err error) {
    err = sl.s.Data(r)
    sl.log.Log(context.TODO(), logLevel(err), fmt.Sprintf("session.Data(): err=%s", err))
    return
}

func logLevel(err error) (lvl slog.Level) {
    switch err {
    case nil:
        lvl = slog.LevelDebug
    default:
        lvl = slog.LevelError
    }
    return
}
