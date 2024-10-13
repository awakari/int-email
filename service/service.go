package service

import (
	"context"
	"github.com/awakari/int-email/service/converter"
	"github.com/awakari/int-email/service/writer"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"io"
)

type Service interface {
	Submit(ctx context.Context, from string, internal bool, r io.Reader) (err error)
}

type svc struct {
	conv   converter.Service
	writer writer.Service
	group  string
}

func NewService(conv converter.Service, writer writer.Service, group string) Service {
	return svc{
		conv:   conv,
		writer: writer,
		group:  group,
	}
}

func (s svc) Submit(ctx context.Context, from string, internal bool, r io.Reader) (err error) {
	evt := &pb.CloudEvent{
		Attributes: make(map[string]*pb.CloudEventAttributeValue),
	}
	err = s.conv.Convert(r, evt, from, internal)
	if err == nil {
		err = s.writer.Write(context.TODO(), evt, s.group, evt.Source)
	}
	return
}
