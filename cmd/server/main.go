package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"femProjectSqlc/internal/app"
	"femProjectSqlc/internal/cache"
	"femProjectSqlc/internal/database"
	"femProjectSqlc/internal/mockdata"
	"femProjectSqlc/internal/routes"
	"femProjectSqlc/internal/worker"

	"github.com/joho/godotenv"
)

func main() {

	seedFlag := flag.Bool("seed", false, "Seed database")
	flag.Parse()

	// ✅ Only load .env locally
	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		_ = godotenv.Load()
	}

	ctx := context.Background()

	// Initialize DB
	db, err := database.NewAppDB()
	if err != nil {
		log.Fatal("Database Error:", err)
	}

	// ✅ OPTIONAL SEED ONLY (not auto-run)
	if *seedFlag {
		if err := mockdata.Seed(db.DB); err != nil {
			log.Fatalf("Seed failed: %v", err)
		}
		log.Println("Seed complete!")
	}

	// In-memory cache (no external Redis required)
	appCache := cache.NewInMemoryCache()
	defer appCache.Close()

	defer db.DB.Close()

	application := app.NewApplication(ctx, db.DB, appCache)

	workerCtx, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()

	imageWorker := worker.NewImageWorker(application.ImageQueue, application.Logger)
	go imageWorker.Run(workerCtx)

	router := routes.SetupRouter(application)

	// ✅ FIX: USE RAILWAY PORT
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      router,
	}

	application.Logger.Info("Starting server on port " + port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			application.Logger.Fatal("Server Error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	application.Logger.Info("Shutting down server...")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctxShutdown); err != nil {
		application.Logger.Fatal("Shutdown Error", err)
	}

	application.Logger.Info("Server exited properly")
}
