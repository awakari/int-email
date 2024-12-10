package service

import (
	"context"
	"errors"
	"github.com/awakari/int-email/api/http/pub"
	"github.com/awakari/int-email/service/converter"
	"github.com/cenkalti/backoff/v4"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"io"
	"time"
)

type Service interface {
	Submit(ctx context.Context, from string, internal bool, r io.Reader) (err error)
}

type svc struct {
	svcConv          converter.Service
	svcPub           pub.Service
	groupId          string
	backoffTimeLimit time.Duration
}

const backoffInitDelay = 100 * time.Millisecond

func NewService(svcConv converter.Service, svcPub pub.Service, groupId string, backoffTimeLimit time.Duration) Service {
	return svc{
		svcConv:          svcConv,
		svcPub:           svcPub,
		groupId:          groupId,
		backoffTimeLimit: backoffTimeLimit,
	}
}

func (s svc) Submit(ctx context.Context, from string, internal bool, r io.Reader) (err error) {
	evt := &pb.CloudEvent{
		Attributes: make(map[string]*pb.CloudEventAttributeValue),
	}
	err = s.svcConv.Convert(r, evt, from, internal)
	if err == nil {
		err = s.svcPub.Publish(ctx, evt, s.groupId, evt.Source)
		if errors.Is(err, pub.ErrNoAck) {
			err = s.retryBackoff(func() error {
				return s.svcPub.Publish(ctx, evt, s.groupId, evt.Source)
			})
		}
	}
	return
}

func (s svc) retryBackoff(op func() error) (err error) {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = backoffInitDelay
	b.MaxElapsedTime = s.backoffTimeLimit
	err = backoff.Retry(op, b)
	return
}
