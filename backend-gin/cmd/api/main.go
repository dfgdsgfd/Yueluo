package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"yuem-go/backend-gin/internal/config"
	httpserver "yuem-go/backend-gin/internal/http/server"
	"yuem-go/backend-gin/internal/services"
	"yuem-go/backend-gin/internal/storage"
)

func main() {
	resetAdminPassword := flag.Bool("reset-admin-password", false, "reset an admin password and exit")
	resetAdminUsername := flag.String("reset-admin-username", storage.DefaultAdminUsername, "admin username to reset")
	resetAdminNewPassword := flag.String("reset-admin-new-password", storage.DefaultAdminInitialPassword, "new admin password for --reset-admin-password")
	flag.Parse()

	cfg := config.Load()
	logger, cleanupLogger, err := services.NewLogger(cfg.Log)
	if err != nil {
		if logger == nil {
			logger, err = zap.NewProduction()
			if err != nil {
				panic(err)
			}
			cleanupLogger = func() { _ = logger.Sync() }
		}
		logger.Warn("file logger unavailable; continuing with console logger", zap.Error(err))
	}
	zap.ReplaceGlobals(logger)
	defer cleanupLogger()

	if *resetAdminPassword {
		resetAdminPasswordAndExit(cfg, logger, *resetAdminUsername, *resetAdminNewPassword)
		return
	}
	router, cleanup, err := httpserver.NewRouterWithShutdown(cfg, logger)
	if err != nil {
		logger.Fatal("failed to create router", zap.Error(err))
	}
	defer cleanup()

	srv := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           router,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		logger.Info("backend-gin listening", zap.String("addr", srv.Addr), zap.String("mode", "native-gin"))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}
}

func resetAdminPasswordAndExit(cfg config.Config, logger *zap.Logger, username string, password string) {
	dbConfig := cfg.Database
	dbConfig.AutoMigrate = false
	db, err := storage.OpenDatabase(dbConfig)
	if err != nil {
		logger.Fatal("failed to open database for admin password reset", zap.Error(err))
	}
	if db == nil {
		logger.Fatal("database is required for admin password reset")
	}
	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	affected, err := storage.ResetAdminPassword(ctx, db, username, password)
	if err != nil {
		logger.Fatal("failed to reset admin password", zap.String("username", username), zap.Error(err))
	}
	if affected == 0 {
		logger.Fatal("admin account not found", zap.String("username", username))
	}
	logger.Info("admin password reset complete", zap.String("username", username), zap.Int64("affected", affected))
}
