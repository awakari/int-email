package service

import (
	"context"
	"github.com/awakari/int-email/config"
	"github.com/awakari/int-email/service/converter"
	"github.com/awakari/int-email/service/writer"
	"github.com/microcosm-cc/bluemonday"
	"github.com/stretchr/testify/assert"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestSvc_Submit(t *testing.T) {
	cases := map[string]struct {
		from     string
		internal bool
		in       io.Reader
		err      error
	}{
		"empty": {
			in:  strings.NewReader(""),
			err: converter.ErrParse,
		},
		"ok": {
			from: "johndoe@example.com",
			in: strings.NewReader(`From: John Doe <john@example.com>
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
		},
		"fail write": {
			from: "fail",
			in: strings.NewReader(`From: John Doe <john@example.com>
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
			err: writer.ErrWrite,
		},
	}
	log := slog.Default()
	s := NewService(
		converter.NewLogging(
			converter.NewConverter(
				"com_awakari_email_v1",
				bluemonday.NewPolicy(),
				config.WriterInternalConfig{},
			),
			log,
		),
		writer.NewLogging(writer.NewMock(), log),
		"default",
	)
	s = NewLogging(s, log)
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			err := s.Submit(context.TODO(), c.from, c.internal, c.in)
			assert.ErrorIs(t, err, c.err)
		})
	}
}
