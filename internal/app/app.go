// Package app configures and runs application.
package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"todo/config"
	"todo/internal/controller/restapi"
	"todo/pkg/httpserver"
	"todo/pkg/logger"
	"todo/pkg/postgres"
)

type useCases struct {
}

type servers struct {
	http *httpserver.Server
}

func initUseCases(_ *postgres.Postgres) useCases {
	return useCases{}
}

func initServers(cfg *config.Config, uc useCases, l logger.Interface) servers {
	gin.SetMode(gin.ReleaseMode)
	handler := gin.New()

	restapi.NewRouter(handler)

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

	uc := initUseCases(pg)
	s := initServers(cfg, uc, l)
	s.startServers(l)
	s.waitForShutdown(l)
}
