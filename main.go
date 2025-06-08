package main

import (
	"crypto/tls"
	"fmt"
	"github.com/awakari/int-email/api/http/pub"
	apiSmtp "github.com/awakari/int-email/api/smtp"
	"github.com/awakari/int-email/config"
	"github.com/awakari/int-email/service"
	"github.com/awakari/int-email/service/converter"
	"github.com/awakari/int-email/util"
	"github.com/emersion/go-smtp"
	"log/slog"
	"net/http"
	"os"
)

func main() {

	// init config
	cfg, err := config.NewConfigFromEnv()
	if err != nil {
		panic(fmt.Sprintf("failed to load the config from env: %s", err))
	}

	// logger
	opts := slog.HandlerOptions{
		Level: slog.Level(cfg.Log.Level),
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &opts))
	log.Info("starting...")

	svcPub := pub.NewService(http.DefaultClient, cfg.Api.Writer.Uri, cfg.Api.Token.Internal)
	svcPub = pub.NewLogging(svcPub, log)

	rcptsPublish := map[string]bool{}
	for _, name := range cfg.Api.Smtp.Recipients.Publish {
		rcptsPublish[name] = true
	}
	svcConv := converter.NewConverter(cfg.Api.EventType.Self, util.HtmlPolicy(), cfg.Api.Writer.Internal, rcptsPublish, cfg.Api.Smtp.Data.TruncUrlQueries)
	svcConv = converter.NewLogging(svcConv, log)
	svc := service.NewService(svcConv, svcPub, cfg.Api.Group, cfg.Api.Writer.Backoff)
	svc = service.NewLogging(svc, log)

	rcptsInternal := map[string]bool{}
	for _, name := range cfg.Api.Smtp.Recipients.Internal {
		rcptsInternal[name] = true
	}
	b := apiSmtp.NewBackend(rcptsPublish, rcptsInternal, int64(cfg.Api.Smtp.Data.Limit), svc)
	b = apiSmtp.NewBackendLogging(b, log)

	srv := smtp.NewServer(b)
	srv.Addr = fmt.Sprintf(":%d", cfg.Api.Smtp.Port)
	srv.Domain = cfg.Api.Smtp.Host
	srv.MaxMessageBytes = int64(cfg.Api.Smtp.Data.Limit)
	srv.MaxRecipients = int(cfg.Api.Smtp.Recipients.Limit)
	srv.ReadTimeout = cfg.Api.Smtp.Timeout.Read
	srv.WriteTimeout = cfg.Api.Smtp.Timeout.Write
	srv.AllowInsecureAuth = false
	srv.EnableREQUIRETLS = true
	// Load the TLS certificate and key from the mounted volume
	var cert tls.Certificate
	cert, err = tls.LoadX509KeyPair(cfg.Api.Smtp.Tls.CertPath, cfg.Api.Smtp.Tls.KeyPath)
	if err != nil {
		panic(err)
	}
	srv.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{
			cert,
		},
		ClientAuth: cfg.Api.Smtp.Tls.ClientAuthType,
		MinVersion: cfg.Api.Smtp.Tls.VersionMin,
	}

	log.Info("starting to listen for emails...")
	if err = srv.ListenAndServe(); err != nil {
		panic(err)
	}
}
