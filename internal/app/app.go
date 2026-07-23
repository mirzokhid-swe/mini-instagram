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
	likecache "mini-instagram/internal/cache/like"
	"mini-instagram/internal/controller/restapi"
	commentrepo "mini-instagram/internal/repo/persistent/comment"
	hashtagrepo "mini-instagram/internal/repo/persistent/hashtag"
	notificationrepo "mini-instagram/internal/repo/persistent/notification"
	postrepo "mini-instagram/internal/repo/persistent/post"
	userrepo "mini-instagram/internal/repo/persistent/user"
	"mini-instagram/internal/usecase"
	authusecase "mini-instagram/internal/usecase/auth"
	commentusecase "mini-instagram/internal/usecase/comment"
	notificationusecase "mini-instagram/internal/usecase/notification"
	postusecase "mini-instagram/internal/usecase/post"
	userusecase "mini-instagram/internal/usecase/user"
	"mini-instagram/internal/worker/likesync"
	"mini-instagram/pkg/httpserver"
	jwtmanager "mini-instagram/pkg/jwt"
	"mini-instagram/pkg/logger"
	"mini-instagram/pkg/postgres"
	"mini-instagram/pkg/redis"
	"mini-instagram/pkg/storage"
)

type useCases struct {
	auth          usecase.Auth
	posts         usecase.Post
	comments      usecase.Comment
	users         usecase.User
	notifications usecase.Notification
}

type servers struct {
	http *httpserver.Server
}

func initUseCases(pg *postgres.Postgres, cfg *config.Config, l logger.Interface, st *storage.Storage, likeCache *likecache.Cache) (useCases, *postrepo.PostRepo) {
	userRepo := userrepo.NewUserRepo(pg)
	postRepo := postrepo.NewPostRepo(pg)
	commentRepo := commentrepo.NewCommentRepo(pg)
	notificationRepo := notificationrepo.NewNotificationRepo(pg)
	hashtagRepo := hashtagrepo.NewHashtagRepo(pg)

	// A nil *likecache.Cache must become a nil postusecase.LikeCache (not a
	// non-nil interface wrapping a nil pointer), so the usecase's nil check
	// for "cache unavailable" works correctly.
	var cache postusecase.LikeCache
	if likeCache != nil {
		cache = likeCache
	}

	return useCases{
		auth:          authusecase.New(userRepo, jwtmanager.New(cfg.JWT.Secret), l),
		posts:         postusecase.New(postRepo, hashtagRepo, cache, st, l),
		comments:      commentusecase.New(commentRepo),
		users:         userusecase.New(userRepo, postRepo, st, l),
		notifications: notificationusecase.New(notificationRepo),
	}, postRepo
}

func initServers(cfg *config.Config, uc useCases, l logger.Interface, st *storage.Storage, redisClient *redis.Client, tokens *jwtmanager.TokenManager) servers {
	gin.SetMode(gin.ReleaseMode)
	handler := gin.New()

	restapi.NewRouter(handler, uc.auth, uc.posts, uc.comments, uc.users, uc.notifications, tokens, l, st, redisClient)

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
		l.Error("redis init failed; rate limiting and like cache disabled", "error", err)
	} else {
		defer redisClient.Close()
	}

	var likeCache *likecache.Cache
	if redisClient != nil {
		likeCache = likecache.New(redisClient)
	}

	uc, postRepo := initUseCases(pg, cfg, l, st, likeCache)
	tokens := jwtmanager.New(cfg.JWT.Secret)
	s := initServers(cfg, uc, l, st, redisClient, tokens)
	s.startServers(l)

	workerCtx, cancelWorker := context.WithCancel(context.Background())
	if likeCache != nil {
		worker := likesync.New(likeCache, postRepo, l)
		go worker.Run(workerCtx)
	}

	s.waitForShutdown(l)
	cancelWorker()
}
