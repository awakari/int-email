package service

import (
	"context"
	"fmt"
	"github.com/awakari/int-email/util"
	"io"
	"log/slog"
)

type logging struct {
	svc Service
	log *slog.Logger
}

func NewLogging(svc Service, log *slog.Logger) Service {
	return logging{
		svc: svc,
		log: log,
	}
}

func (l logging) Submit(ctx context.Context, from string, r io.Reader) (err error) {
	err = l.svc.Submit(ctx, from, r)
	l.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("service.Submit(from=%s): %s", from, err))
	return
}
