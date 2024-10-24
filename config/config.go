package config

import (
	"crypto/tls"
	"github.com/kelseyhightower/envconfig"
	"time"
)

type Config struct {
	Api ApiConfig
	Log struct {
		Level int `envconfig:"LOG_LEVEL" default:"-4" required:"true"`
	}
}

type ApiConfig struct {
	Smtp struct {
		Host string `envconfig:"API_SMTP_HOST" required:"true"`
		Port uint16 `envconfig:"API_SMTP_PORT" default:"465" required:"true"`
		Data struct {
			Limit           uint32 `envconfig:"API_SMTP_DATA_LIMIT" default:"1048576" required:"true"`
			TruncUrlQueries bool   `envconfig:"API_SMTP_DATA_TRUNC_URL_QUERIES" default:"false"`
		}
		Recipients struct {
			Publish  []string `envconfig:"API_SMTP_RECIPIENTS_PUBLISH" required:"true"`
			Internal []string `envconfig:"API_SMTP_RECIPIENTS_INTERNAL" required:"true"`
			Limit    uint16   `envconfig:"API_SMTP_RECIPIENTS_LIMIT" default:"100" required:"true"`
		}
		Timeout struct {
			Read  time.Duration `envconfig:"API_SMTP_TIMEOUT_READ" default:"1m" required:"true"`
			Write time.Duration `envconfig:"API_SMTP_TIMEOUT_WRITE" default:"1m" required:"true"`
		}
		Tls struct {
			CertPath       string             `envconfig:"API_SMTP_TLS_CERT_PATH" default:"/etc/smtp/tls/tls.crt" required:"true"`
			KeyPath        string             `envconfig:"API_SMTP_TLS_KEY_PATH" default:"/etc/smtp/tls/tls.key" required:"true"`
			VersionMin     uint16             `envconfig:"API_SMTP_TLS_VERSION_MIN" default:"769" required:"true"`
			ClientAuthType tls.ClientAuthType `envconfig:"API_SMTP_TLS_CLIENT_AUTH_TYPE" default:"4" required:"true"`
		}
	}
	Group     string `envconfig:"API_GROUP" default:"default" required:"true"`
	EventType EventTypeConfig
	Writer    struct {
		Backoff   time.Duration `envconfig:"API_WRITER_BACKOFF" default:"10s" required:"true"`
		BatchSize uint32        `envconfig:"API_WRITER_BATCH_SIZE" default:"16" required:"true"`
		Cache     WriterCacheConfig
		Internal  WriterInternalConfig
		Uri       string `envconfig:"API_WRITER_URI" default:"resolver:50051" required:"true"`
	}
}

type WriterCacheConfig struct {
	Size uint32        `envconfig:"API_WRITER_CACHE_SIZE" default:"100" required:"true"`
	Ttl  time.Duration `envconfig:"API_WRITER_CACHE_TTL" default:"24h" required:"true"`
}

type WriterInternalConfig struct {
	Name               string `envconfig:"API_WRITER_INTERNAL_NAME" default:"awkinternal" required:"true"`
	Value              int32  `envconfig:"API_WRITER_INTERNAL_VALUE" required:"true"`
	RateLimitPerMinute int    `envconfig:"API_WRITER_INTERNAL_RATE_LIMIT_PER_MINUTE" default:"1" required:"true"`
}

type ReaderConfig struct {
	UriEventBase string `envconfig:"API_READER_URI_EVT_BASE" default:"https://awakari.com/pub-msg.html?id=" required:"true"`
}

type EventTypeConfig struct {
	Self string `envconfig:"API_EVENT_TYPE_SELF" required:"true" default:"com_awakari_email_v1"`
}

func NewConfigFromEnv() (cfg Config, err error) {
	err = envconfig.Process("", &cfg)
	return
}
