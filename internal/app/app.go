package app

import (
	"context"
	"database/sql"
	"femProjectSqlc/internal/auth"
	"femProjectSqlc/internal/cache"
	dbpkg "femProjectSqlc/internal/db"
	"femProjectSqlc/internal/handlers"
	"femProjectSqlc/internal/logger"
	"femProjectSqlc/internal/middleware"
	"femProjectSqlc/internal/queue"
	"femProjectSqlc/internal/store"
	"femProjectSqlc/internal/utils"
)

type Application struct {
	Logger         *logger.Logger
	ctx            context.Context
	DB             *sql.DB
	Queries        *dbpkg.Queries
	UserStore      store.UserStoreInterface
	TokenStore     store.TokenStoreInterface
	EventStore     store.EventStoreInterface
	TicketStore    store.TicketStoreInterface
	AuthHandler    *auth.Handler
	UserHandler    *handlers.UserHandler
	EventHandler   *handlers.EventHandler
	TicketHandler  *handlers.TicketHandler
	AdminHandler   *handlers.AdminHandler
	PaymentHandler *handlers.PaymentHandler
	UploadHandler  *handlers.UploadHandler
	ImageQueue     *queue.ImageQueue
	AuthMiddleware *middleware.Middleware
}

func NewApplication(ctx context.Context, database *sql.DB, appCache *cache.InMemoryCache) *Application {
	// logger
	appLogger := logger.New()

	if appCache == nil {
		appLogger.Fatal("Failed to initialize InMemoryCache", nil)
	}

	// sqlc Queries
	queries := dbpkg.New(database)

	// stores
	userStore := store.NewUserStore(database, queries)
	if userStore == nil {
		appLogger.Fatal("Failed to initialize UserStore", nil)
	}
	tokenStore := store.NewTokenStore(database, queries)
	if tokenStore == nil {
		appLogger.Fatal("Failed to initialize TokenStore", nil)
	}

	eventStore := store.NewEventStore(database, queries)
	if eventStore == nil {
		appLogger.Fatal("Failed to initialize EventStore", nil)
	}
	ticketStore := store.NewTicketStore(database, queries)
	if ticketStore == nil {
		appLogger.Fatal("Failed to initialize TicketStore", nil)
	}
	transactionStore := store.NewTransactionStore(database, queries)
	if transactionStore == nil {
		appLogger.Fatal("Failed to initialize TransactionStore", nil)
	}

	// ticket signer
	ticketSigner, err := utils.NewTicketSigner()
	if err != nil {
		appLogger.Info("TICKET_SECRET_KEY not set, generating a temporary key (set env var for production)")
		key, _ := utils.GenerateSecretKey(64)
		ticketSigner = utils.NewTicketSignerFromKey(key)
	}

	// image upload queue
	imageQueue := queue.NewImageQueue(appCache)

	// handlers & middleware
	authHandler := auth.NewHandler(userStore, tokenStore)
	userHandler := handlers.NewUserHandler(userStore, eventStore, ticketStore, appLogger, appCache)
	eventHandler := handlers.NewEventHandler(eventStore, appLogger, appCache)
	ticketHandler := handlers.NewTicketHandler(ticketStore, eventStore, ticketSigner, appLogger, appCache)
	adminHandler := handlers.NewAdminHandler(userStore, eventStore, ticketStore, appLogger)
	paymentHandler := handlers.NewPaymentHandler(eventStore, ticketStore, transactionStore, ticketSigner, appLogger)
	uploadHandler := handlers.NewUploadHandler(imageQueue, appLogger)
	authMiddleware := middleware.New(tokenStore, appLogger)

	return &Application{
		ctx:            ctx,
		DB:             database,
		Queries:        queries,
		UserStore:      userStore,
		TokenStore:     tokenStore,
		EventStore:     eventStore,
		TicketStore:    ticketStore,
		Logger:         appLogger,
		AuthHandler:    authHandler,
		UserHandler:    userHandler,
		EventHandler:   eventHandler,
		TicketHandler:  ticketHandler,
		AdminHandler:   adminHandler,
		PaymentHandler: paymentHandler,
		UploadHandler:  uploadHandler,
		ImageQueue:     imageQueue,
		AuthMiddleware: authMiddleware,
	}
}
