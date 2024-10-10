package writer

import (
	"context"
	"fmt"
	"github.com/awakari/int-email/util"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
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

func (l logging) Close() (err error) {
	err = l.svc.Close()
	l.log.Log(context.TODO(), util.LogLevel(err), fmt.Sprintf("writer.Close(): %s", err))
	return
}

func (l logging) Write(ctx context.Context, evt *pb.CloudEvent, groupId, userId string) (err error) {
	err = l.svc.Write(ctx, evt, groupId, userId)
	l.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("writer.Write(evt=%s, groupId=%s, userId=%s): %s", evt.Id, groupId, userId, err))
	return
}
