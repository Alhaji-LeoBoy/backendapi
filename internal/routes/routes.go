package routes

import (
	"femProjectSqlc/internal/app"
	"femProjectSqlc/internal/handlers"
	"femProjectSqlc/internal/middleware"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func SetupRouter(app *app.Application) *chi.Mux {
	r := chi.NewRouter()

	// Global limiter: 10 req/s sustained, bursts of up to 20.
	// Use stricter limits on sensitive auth routes below.
	globalLimiter := middleware.NewRateLimiter(middleware.DefaultRateLimiterConfig())

	// Tighter limiter for auth endpoints to slow credential-stuffing attacks.
	authLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		RequestsPerSecond: 3,
		Burst:             5,
		CleanupInterval:   5 * time.Minute,
		ClientTTL:         10 * time.Minute,
		TrustProxy:        false,
	})
	// allordOrigns :=
	// Global middleware
	r.Use(middleware.CORS("localhost:8080"))
	r.Use(middleware.SecureHeaders)
	r.Use(globalLimiter.Limit)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)

	// Public routes
	r.Get("/health", handlers.HealthCheck)

	// Auth routes - public endpoints
	r.Route("/auth", func(r chi.Router) {
		r.Use(authLimiter.Limit)
		r.Post("/register", app.AuthHandler.Register)
		r.Post("/login", app.AuthHandler.Login)
		r.Post("/refresh", app.AuthHandler.Refresh)
		// Protected auth endpoints use With() for per-route middleware
		r.With(app.AuthMiddleware.ChiAuthenticate).Post("/logout", app.AuthHandler.Logout)
	})

	// Protected API routes
	r.Route("/users", func(r chi.Router) {
		r.Use(app.AuthMiddleware.ChiAuthenticate)
		r.Get("/", app.UserHandler.ListUsers)
		r.Get("/search", app.UserHandler.SearchUsers)
		r.Get("/me", app.UserHandler.UserProfile)
		r.Get("/{id}", app.UserHandler.GetUserByID)
		r.Get("/me/events", app.UserHandler.GetUserEvents)
		r.Get("/me/events/summary", app.UserHandler.GetUserEventsSummary)
		r.Get("/me/tickets", app.UserHandler.GetUserTickets)
		r.Patch("/me", app.UserHandler.PatchUser)
		r.Patch("/me/password", app.UserHandler.PatchPassword)
		r.Patch("/me/email", app.UserHandler.PatchEmail)
		r.Delete("/me", app.UserHandler.DeleteUser)
	})

	// Event routes (public reads, protected writes)
	r.Route("/events", func(r chi.Router) {
		r.Get("/", app.EventHandler.ListEvents)
		r.Get("/search", app.EventHandler.SearchEvents)
		r.Get("/stats", app.EventHandler.ListEventsWithStats)
		r.Get("/{id}", app.EventHandler.GetEvent)
		r.Get("/{id}/tickets", app.TicketHandler.GetEventTickets)

		r.Group(func(r chi.Router) {
			r.Use(app.AuthMiddleware.ChiAuthenticate)
			r.Post("/", app.EventHandler.CreateEvent)
			r.Patch("/{id}", app.EventHandler.PatchEvent)
			r.Delete("/{id}", app.EventHandler.DeleteEvent)
		})
	})

	// Ticket routes
	r.Route("/tickets", func(r chi.Router) {
		// Public: verify a ticket's authenticity
		r.Get("/{id}/verify", app.TicketHandler.VerifyTicket)

		// Protected
		r.Group(func(r chi.Router) {
			r.Use(app.AuthMiddleware.ChiAuthenticate)
			r.Post("/", app.TicketHandler.CreateTicket)
			r.Get("/", app.TicketHandler.ListTickets)
			r.Get("/{id}", app.TicketHandler.GetTicket)
			r.Patch("/{id}", app.TicketHandler.PatchTicket)
			r.Post("/{id}/mark-used", app.TicketHandler.MarkTicketUsed)
			r.Delete("/{id}", app.TicketHandler.DeleteTicket)
		})
	})

	// Payment route (fake/sandbox)
	r.Route("/payments", func(r chi.Router) {
		r.Use(app.AuthMiddleware.ChiAuthenticate)
		r.Post("/process", app.PaymentHandler.ProcessPayment)
		r.Get("/history", app.PaymentHandler.GetUserTransactions)
	})

	// Upload routes — protected; serves uploaded images as static files
	r.Route("/uploads", func(r chi.Router) {
		// POST /uploads/image  — authenticated upload
		r.With(app.AuthMiddleware.ChiAuthenticate).Post("/image", app.UploadHandler.UploadImage)

		// GET /uploads/images/<filename>  — public static file server
		r.Handle("/images/*", http.StripPrefix("/uploads/images/",
			http.FileServer(http.Dir(handlers.UploadDir))))
	})

	// Admin routes - require authentication + admin role
	r.Route("/admin", func(r chi.Router) {
		r.Use(app.AuthMiddleware.ChiAuthenticate)
		r.Use(app.AuthMiddleware.ChiRequireAdmin)

		r.Post("/users", app.AdminHandler.CreateUser)
		r.Patch("/users/{id}", app.AdminHandler.PatchUser)
		r.Patch("/users/{id}/role", app.AdminHandler.SetUserAdmin)
		r.Delete("/users/{id}", app.AdminHandler.DeleteUser)
		r.Delete("/events/{id}", app.AdminHandler.DeleteEvent)
		r.Delete("/tickets/{id}", app.AdminHandler.DeleteTicket)
	})

	return r
}
