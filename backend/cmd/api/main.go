package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/database"
	"github.com/sainaif/holy-home/internal/handlers"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to MongoDB
	db, err := database.NewMongoDB(cfg.Mongo.URI, cfg.Mongo.Database)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := db.Close(ctx); err != nil {
			log.Printf("Error closing MongoDB connection: %v", err)
		}
	}()

	// Bootstrap admin user
	authService := services.NewAuthService(db.Database, cfg)
	if err := authService.BootstrapAdmin(context.Background()); err != nil {
		log.Fatalf("Failed to bootstrap admin: %v", err)
	}
	log.Println("Admin bootstrap complete")

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ErrorHandler: customErrorHandler,
	})

	// Global Middleware
	app.Use(recover.New())
	app.Use(middleware.RequestIDMiddleware())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, Idempotency-Key, X-Request-ID",
	}))

	if cfg.Logging.Format == "json" {
		app.Use(logger.New(logger.Config{
			Format: `{"time":"${time}","method":"${method}","path":"${path}","status":${status},"latency_ms":${latency},"ip":"${ip}","request_id":"${locals:requestId}"}` + "\n",
		}))
	} else {
		app.Use(logger.New())
	}

	// Health check endpoint
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"time":   time.Now(),
		})
	})

	// Initialize services
	userService := services.NewUserService(db.Database)
	groupService := services.NewGroupService(db.Database)
	billService := services.NewBillService(db.Database)
	consumptionService := services.NewConsumptionService(db.Database)
	loanService := services.NewLoanService(db.Database)
	choreService := services.NewChoreService(db.Database)
	predictionService := services.NewPredictionService(db.Database, cfg)
	eventService := services.NewEventService()
	exportService := services.NewExportService(db.Database)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	groupHandler := handlers.NewGroupHandler(groupService)
	billHandler := handlers.NewBillHandler(billService, consumptionService)
	loanHandler := handlers.NewLoanHandler(loanService)
	choreHandler := handlers.NewChoreHandler(choreService)
	predictionHandler := handlers.NewPredictionHandler(predictionService)
	// eventHandler := handlers.NewEventHandler(eventService) // Disabled due to crashes
	exportHandler := handlers.NewExportHandler(exportService)

	// Authentication routes
	auth := app.Group("/auth")
	auth.Post("/login", middleware.RateLimitMiddleware(5, 15*time.Minute), authHandler.Login)
	auth.Post("/refresh", middleware.RateLimitMiddleware(10, 15*time.Minute), authHandler.Refresh)

	// User routes
	users := app.Group("/users")
	users.Get("/", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), userHandler.GetUsers)
	users.Post("/", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), userHandler.CreateUser)
	users.Get("/me", middleware.AuthMiddleware(cfg), userHandler.GetMe)
	users.Get("/:id", middleware.AuthMiddleware(cfg), userHandler.GetUser)
	users.Patch("/:id", middleware.AuthMiddleware(cfg), userHandler.UpdateUser)
	users.Post("/change-password", middleware.AuthMiddleware(cfg), userHandler.ChangePassword)
	users.Post("/:id/force-password-change", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), userHandler.ForcePasswordChange)

	// Group routes
	groups := app.Group("/groups")
	groups.Get("/", middleware.AuthMiddleware(cfg), groupHandler.GetGroups)
	groups.Post("/", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), groupHandler.CreateGroup)
	groups.Get("/:id", middleware.AuthMiddleware(cfg), groupHandler.GetGroup)
	groups.Patch("/:id", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), groupHandler.UpdateGroup)
	groups.Delete("/:id", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), groupHandler.DeleteGroup)

	// Bill routes
	bills := app.Group("/bills")
	bills.Post("/", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), billHandler.CreateBill)
	bills.Get("/", middleware.AuthMiddleware(cfg), billHandler.GetBills)
	bills.Get("/:id", middleware.AuthMiddleware(cfg), billHandler.GetBill)
	bills.Post("/:id/allocate", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), billHandler.AllocateBill)
	bills.Post("/:id/post", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), billHandler.PostBill)
	bills.Post("/:id/close", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), billHandler.CloseBill)
	bills.Delete("/:id", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), billHandler.DeleteBill)

	// Consumption routes
	consumptions := app.Group("/consumptions")
	consumptions.Post("/", middleware.AuthMiddleware(cfg), billHandler.CreateConsumption)
	consumptions.Get("/", middleware.AuthMiddleware(cfg), billHandler.GetConsumptions)
	consumptions.Delete("/:id", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), billHandler.DeleteConsumption)
	consumptions.Post("/:id/mark-invalid", middleware.AuthMiddleware(cfg), billHandler.MarkConsumptionInvalid)

	// Allocation routes
	allocations := app.Group("/allocations")
	allocations.Get("/", middleware.AuthMiddleware(cfg), billHandler.GetAllocations)

	// Loan routes
	loans := app.Group("/loans")
	loans.Post("/", middleware.AuthMiddleware(cfg), loanHandler.CreateLoan)
	loans.Get("/", middleware.AuthMiddleware(cfg), loanHandler.GetLoans)
	loans.Get("/balances", middleware.AuthMiddleware(cfg), loanHandler.GetBalances)
	loans.Get("/balances/me", middleware.AuthMiddleware(cfg), loanHandler.GetMyBalance)
	loans.Get("/balances/user/:id", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), loanHandler.GetUserBalance)

	// Loan payment routes
	loanPayments := app.Group("/loan-payments")
	loanPayments.Post("/", middleware.AuthMiddleware(cfg), loanHandler.CreateLoanPayment)

	// Chore routes
	chores := app.Group("/chores")
	chores.Post("/", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), choreHandler.CreateChore)
	chores.Get("/", middleware.AuthMiddleware(cfg), choreHandler.GetChores)
	chores.Get("/with-assignments", middleware.AuthMiddleware(cfg), choreHandler.GetChoresWithAssignments)
	chores.Post("/assign", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), choreHandler.AssignChore)
	chores.Post("/swap", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), choreHandler.SwapChoreAssignment)
	chores.Post("/:id/rotate", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), choreHandler.RotateChore)
	chores.Post("/:id/auto-assign", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), choreHandler.AutoAssignChore)

	// Chore assignment routes
	choreAssignments := app.Group("/chore-assignments")
	choreAssignments.Get("/", middleware.AuthMiddleware(cfg), choreHandler.GetChoreAssignments)
	choreAssignments.Get("/me", middleware.AuthMiddleware(cfg), choreHandler.GetMyChoreAssignments)
	choreAssignments.Patch("/:id", middleware.AuthMiddleware(cfg), choreHandler.UpdateChoreAssignment)

	// Chore leaderboard
	app.Get("/chores/leaderboard", middleware.AuthMiddleware(cfg), choreHandler.GetUserLeaderboard)

	// Prediction routes
	predictions := app.Group("/predictions")
	predictions.Post("/recompute", middleware.AuthMiddleware(cfg), middleware.RequireRole("ADMIN"), predictionHandler.RecomputePrediction)
	predictions.Get("/", middleware.AuthMiddleware(cfg), predictionHandler.GetPredictions)

	// Events/SSE route - temporarily disabled due to crashes
	// events := app.Group("/events")
	// events.Get("/stream", middleware.AuthMiddleware(cfg), eventHandler.StreamEvents)

	// Export routes
	exports := app.Group("/exports")
	exports.Get("/bills", middleware.AuthMiddleware(cfg), exportHandler.ExportBills)
	exports.Get("/bills/:id/allocations", middleware.AuthMiddleware(cfg), exportHandler.ExportAllocations)
	exports.Get("/balances", middleware.AuthMiddleware(cfg), exportHandler.ExportBalances)
	exports.Get("/chores", middleware.AuthMiddleware(cfg), exportHandler.ExportChores)
	exports.Get("/consumptions", middleware.AuthMiddleware(cfg), exportHandler.ExportConsumptions)

	// Start nightly prediction job
	stopCron := startNightlyPredictionJob(predictionService, eventService)
	defer stopCron()
	log.Println("Nightly prediction job scheduled for 02:00 Europe/Warsaw")

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.App.Host, cfg.App.Port)
	go func() {
		log.Printf("Starting server on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}

// startNightlyPredictionJob starts a goroutine that runs predictions daily at 02:00
func startNightlyPredictionJob(predictionService *services.PredictionService, eventService *services.EventService) func() {
	stopChan := make(chan bool)

	go func() {
		// Load Warsaw timezone
		location, err := time.LoadLocation("Europe/Warsaw")
		if err != nil {
			log.Printf("Failed to load timezone: %v, using UTC", err)
			location = time.UTC
		}

		for {
			now := time.Now().In(location)

			// Calculate next 02:00
			next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, location)
			if now.After(next) {
				// If we're past 02:00 today, schedule for tomorrow
				next = next.Add(24 * time.Hour)
			}

			duration := next.Sub(now)
			log.Printf("Next prediction job scheduled for: %s (in %v)", next.Format(time.RFC3339), duration)

			select {
			case <-time.After(duration):
				// Run the prediction job
				log.Println("Starting nightly prediction job...")
				runPredictionJob(predictionService, eventService)
				log.Println("Nightly prediction job completed")

			case <-stopChan:
				log.Println("Stopping nightly prediction job")
				return
			}
		}
	}()

	// Return stop function
	return func() {
		close(stopChan)
	}
}

// runPredictionJob executes predictions for all targets
func runPredictionJob(predictionService *services.PredictionService, eventService *services.EventService) {
	ctx := context.Background()
	targets := []string{"electricity", "gas", "shared_budget"}
	horizon := 3 // 3 months default

	for _, target := range targets {
		log.Printf("Computing prediction for target: %s", target)

		req := services.RecomputeRequest{
			Target:  target,
			Horizon: horizon,
		}

		prediction, err := predictionService.RecomputePrediction(ctx, req)
		if err != nil {
			log.Printf("Failed to compute prediction for %s: %v", target, err)
			continue
		}

		// Broadcast SSE event
		eventService.Broadcast(services.EventPredictionUpdated, map[string]interface{}{
			"target":       target,
			"predictionId": prediction.ID.Hex(),
			"createdAt":    prediction.CreatedAt,
		})

		log.Printf("Successfully computed prediction for %s (ID: %s)", target, prediction.ID.Hex())
	}
}