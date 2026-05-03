package handlers

import (
	"database/sql"
	"femProjectSqlc/internal/db"
	"femProjectSqlc/internal/logger"
	"femProjectSqlc/internal/middleware"
	"femProjectSqlc/internal/store"
	"femProjectSqlc/internal/utils"
	"math/rand"
	"net/http"
	"time"
)

type PaymentHandler struct {
	eventStore       store.EventStoreInterface
	ticketStore      store.TicketStoreInterface
	transactionStore store.TransactionStoreInterface
	signer           *utils.TicketSigner
	logger           *logger.Logger
}

func NewPaymentHandler(
	eventStore store.EventStoreInterface,
	ticketStore store.TicketStoreInterface,
	transactionStore store.TransactionStoreInterface,
	signer *utils.TicketSigner,
	log *logger.Logger,
) *PaymentHandler {
	return &PaymentHandler{
		eventStore:       eventStore,
		ticketStore:      ticketStore,
		transactionStore: transactionStore,
		signer:           signer,
		logger:           log,
	}
}

// ProcessPaymentRequest — only card details + event_id needed.
// Name comes from auth user, price comes from event.
type ProcessPaymentRequest struct {
	EventID    int64  `json:"event_id"`
	CardNumber string `json:"card_number"`
	ExpiryDate string `json:"expiry_date"`
	CVV        string `json:"cvv"`
	HolderName string `json:"holder_name"`
}

type PaymentResponse struct {
	TransactionID string     `json:"transaction_id"`
	Status        string     `json:"status"`
	Amount        float64    `json:"amount"`
	Last4         string     `json:"last4"`
	ProcessedAt   string     `json:"processed_at"`
	Ticket        *db.Ticket `json:"ticket"`
}

// ProcessPayment simulates payment, creates ticket, and saves transaction record
func (h *PaymentHandler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ProcessPaymentRequest
	if err := utils.DecodeJSON(r.Body, &req); err != nil {
		h.logger.Error("Failed to decode payment request", err)
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.EventID == 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "event_id is required")
		return
	}
	if req.CardNumber == "" || req.CVV == "" || req.ExpiryDate == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "card_number, cvv, and expiry_date are required")
		return
	}

	// Get event to determine price
	event, err := h.eventStore.GetEvent(r.Context(), req.EventID)
	if err != nil {
		h.logger.Error("Event not found", err)
		utils.RespondWithError(w, http.StatusNotFound, "event not found")
		return
	}

	if event.Price <= 0 {
		h.logger.Error("Event has no price set", err)
		utils.RespondWithError(w, http.StatusBadRequest, "this event has no price set")
		return
	}

	// Simulate processing delay
	time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)

	// Fake last4 digits
	last4 := req.CardNumber
	if len(last4) >= 4 {
		last4 = last4[len(last4)-4:]
	}

	txnID := generateTransactionID()

	// Ticket name from the logged-in user's username
	ticketName := authUser.Username + " - " + event.Title
	signature := h.signer.Sign(authUser.ID, event.ID, ticketName)

	// Create ticket with price from event
	ticket, err := h.ticketStore.CreateTicket(r.Context(), db.CreateTicketParams{
		UserID:    authUser.ID,
		EventID:   event.ID,
		Name:      ticketName,
		Signature: signature,
		Price:     event.Price,
		Status:    "active",
	})
	if err != nil {
		h.logger.Error("Failed to create ticket", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to create ticket")
		return
	}

	// Save transaction record
	_, err = h.transactionStore.CreateTransaction(r.Context(), db.CreateTransactionParams{
		UserID:        authUser.ID,
		EventID:       event.ID,
		TicketID:      sql.NullInt64{Int64: ticket.ID, Valid: true},
		TransactionID: txnID,
		Amount:        event.Price,
		CardLast4:     last4,
		Status:        "approved",
	})
	if err != nil {
		h.logger.Error("Failed to save transaction record", err)
		// Ticket already created — don't fail the user, just log
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"payment": PaymentResponse{
			TransactionID: txnID,
			Status:        "approved",
			Amount:        event.Price,
			Last4:         last4,
			ProcessedAt:   time.Now().UTC().Format(time.RFC3339),
			Ticket:        ticket,
		},
	})
}

// GetUserTransactions returns payment history for the authenticated user
func (h *PaymentHandler) GetUserTransactions(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	transactions, err := h.transactionStore.GetUserTransactions(r.Context(), authUser.ID)
	if err != nil {
		h.logger.Error("Failed to get transactions", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get transactions")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": transactions,
	})
}

func generateTransactionID() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return "TXN-" + string(b)
}
