package main

import (
	"crypto/tls"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	apiSmtp "github.com/awakari/int-email/api/smtp"
	"github.com/awakari/int-email/config"
	"github.com/awakari/int-email/service"
	"github.com/awakari/int-email/service/converter"
	"github.com/awakari/int-email/service/writer"
	"github.com/awakari/int-email/util"
	"github.com/emersion/go-smtp"
	"log/slog"
	"os"
	"strings"
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
	log.Info("starting the update for the feeds")

	// awakari API client
	var clientAwk api.Client
	clientAwk, err = api.
		NewClientBuilder().
		WriterUri(cfg.Api.Writer.Uri).
		SubscriptionsUri(cfg.Api.Interests.Uri).
		Build()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize the Awakari API client: %s", err))
	}
	defer clientAwk.Close()
	log.Info("initialized the Awakari API client")

	svcWriter := writer.NewService(clientAwk, cfg.Api.Writer.Backoff, cfg.Api.Writer.Cache, log)
	svcWriter = writer.NewLogging(svcWriter, log)

	rcptsPublish := map[string]bool{}
	for _, name := range cfg.Api.Smtp.Recipients.Publish {
		rcptsPublish[strings.ToLower(name)] = true
	}
	svcConv := converter.NewConverter(cfg.Api.EventType.Self, util.HtmlPolicy(), cfg.Api.Writer.Internal, rcptsPublish)
	svcConv = converter.NewLogging(svcConv, log)
	svc := service.NewService(svcConv, svcWriter, cfg.Api.Group)
	svc = service.NewLogging(svc, log)

	rcptsInternal := map[string]bool{}
	for _, name := range cfg.Api.Smtp.Recipients.Internal {
		rcptsInternal[strings.ToLower(name)] = true
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
