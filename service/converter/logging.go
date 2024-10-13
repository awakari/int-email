package converter

import (
	"context"
	"fmt"
	"github.com/awakari/int-email/util"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
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

func (l logging) Convert(src io.Reader, dst *pb.CloudEvent, from string, internal bool) (err error) {
	err = l.svc.Convert(src, dst, from, internal)
	l.log.Log(context.TODO(), util.LogLevel(err), fmt.Sprintf("converter.Convert(source=%s, objectUrl=%s, evtId=%s, from=%s, internal=%t): %s", dst.Source, dst.Attributes[ceKeyObjectUrl], dst.Id, from, internal, err))
	return
}
