package mockdata

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	dbpkg "femProjectSqlc/internal/db"
	"femProjectSqlc/internal/tokens"
	"femProjectSqlc/internal/utils"
)

// SeedUser represents a user to be seeded
type SeedUser struct {
	Username string
	Email    string
	Password string
	Bio      string
	IsAdmin  bool
}

// SeedEvent represents an event to be seeded
type SeedEvent struct {
	OwnerIndex  int // index into users slice
	Title       string
	OwnerName   string
	Description string
	Location    string
	StartTime   time.Time
	ImageURL    string
	Price       float64
}

// SeedTicket represents a ticket to be seeded
type SeedTicket struct {
	UserIndex  int // index into users slice
	EventIndex int // index into events slice
	Name       string
	Status     string
}

// DefaultUsers returns a set of fake users
func DefaultUsers() []SeedUser {
	return []SeedUser{
		{
			Username: "admin",
			Email:    "admin@example.com",
			Password: "admin1234",
			Bio:      "Platform administrator",
			IsAdmin:  true,
		},
		{
			Username: "amarakoroma",
			Email:    "amara@example.com",
			Password: "password123",
			Bio:      "Music lover and event organizer from Freetown",
			IsAdmin:  false,
		},
		{
			Username: "fatimabangura",
			Email:    "fatima@example.com",
			Password: "password123",
			Bio:      "Tech enthusiast and conference goer based in Bo",
			IsAdmin:  false,
		},
		{
			Username: "ibrahimconteh",
			Email:    "ibrahim@example.com",
			Password: "password123",
			Bio:      "Sports fan and community organizer from Kenema",
			IsAdmin:  false,
		},
		{
			Username: "isatukamara",
			Email:    "isatu@example.com",
			Password: "password123",
			Bio:      "Art curator and cultural promoter from Makeni",
			IsAdmin:  false,
		},
	}
}

// DefaultEvents returns a set of fake events
func DefaultEvents() []SeedEvent {
	now := time.Now()
	return []SeedEvent{
		{
			OwnerIndex:  0,
			Title:       "Freetown Music Festival 2026",
			OwnerName:   "admin",
			Description: "A three-day outdoor music festival featuring Sierra Leone's finest artists alongside regional stars. Food stalls, cultural displays, and live drumming throughout.",
			Location:    "Lumley Beach, Freetown",
			StartTime:   now.Add(30 * 24 * time.Hour),
			ImageURL:    "https://images.unsplash.com/photo-1459749411175-04bf5292ceea?w=800",
			Price:       150.00,
		},
		{
			OwnerIndex:  0,
			Title:       "Afrobeats Night at Paddy's",
			OwnerName:   "admin",
			Description: "An electrifying evening of Afrobeats, palm wine music, and bubu sounds. Live band performance with dinner included.",
			Location:    "Paddy's Bar, Aberdeen, Freetown",
			StartTime:   now.Add(7 * 24 * time.Hour),
			ImageURL:    "https://images.unsplash.com/photo-1511192336575-5a79af67a629?w=800",
			Price:       75.00,
		},
		{
			OwnerIndex:  0,
			Title:       "SierraLeone TechSummit 2026",
			OwnerName:   "admin",
			Description: "Annual technology summit covering fintech, agritech, and digital innovation across West Africa. Keynote speakers from leading African tech companies.",
			Location:    "Bintumani Conference Centre, Freetown",
			StartTime:   now.Add(45 * 24 * time.Hour),
			ImageURL:    "https://images.unsplash.com/photo-1540575467063-178a50e2fd60?w=800",
			Price:       300.00,
		},
		{
			OwnerIndex:  0,
			Title:       "Startup Pitch Night Freetown",
			OwnerName:   "admin",
			Description: "Watch 10 promising Sierra Leonean startups pitch to a panel of local and diaspora investors. Network with founders afterwards.",
			Location:    "KITE Sierra Leone, Freetown",
			StartTime:   now.Add(14 * 24 * time.Hour),
			ImageURL:    "https://images.unsplash.com/photo-1475721027785-f74eccf877e2?w=800",
			Price:       25.00,
		},
		{
			OwnerIndex:  0,
			Title:       "Freetown City Run 2026",
			OwnerName:   "admin",
			Description: "Annual city run with 5K, 10K, and half-marathon categories through the scenic streets of Freetown. All fitness levels welcome.",
			Location:    "National Stadium, Freetown",
			StartTime:   now.Add(60 * 24 * time.Hour),
			ImageURL:    "https://images.unsplash.com/photo-1513593771513-7b58b6c4af38?w=800",
			Price:       50.00,
		},
		{
			OwnerIndex:  0,
			Title:       "Bo Football Tournament",
			OwnerName:   "admin",
			Description: "5-a-side football tournament open to all clubs in the Southern Province. Prizes for top teams. Food and entertainment all day.",
			Location:    "Bo Stadium, Bo",
			StartTime:   now.Add(21 * 24 * time.Hour),
			ImageURL:    "https://images.unsplash.com/photo-1546519638-68e109498ffc?w=800",
			Price:       35.00,
		},
		{
			OwnerIndex:  0,
			Title:       "Sierra Leone Cultural Arts Exhibition",
			OwnerName:   "admin",
			Description: "A curated showcase of contemporary and traditional Sierra Leonean art, crafts, and sculpture from emerging local artists. Opening night reception included.",
			Location:    "Cotton Tree Gallery, Freetown",
			StartTime:   now.Add(10 * 24 * time.Hour),
			ImageURL:    "https://images.unsplash.com/photo-1531243269054-5ebf6f34081e?w=800",
			Price:       45.00,
		},
		{
			OwnerIndex:  0,
			Title:       "Photography Masterclass",
			OwnerName:   "admin",
			Description: "Hands-on workshop covering portrait photography, natural lighting in tropical settings, and post-processing. Bring your own camera.",
			Location:    "Cockle Bay Arts Centre, Freetown",
			StartTime:   now.Add(5 * 24 * time.Hour),
			ImageURL:    "https://images.unsplash.com/photo-1471341971476-ae15ff5dd4ea?w=800",
			Price:       120.00,
		},
	}
}

