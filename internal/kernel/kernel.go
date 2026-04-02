package kernel

import (
	"context"
	"errors"
	"fmt"
	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/config"
	"go-timekeeper/internal/cron"
	"go-timekeeper/internal/database"
	"go-timekeeper/internal/handler"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/repository"
	"go-timekeeper/internal/router"
	"go-timekeeper/internal/service"
	"go-timekeeper/internal/uow"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	cronlib "github.com/robfig/cron/v3"
)

// Kernel is the main application kernel. It contains all required dependencies and handles the application lifecycle.
type Kernel struct {
	Config       *config.Config
	Logger       *logger.Logger
	DBConnection *database.Database
	Router       *gin.Engine
	HTTPServer   *http.Server
	Cron         *cronlib.Cron
}

// New creates and initializes a new Kernel instance.
func New() (*Kernel, error) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	log := logger.New(cfg.Logger.Level, cfg.Logger.Format)
	log.Info("Starting application kernel")

	db, err := database.New(cfg.GetDatabaseDSN(), log.Logger)
	if err != nil {
		log.WithError(err).Error(logger.LogMessageFailedToInitializeDatabase)
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Repositories
	unitOfWork := uow.NewUnitOfWorkManager(db.DB)
	userRepo := repository.NewUserRepository(db.DB, log)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db.DB, log)
	projectRepo := repository.NewProjectRepository(db.DB, log)
	taskRepo := repository.NewTaskRepository(db.DB, log)
	timeRecordRepo := repository.NewTimeRecordRepository(db.DB, log)

	// Services
	tokenManager := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTLMinutes, cfg.Auth.RefreshTokenTTLHours)
	userService := service.NewUserService(userRepo, tokenManager, refreshTokenRepo)
	projectService := service.NewProjectService(projectRepo)
	timeRecordService := service.NewTimeRecordService(timeRecordRepo, unitOfWork)
	taskService := service.NewTaskService(taskRepo, timeRecordRepo, timeRecordService, unitOfWork)
	reportService := service.NewReportService(timeRecordRepo, taskRepo, projectRepo)

	// Handlers
	handlersPool := handler.NewHandlersPool(
		userService,
		projectService,
		taskService,
		reportService,
		timeRecordService,
		log,
	)

	// Gin router and HTTP server setup
	routerEngine := gin.Default()
	router.SetupRoutes(routerEngine, handlersPool, tokenManager, log)

	port := strconv.Itoa(cfg.Server.Port)
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: routerEngine,
	}

	c := cron.InitCrons(log, cfg, userService)

	kernel := &Kernel{
		Config:       cfg,
		Logger:       log,
		DBConnection: db,
		Router:       routerEngine,
		HTTPServer:   httpServer,
		Cron:         c,
	}

	log.Info("Kernel initialized successfully")
	return kernel, nil
}

// Start starts the application kernel and the HTTP server (in a goroutine).
func (k *Kernel) Start(ctx context.Context) error {
	k.Logger.Info("Starting kernel...")

	if err := k.DBConnection.HealthCheck(); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	k.Logger.Info("Database health check successful")

	go func() {
		k.Logger.Infof("🚀 Starting HTTP server on %s...", k.HTTPServer.Addr)
		if err := k.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			k.Logger.WithError(err).Error(logger.LogMessageServerFailed)
		}
	}()

	if k.Cron != nil {
		k.Cron.Start()
	}
	k.Logger.Info("Kernel started successfully")
	return nil
}

// Stop gracefully shuts down the HTTP server and closes the database connection.
func (k *Kernel) Stop(ctx context.Context) error {
	k.Logger.Info("Stopping kernel...")
	if k.Cron != nil {
		k.Cron.Stop()
	}

	// Graceful server shutdown with timeout
	ctxShutdown, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := k.HTTPServer.Shutdown(ctxShutdown); err != nil {
		k.Logger.WithError(err).Error(logger.LogMessageGracefulServerShutdownFailed)
	} else {
		k.Logger.Info("HTTP server stopped gracefully")
	}

	if err := k.DBConnection.Close(); err != nil {
		k.Logger.WithError(err).Error(logger.LogMessageFailedToCloseDatabaseConnection)
	} else {
		k.Logger.Info("Database connection closed")
	}

	k.Logger.Info("Kernel stopped successfully")
	return nil
}

// WaitForShutdown waits for a SIGINT or SIGTERM signal and then returns.
func (k *Kernel) WaitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	k.Logger.WithField("signal", sig.String()).Info("Received shutdown signal")
}

// Run starts the kernel, waits for a shutdown signal, then stops gracefully.
func (k *Kernel) Run() error {
	ctx := context.Background()

	if err := k.Start(ctx); err != nil {
		return err
	}

	k.WaitForShutdown()

	return k.Stop(ctx)
}
