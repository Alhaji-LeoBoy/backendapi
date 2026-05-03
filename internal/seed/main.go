package main

import (
	"log"

	"femProjectSqlc/internal/database"
	"femProjectSqlc/internal/mockdata"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	db, err := database.NewAppDB()
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.DB.Close()

	if err := mockdata.Seed(db.DB); err != nil {
		log.Fatalf("Seed failed: %v", err)
	}

	log.Println("Seed complete!")
}
