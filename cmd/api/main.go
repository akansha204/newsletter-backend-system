package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/akansh204/newsletter-backend-system/internal/api"

	"github.com/akansh204/newsletter-backend-system/internal/config"
	"github.com/akansh204/newsletter-backend-system/internal/database"
)

func main() {
	cfg := config.Load()

	fmt.Println("=== Newsletter System Starting ===")

	db := database.Connect(&cfg.DB)
	defer db.Close()

	app := fiber.New(fiber.Config{
		AppName: "Newsletter System v1",
	})

	api.SetupRoutes(app, db)
	log.Printf("server starting on port %s", cfg.App.Port)
	if err := app.Listen(":" + cfg.App.Port); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}

}
