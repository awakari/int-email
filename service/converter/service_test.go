package converter

import (
	"fmt"
	"github.com/awakari/int-email/config"
	"github.com/awakari/int-email/util"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/microcosm-cc/bluemonday"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestSvc_Convert(t *testing.T) {
	cases := map[string]struct {
		r        io.Reader
		internal bool
		out      *pb.CloudEvent
		err      error
	}{
		"empty": {
			r:   strings.NewReader(``),
			err: ErrParse,
		},
		"no date": {
			r: strings.NewReader(`From: John Doe <john@example.com>
To: Jane Smith <jane.smith@example.com>
Subject: Meeting Notes and Attachment
Message-ID: <unique-message-id@example.com>
MIME-Version: 1.0
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: 7bit

Hi Jane,

Please find attached the meeting notes and presentation slides.

Best regards,
John`),
			err: ErrParse,
		},
		"no message id": {
			r: strings.NewReader(`From: John Doe <john@example.com>
To: Jane Smith <jane.smith@example.com>
Subject: Meeting Notes and Attachment
Date: Thu, 10 Oct 2024 12:34:56 +0000
MIME-Version: 1.0
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: 7bit

Hi Jane,

Please find attached the meeting notes and presentation slides.

Best regards,
John`),
			err: ErrParse,
		},
		"parse fail": {
			r:   strings.NewReader(`?`),
			err: ErrParse,
		},
		"ok": {
			r: strings.NewReader(`From: John Doe <john@example.com>
To: Jane Smith <jane.smith@example.com>
Subject: Meeting Notes and Attachment
Date: Thu, 10 Oct 2024 12:34:56 +0000
Message-ID: <unique-message-id@example.com>
MIME-Version: 1.0
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: 7bit

Hi Jane,

Please find attached the meeting notes and presentation slides.

Best regards,
John`),
			out: &pb.CloudEvent{
				Attributes: map[string]*pb.CloudEventAttributeValue{
					"contenttype": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "text/plain; charset=\"UTF-8\"",
						},
					},
					"contenttransferencod": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "7bit",
						},
					},
					"objecturl": {
						Attr: &pb.CloudEventAttributeValue_CeUri{
							CeUri: "unique-message-id@example.com",
						},
					},
					"mimeversion": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "1.0",
						},
					},
					"subject": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "Meeting Notes and Attachment",
						},
					},
					"time": {
						Attr: &pb.CloudEventAttributeValue_CeTimestamp{
							CeTimestamp: &timestamppb.Timestamp{
								Seconds: 1728563696,
							},
						},
					},
				},
			},
		},
		"internal": {
			internal: true,
			r: strings.NewReader(`From: John Doe <john@example.com>
To: Jane Smith <jane.smith@example.com>
Subject: Meeting Notes and Attachment
Date: Thu, 10 Oct 2024 12:34:56 +0000
Message-ID: <unique-message-id@example.com>
MIME-Version: 1.0
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: 7bit

Hi Jane,

Please find attached the meeting notes and presentation slides.

Best regards,
John`),
			out: &pb.CloudEvent{
				Attributes: map[string]*pb.CloudEventAttributeValue{
					"awkinternal": {
						Attr: &pb.CloudEventAttributeValue_CeInteger{
							CeInteger: 12345,
						},
					},
					"contenttype": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "text/plain; charset=\"UTF-8\"",
						},
					},
					"contenttransferencod": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "7bit",
						},
					},
					"objecturl": {
						Attr: &pb.CloudEventAttributeValue_CeUri{
							CeUri: "unique-message-id@example.com",
						},
					},
					"mimeversion": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "1.0",
						},
					},
					"subject": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "Meeting Notes and Attachment",
						},
					},
					"time": {
						Attr: &pb.CloudEventAttributeValue_CeTimestamp{
							CeTimestamp: &timestamppb.Timestamp{
								Seconds: 1728563696,
							},
						},
					},
				},
			},
		},
		"invalid date format": {
			r: strings.NewReader(`From: John Doe <john@example.com>
To: Jane Smith <jane.smith@example.com>
Subject: Meeting Notes and Attachment
Date: Thu, 40 Oct 1024 12-34:56
MIME-Version: 1.0
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: 7bit

Hi Jane,

Please find attached the meeting notes and presentation slides.

Best regards,
John`),
			err: ErrParse,
		},
	}
	conv := NewConverter("com_awakari_email_v1", bluemonday.NewPolicy(), config.WriterInternalConfig{
		Name:  "awkinternal",
		Value: 12345,
	})
	conv = NewLogging(conv, slog.Default())
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			dst := &pb.CloudEvent{
				Attributes: make(map[string]*pb.CloudEventAttributeValue),
			}
			err := conv.Convert(c.r, dst, c.internal)
			if c.err == nil {
				assert.NotZero(t, dst.Id)
				for attrK, attrV := range c.out.Attributes {
					assert.True(t, dst.Attributes[attrK] != nil, attrK)
					assert.Equal(t, dst.Attributes[attrK].Attr, attrV.Attr, attrK)
				}
			}
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func Test_handleHtml(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping test in CI environment")
	}
	d, err := os.ReadFile("emaildata.html")
	require.Nil(t, err)
	conv := svc{
		htmlPolicy: util.HtmlPolicy(),
	}
	evt := &pb.CloudEvent{
		Attributes: make(map[string]*pb.CloudEventAttributeValue),
	}
	err = conv.handleHtml(string(d), evt)
	assert.Nil(t, err)
	fmt.Printf("%+v\n", evt.Attributes)
}
