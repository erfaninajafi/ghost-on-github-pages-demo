package main

import (
	"context"
	"errors"
	delivery "expertsacademy/blog/internal/delivery/http"
	"expertsacademy/blog/internal/domain"
	"expertsacademy/blog/internal/platform"
	"expertsacademy/blog/internal/repository"
	"expertsacademy/blog/internal/usecase"
	"expertsacademy/blog/pkg/mailer"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if e := run(log); e != nil {
		log.Error("fatal", "error", e)
		os.Exit(1)
	}
}
func run(log *slog.Logger) error {
	cfg, e := platform.Load()
	if e != nil {
		return e
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	pc, e := pgxpool.ParseConfig(cfg.DatabaseURL)
	if e != nil {
		return e
	}
	pc.MaxConns = 20
	pc.MinConns = 2
	pc.ConnConfig.ConnectTimeout = 5 * time.Second
	pool, e := pgxpool.NewWithConfig(ctx, pc)
	if e != nil {
		return e
	}
	defer pool.Close()
	if e = pool.Ping(ctx); e != nil {
		return e
	}
	options, e := redis.ParseURL(cfg.RedisURL)
	if e != nil {
		return e
	}
	options.DialTimeout = time.Second
	options.ReadTimeout = time.Second
	options.WriteTimeout = time.Second
	cache := platform.Redis{Client: redis.NewClient(options)}
	defer cache.Client.Close()
	svc := &usecase.Service{Repo: repository.New(pool), Cache: cache, Invite: cfg.Invite, SiteURL: cfg.SiteURL, Log: log}
	var mail domain.Mailer = mailer.Mock{Log: log}
	if cfg.MailDriver == "smtp" {
		mail = mailer.SMTP{Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.SMTPFrom}
	}
	workerDone := make(chan struct{})
	go func() { defer close(workerDone); svc.DeliverEmails(ctx, mail) }()
	handler := &delivery.Server{Service: svc, Redis: cache, Config: cfg, Log: log, Ready: func(ctx context.Context) error {
		if e := pool.Ping(ctx); e != nil {
			return e
		}
		return cache.Client.Ping(ctx).Err()
	}}
	server := &http.Server{Addr: cfg.Address, Handler: handler.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 35 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 16}
	done := make(chan error, 1)
	go func() { log.Info("api listening", "address", cfg.Address); done <- server.ListenAndServe() }()
	select {
	case e = <-done:
		stop()
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		return err
	}
	<-workerDone
	if errors.Is(e, http.ErrServerClosed) {
		return nil
	}
	return e
}
