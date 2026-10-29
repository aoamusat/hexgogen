package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	v1 "github.com/aoamusat/hexgogen/api/v1"
	"github.com/aoamusat/hexgogen/configs"
	domainservices "github.com/aoamusat/hexgogen/domain/services"
	"github.com/aoamusat/hexgogen/internal/adapter/db"
	"github.com/aoamusat/hexgogen/internal/adapter/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Start(ctx context.Context, cfg configs.Config, log domainservices.Logger) error {
	router := initializeRouter(cfg, log)
	dbConn, err := db.Init(cfg.DbURL)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	sqlDB, err := dbConn.DB()
	if err != nil {
		return fmt.Errorf("get database connection: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Error("Close database connection", err)
		}
	}()
	h := v1.Handler{UserService: services.NewUserService(cfg, db.NewUsersDB(dbConn))}
	RegisterRoutes(router, &h)
	server := &http.Server{
		Handler:           router,
		Addr:              ":" + cfg.Port,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Info("Starting HTTP server on " + server.Addr)
	return serve(ctx, server, log)
}

type httpServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
	Close() error
}

func serve(ctx context.Context, server httpServer, log domainservices.Logger) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()
	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		log.Info("Shutting down HTTP server")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return errors.Join(fmt.Errorf("shut down HTTP server: %w", err), server.Close())
	}
	if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	log.Info("HTTP server stopped")
	return nil
}

func initializeRouter(cfg configs.Config, log domainservices.Logger) *gin.Engine {
	if cfg.Env == configs.PRODUCTION {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()
		log.Info(fmt.Sprintf("%s %s status=%d latency=%s", ctx.Request.Method,
			ctx.Request.URL.Path, ctx.Writer.Status(), time.Since(start)))
	})
	router.Use(gin.CustomRecoveryWithWriter(nil, func(ctx *gin.Context, recovered any) {
		log.Error("HTTP handler panic", fmt.Errorf("%v\n%s", recovered, debug.Stack()))
		ctx.AbortWithStatus(http.StatusInternalServerError)
	}))
	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"POST", "GET", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"*"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}))
	return router
}

func RegisterRoutes(router *gin.Engine, v1Handler *v1.Handler) {
	apiRouter := router.Group("/v1")
	apiRouter.GET("/users", v1Handler.GetUser())
}
