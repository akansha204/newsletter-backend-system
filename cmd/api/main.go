package main

import (
	"fmt"
	"log"

	"github.com/akansh204/newsletter-backend-system/internal/config"
	"github.com/akansh204/newsletter-backend-system/internal/database"
)

func main() {
	cfg := config.Load()

	fmt.Println("=== Config Loaded ===")
	fmt.Printf("App Port : %s\n", cfg.App.Port)
	fmt.Printf("DB Host  : %s\n", cfg.DB.Host)
	fmt.Printf("DB Name  : %s\n", cfg.DB.Name)
	fmt.Println("=====================")

	db := database.Connect(&cfg.DB)
	defer db.Close()

	log.Println("all systems ready")
}
