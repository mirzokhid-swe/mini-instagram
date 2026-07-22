// Package app configures and runs application.
package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"mini-instagram/config"
	"mini-instagram/internal/controller/restapi"
	"mini-instagram/internal/repo/persistent"
	"mini-instagram/internal/usecase"
	authusecase "mini-instagram/internal/usecase/auth"
	"mini-instagram/pkg/httpserver"
	jwtmanager "mini-instagram/pkg/jwt"
	"mini-instagram/pkg/logger"
	"mini-instagram/pkg/postgres"
	"mini-instagram/pkg/redis"
	"mini-instagram/pkg/storage"
)

type useCases struct {
	auth usecase.Auth
}

type servers struct {
	http *httpserver.Server
}

func initUseCases(pg *postgres.Postgres, cfg *config.Config, l logger.Interface) useCases {
	userRepo := persistent.NewUserRepo(pg)
	return useCases{
		auth: authusecase.New(userRepo, jwtmanager.New(cfg.JWT.Secret), l),
	}
}

func initServers(cfg *config.Config, uc useCases, l logger.Interface, st *storage.Storage, redisClient *redis.Client) servers {
	gin.SetMode(gin.ReleaseMode)
	handler := gin.New()

	restapi.NewRouter(handler, uc.auth, l, st, redisClient)

	httpServer := httpserver.New(handler, httpserver.Port(cfg.HTTP.Port))

	return servers{
		http: httpServer,
	}
}

func (s *servers) startServers(l logger.Interface) {
	s.http.Start()

	l.Info("server started")
}

func (s *servers) waitForShutdown(l logger.Interface) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	var err error

	select {
	case sig := <-interrupt:
		l.Info("signal received", "signal", sig.String())
	case err = <-s.http.Notify():
		l.Error("http server error", "error", err)
	}

	s.shutdownServers(l)
}

func (s *servers) shutdownServers(l logger.Interface) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.http.Shutdown(shutdownCtx); err != nil {
		l.Error("http server shutdown failed", "error", err)
	}
}

// Run creates objects via constructors and starts the application.
func Run(cfg *config.Config) {
	l := logger.New(cfg.Log.Level)

	pg, err := postgres.New(cfg.Postgres.URL, postgres.MaxPoolSize(int32(cfg.Postgres.PoolMax)))
	if err != nil {
		l.Error("postgres init failed", "error", err)
		os.Exit(1)
	}
	defer pg.Close()

	st := storage.New(cfg.Media.Path)

	redisClient, err := redis.New(context.Background(), cfg.Redis.URL)
	if err != nil {
		l.Error("redis init failed; rate limiting disabled", "error", err)
	} else {
		defer redisClient.Close()
	}

	uc := initUseCases(pg, cfg, l)
	s := initServers(cfg, uc, l, st, redisClient)
	s.startServers(l)
	s.waitForShutdown(l)
}