// DefaultTickets returns a set of fake tickets
func DefaultTickets() []SeedTicket {
	return []SeedTicket{
		// User 1 (amarakoroma) buys tickets
		{UserIndex: 1, EventIndex: 2, Name: "Amara Koroma - TechSummit", Status: "active"},
		{UserIndex: 1, EventIndex: 6, Name: "Amara Koroma - Arts Exhibition", Status: "active"},

		// User 2 (fatimabangura) buys tickets
		{UserIndex: 2, EventIndex: 0, Name: "Fatima Bangura - Music Festival", Status: "active"},
		{UserIndex: 2, EventIndex: 5, Name: "Fatima Bangura - Football Tournament", Status: "active"},

		// User 3 (ibrahimconteh) buys tickets
		{UserIndex: 3, EventIndex: 0, Name: "Ibrahim Conteh - Music Festival", Status: "active"},
		{UserIndex: 3, EventIndex: 2, Name: "Ibrahim Conteh - TechSummit", Status: "active"},
		{UserIndex: 3, EventIndex: 6, Name: "Ibrahim Conteh - Arts Exhibition", Status: "used"},

		// User 4 (isatukamara) buys tickets
		{UserIndex: 4, EventIndex: 0, Name: "Isatu Kamara - Music Festival", Status: "active"},
		{UserIndex: 4, EventIndex: 1, Name: "Isatu Kamara - Afrobeats Night", Status: "active"},
		{UserIndex: 4, EventIndex: 4, Name: "Isatu Kamara - City Run", Status: "active"},

		// Admin buys a ticket too
		{UserIndex: 0, EventIndex: 2, Name: "Admin - TechSummit", Status: "active"},
	}
}

// Seed populates the database with mock data.
// It skips seeding if users already exist.
func Seed(database *sql.DB) error {
	ctx := context.Background()
	queries := dbpkg.New(database)

	// Check if data already exists
	count, err := queries.CountUsers(ctx)
	if err == nil && count > 0 {
		log.Printf("[seed] Database already has %d users, skipping seed\n", count)
		return nil
	}

	log.Println("[seed] Seeding database with mock data...")

	// Create ticket signer
	signer, err := utils.NewTicketSigner()
	if err != nil {
		key, _ := utils.GenerateSecretKey(32)
		signer = utils.NewTicketSignerFromKey(key)
	}

	users := DefaultUsers()
	events := DefaultEvents()
	tickets := DefaultTickets()

	// --- Seed users ---
	userIDs := make([]int64, len(users))
	for i, u := range users {
		hashedPw, err := tokens.HashPassword(u.Password)
		if err != nil {
			return fmt.Errorf("hash password for %s: %w", u.Username, err)
		}

		row, err := queries.CreateUser(ctx, dbpkg.CreateUserParams{
			Username: u.Username,
			Email:    u.Email,
			Password: hashedPw,
			Bio:      sql.NullString{String: u.Bio, Valid: u.Bio != ""},
		})
		if err != nil {
			return fmt.Errorf("create user %s: %w", u.Username, err)
		}
		userIDs[i] = row.ID
		log.Printf("[seed] Created user: %s (id=%d, admin=%v)\n", u.Username, row.ID, u.IsAdmin)

		// Set admin flag if needed
		if u.IsAdmin {
			_, err = database.ExecContext(ctx,
				"UPDATE users SET is_admin = 1 WHERE id = ?", row.ID)
			if err != nil {
				return fmt.Errorf("set admin for %s: %w", u.Username, err)
			}
		}
	}

	// --- Seed events ---
	eventIDs := make([]int64, len(events))
	for i, e := range events {
		row, err := queries.CreateEvent(ctx, dbpkg.CreateEventParams{
			UserID:      userIDs[e.OwnerIndex],
			Title:       e.Title,
			OwnerName:   e.OwnerName,
			Description: e.Description,
			Location:    e.Location,
			StartTime:   e.StartTime.UTC(),
			ImageUrl:    e.ImageURL,
			Price:       e.Price,
		})
		if err != nil {
			return fmt.Errorf("create event %s: %w", e.Title, err)
		}
		eventIDs[i] = row.ID
		log.Printf("[seed] Created event: %s (id=%d)\n", e.Title, row.ID)
	}

	// --- Seed tickets (price comes from the event) ---
	for _, t := range tickets {
		userID := userIDs[t.UserIndex]
		eventID := eventIDs[t.EventIndex]
		eventPrice := events[t.EventIndex].Price
		signature := signer.Sign(userID, eventID, t.Name)

		_, err := queries.CreateTicket(ctx, dbpkg.CreateTicketParams{
			UserID:    userID,
			EventID:   eventID,
			Name:      t.Name,
			Signature: signature,
			Price:     eventPrice,
			Status:    t.Status,
		})
		if err != nil {
			return fmt.Errorf("create ticket %s: %w", t.Name, err)
		}
		log.Printf("[seed] Created ticket: %s (user=%d, event=%d)\n", t.Name, userID, eventID)
	}

	log.Printf("[seed] Done! Created %d users, %d events, %d tickets\n",
		len(users), len(events), len(tickets))
	return nil
}
