package config

import (
	"crypto/tls"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	os.Setenv("API_SMTP_HOST", "email.awakari.com")
	os.Setenv("API_WRITER_BACKOFF", "23h")
	os.Setenv("API_WRITER_URI", "writer:56789")
	os.Setenv("LOG_LEVEL", "4")
	os.Setenv("API_SMTP_RECIPIENTS_PUBLISH", "rcpt1,rcpt2")
	os.Setenv("API_SMTP_RECIPIENTS_INTERNAL", "rcpt3,rcpt4")
	os.Setenv("API_WRITER_INTERNAL_VALUE", "123")
	cfg, err := NewConfigFromEnv()
	assert.Nil(t, err)
	assert.Equal(t, 23*time.Hour, cfg.Api.Writer.Backoff)
	assert.Equal(t, "writer:56789", cfg.Api.Writer.Uri)
	assert.Equal(t, slog.LevelWarn, slog.Level(cfg.Log.Level))
	assert.Equal(t, tls.RequireAndVerifyClientCert, cfg.Api.Smtp.Tls.ClientAuthType)
	assert.Equal(t, []string{
		"rcpt1",
		"rcpt2",
	}, cfg.Api.Smtp.Recipients.Publish)
}
