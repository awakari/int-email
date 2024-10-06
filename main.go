package main

import (
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

    // init config and logger
    cfg, err := config.NewConfigFromEnv()
    if err != nil {
        panic(fmt.Sprintf("failed to load the config from env: %s", err))
    }
    //
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
    srv.AllowInsecureAuth = true
    srv.Domain = cfg.Api.Smtp.Host
    srv.MaxMessageBytes = int64(cfg.Api.Smtp.Data.Limit)
    srv.MaxRecipients = int(cfg.Api.Smtp.Recipients.Limit)
    srv.ReadTimeout = cfg.Api.Smtp.Timeout.Read
    srv.WriteTimeout = cfg.Api.Smtp.Timeout.Write
    log.Info("starting to listen for emails...")
    if err = srv.ListenAndServe(); err != nil {
        panic(err)
    }
}
