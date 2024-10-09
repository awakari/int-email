package main

import (
	"crypto/tls"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	apiSmtp "github.com/awakari/int-email/api/smtp"
	"github.com/awakari/int-email/config"
	"github.com/awakari/int-email/service/writer"
	"github.com/emersion/go-smtp"
	"log/slog"
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

	rcpts := map[string]bool{}
	for _, name := range cfg.Api.Smtp.Recipients.Names {
		rcpt := fmt.Sprintf("%s@%s", name, cfg.Api.Smtp.Host)
		rcpts[rcpt] = true
	}
	b := apiSmtp.NewBackend(svcWriter, rcpts, int64(cfg.Api.Smtp.Data.Limit))
	b = apiSmtp.NewBackendLogging(b, log)
	srv := smtp.NewServer(b)
	srv.Addr = fmt.Sprintf(":%d", cfg.Api.Smtp.Port)
	srv.Domain = cfg.Api.Smtp.Host
	srv.MaxMessageBytes = int64(cfg.Api.Smtp.Data.Limit)
	srv.MaxRecipients = int(cfg.Api.Smtp.Recipients.Limit)
	srv.ReadTimeout = cfg.Api.Smtp.Timeout.Read
	srv.WriteTimeout = cfg.Api.Smtp.Timeout.Write
	srv.AllowInsecureAuth = false

	// Load the TLS certificate and key from the mounted volume
	var cert tls.Certificate
	cert, err = tls.LoadX509KeyPair("/etc/smtp/tls/tls.crt", "/etc/smtp/tls/tls.key")
	if err != nil {
		panic(err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			cert,
		},
		Renegotiation: tls.RenegotiateNever,
		ClientAuth:    tls.RequireAndVerifyClientCert,
		MinVersion:    tls.VersionTLS12,
	}
	l, err := tls.Listen("tcp", srv.Addr, tlsConfig)
	if err != nil {
		panic(err)
	}

	log.Info("starting to listen for emails...")
	if err = srv.Serve(l); err != nil {
		panic(err)
	}
}
