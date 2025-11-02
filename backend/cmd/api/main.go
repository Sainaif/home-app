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

	// Initialize services
	sessionService := services.NewSessionService(db.Database)
	authService := services.NewAuthService(db.Database, cfg, sessionService)
	if err := authService.BootstrapAdmin(context.Background()); err != nil {
		log.Fatalf("Failed to bootstrap admin: %v", err)
	}
	log.Println("Admin bootstrap complete")

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:           cfg.App.Name,
		ErrorHandler:      customErrorHandler,
		EnableTrustedProxyCheck: true,
		TrustedProxies:    []string{"172.20.0.0/16", "10.0.0.0/8", "127.0.0.1"},
		ProxyHeader:       fiber.HeaderXForwardedFor,
	})

	// Global Middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			log.Printf("PANIC: %s %s - %v", c.Method(), c.Path(), e)
		},
	}))
	app.Use(middleware.RequestIDMiddleware())

	// Smart cache control middleware - GET requests cacheable, mutations not
	app.Use(func(c *fiber.Ctx) error {
		method := c.Method()

		if method == "GET" {
			// GET requests: Allow caching but require revalidation
			c.Set("Cache-Control", "no-cache")
		} else {
			// POST, PATCH, DELETE, PUT: No caching
			c.Set("Cache-Control", "no-store, must-revalidate")
		}

		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.Next()
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, Idempotency-Key, X-Request-ID",
		ExposeHeaders: "Cache-Control, Pragma, Expires",
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
	userService := services.NewUserService(db.Database, cfg)
	groupService := services.NewGroupService(db.Database)
	billService := services.NewBillService(db.Database)
	consumptionService := services.NewConsumptionService(db.Database)
	allocationService := services.NewAllocationService(db.Database)
	loanService := services.NewLoanService(db.Database)
	choreService := services.NewChoreService(db.Database)
	supplyService := services.NewSupplyService(db.Database)
	recurringBillService := services.NewRecurringBillService(db.Database, cfg)
	eventService := services.NewEventService()
	exportService := services.NewExportService(db.Database)
	backupService := services.NewBackupService(db.Database)
	auditService := services.NewAuditService(db.Database)
	permissionService := services.NewPermissionService(db.Database)
	roleService := services.NewRoleService(db.Database)
	approvalService := services.NewApprovalService(db.Database)

	// Initialize default permissions and roles
	if err := permissionService.InitializeDefaultPermissions(context.Background()); err != nil {
		log.Printf("Warning: Failed to initialize permissions: %v", err)
	}
	if err := roleService.InitializeDefaultRoles(context.Background()); err != nil {
		log.Printf("Warning: Failed to initialize roles: %v", err)
	}
	log.Println("Permissions and roles initialized")

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, userService, auditService)
	sessionHandler := handlers.NewSessionHandler(sessionService)
	userHandler := handlers.NewUserHandler(userService, auditService, roleService, cfg)
	groupHandler := handlers.NewGroupHandler(groupService, auditService)
	billHandler := handlers.NewBillHandler(billService, consumptionService, allocationService, auditService, eventService)
	recurringBillHandler := handlers.NewRecurringBillHandler(recurringBillService, auditService)
	loanHandler := handlers.NewLoanHandler(loanService, eventService, auditService)
	choreHandler := handlers.NewChoreHandler(choreService, approvalService, roleService, auditService, eventService)
	supplyHandler := handlers.NewSupplyHandler(supplyService, auditService, eventService)
	backupHandler := handlers.NewBackupHandler(backupService)
	eventHandler := handlers.NewEventHandler(eventService)
	exportHandler := handlers.NewExportHandler(exportService)
	auditHandler := handlers.NewAuditHandler(auditService)
	roleHandler := handlers.NewRoleHandler(roleService, permissionService, auditService, eventService, userService)
	approvalHandler := handlers.NewApprovalHandler(approvalService)

	// Helper function to provide RoleService to middleware
	getRoleService := func() interface{} { return roleService }

	// Authentication routes
	auth := app.Group("/auth")
	auth.Post("/login", middleware.RateLimitMiddleware(5, 15*time.Minute), authHandler.Login)
	auth.Post("/refresh", middleware.RateLimitMiddleware(10, 15*time.Minute), authHandler.Refresh)
	auth.Post("/enable-2fa", middleware.AuthMiddleware(cfg), authHandler.Enable2FA)
	auth.Post("/disable-2fa", middleware.AuthMiddleware(cfg), authHandler.Disable2FA)

	// Passkey routes
	auth.Post("/passkey/register/begin", middleware.AuthMiddleware(cfg), authHandler.BeginPasskeyRegistration)
	auth.Post("/passkey/register/finish", middleware.AuthMiddleware(cfg), authHandler.FinishPasskeyRegistration)
	auth.Post("/passkey/login/begin", authHandler.BeginPasskeyLogin)
	auth.Post("/passkey/login/finish", authHandler.FinishPasskeyLogin)
	auth.Get("/passkeys", middleware.AuthMiddleware(cfg), authHandler.ListPasskeys)
	auth.Delete("/passkeys", middleware.AuthMiddleware(cfg), authHandler.DeletePasskey)

	// Password reset routes (public)
	auth.Get("/validate-reset-token", authHandler.ValidateResetToken)
	auth.Post("/reset-password", middleware.RateLimitMiddleware(5, 15*time.Minute), authHandler.ResetPasswordWithToken)

	// Session routes
	sessions := app.Group("/sessions")
	sessions.Get("/", middleware.AuthMiddleware(cfg), sessionHandler.GetSessions)
	sessions.Patch("/:id", middleware.AuthMiddleware(cfg), sessionHandler.RenameSession)
	sessions.Delete("/:id", middleware.AuthMiddleware(cfg), sessionHandler.DeleteSession)

	// User routes
	users := app.Group("/users")
	users.Get("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("users.read", getRoleService), userHandler.GetUsers)
	users.Post("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("users.create", getRoleService), userHandler.CreateUser)
	users.Get("/me", middleware.AuthMiddleware(cfg), userHandler.GetMe)
	users.Get("/:id", middleware.AuthMiddleware(cfg), userHandler.GetUser)
	users.Patch("/:id", middleware.AuthMiddleware(cfg), userHandler.UpdateUser)
	users.Delete("/:id", middleware.AuthMiddleware(cfg), middleware.RequirePermission("users.delete", getRoleService), userHandler.DeleteUser)
	users.Post("/change-password", middleware.AuthMiddleware(cfg), userHandler.ChangePassword)
	users.Post("/:id/force-password-change", middleware.AuthMiddleware(cfg), middleware.RequirePermission("users.update", getRoleService), userHandler.ForcePasswordChange)
	users.Post("/:id/generate-reset-link", middleware.AuthMiddleware(cfg), middleware.RequirePermission("users.update", getRoleService), userHandler.GeneratePasswordResetLink)

	// Group routes
	groups := app.Group("/groups")
	groups.Get("/", middleware.AuthMiddleware(cfg), groupHandler.GetGroups)
	groups.Post("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("groups.create", getRoleService), groupHandler.CreateGroup)
	groups.Get("/:id", middleware.AuthMiddleware(cfg), groupHandler.GetGroup)
	groups.Patch("/:id", middleware.AuthMiddleware(cfg), middleware.RequirePermission("groups.update", getRoleService), groupHandler.UpdateGroup)
	groups.Delete("/:id", middleware.AuthMiddleware(cfg), middleware.RequirePermission("groups.delete", getRoleService), groupHandler.DeleteGroup)

	// Bill routes
	bills := app.Group("/bills")
	bills.Post("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("bills.create", getRoleService), billHandler.CreateBill)
	bills.Get("/", middleware.AuthMiddleware(cfg), billHandler.GetBills)
	bills.Get("/:id", middleware.AuthMiddleware(cfg), billHandler.GetBill)
	bills.Post("/:id/post", middleware.AuthMiddleware(cfg), middleware.RequirePermission("bills.post", getRoleService), billHandler.PostBill)
	bills.Post("/:id/close", middleware.AuthMiddleware(cfg), middleware.RequirePermission("bills.close", getRoleService), billHandler.CloseBill)
	bills.Post("/:id/reopen", middleware.AuthMiddleware(cfg), middleware.RequirePermission("bills.update", getRoleService), billHandler.ReopenBill)
	bills.Delete("/:id", middleware.AuthMiddleware(cfg), middleware.RequirePermission("bills.delete", getRoleService), billHandler.DeleteBill)
	bills.Get("/:id/allocation", middleware.AuthMiddleware(cfg), billHandler.GetBillAllocation)
	bills.Get("/:id/payment-status", middleware.AuthMiddleware(cfg), billHandler.GetBillPaymentStatus)

	// Consumption routes
	consumptions := app.Group("/consumptions")
	consumptions.Post("/", middleware.AuthMiddleware(cfg), billHandler.CreateConsumption)
	consumptions.Get("/", middleware.AuthMiddleware(cfg), billHandler.GetConsumptions)
	consumptions.Delete("/:id", middleware.AuthMiddleware(cfg), middleware.RequirePermission("readings.delete", getRoleService), billHandler.DeleteConsumption)
	consumptions.Post("/:id/mark-invalid", middleware.AuthMiddleware(cfg), billHandler.MarkConsumptionInvalid)

	// Recurring bill routes
	recurringBills := app.Group("/recurring-bills")
	recurringBills.Post("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("bills.create", getRoleService), recurringBillHandler.CreateRecurringBillTemplate)
	recurringBills.Get("/", middleware.AuthMiddleware(cfg), recurringBillHandler.GetRecurringBillTemplates)
	recurringBills.Get("/:id", middleware.AuthMiddleware(cfg), recurringBillHandler.GetRecurringBillTemplate)
	recurringBills.Patch("/:id", middleware.AuthMiddleware(cfg), middleware.RequirePermission("bills.update", getRoleService), recurringBillHandler.UpdateRecurringBillTemplate)
	recurringBills.Delete("/:id", middleware.AuthMiddleware(cfg), middleware.RequirePermission("bills.delete", getRoleService), recurringBillHandler.DeleteRecurringBillTemplate)
	recurringBills.Post("/generate", middleware.AuthMiddleware(cfg), middleware.RequirePermission("bills.create", getRoleService), recurringBillHandler.GenerateRecurringBills)

	// Payment routes
	paymentHandler := handlers.NewPaymentHandler(db.Database, auditService, recurringBillService)
	payments := app.Group("/payments")
	payments.Post("/", middleware.AuthMiddleware(cfg), paymentHandler.RecordPayment)
	payments.Get("/me", middleware.AuthMiddleware(cfg), paymentHandler.GetUserPayments)
	payments.Get("/bill/:billId", middleware.AuthMiddleware(cfg), paymentHandler.GetBillPayments)

	// Loan routes
	loans := app.Group("/loans")
	loans.Post("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("loans.create", getRoleService), loanHandler.CreateLoan)
	loans.Post("/compensate", middleware.AuthMiddleware(cfg), middleware.RequirePermission("loans.create", getRoleService), loanHandler.CompensateLoan)
	loans.Get("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("loans.read", getRoleService), loanHandler.GetLoans)
	loans.Get("/balances", middleware.AuthMiddleware(cfg), middleware.RequirePermission("loans.read", getRoleService), loanHandler.GetBalances)
	loans.Get("/balances/me", middleware.AuthMiddleware(cfg), middleware.RequirePermission("loans.read", getRoleService), loanHandler.GetMyBalance)
	loans.Get("/balances/user/:id", middleware.AuthMiddleware(cfg), middleware.RequirePermission("loans.read", getRoleService), loanHandler.GetUserBalance)
	loans.Get("/:id/payments", middleware.AuthMiddleware(cfg), middleware.RequirePermission("loans.read", getRoleService), loanHandler.GetLoanPayments)
	loans.Delete("/:id", middleware.AuthMiddleware(cfg), middleware.RequirePermission("loans.delete", getRoleService), loanHandler.DeleteLoan)

	// Loan payment routes
	loanPayments := app.Group("/loan-payments")
	loanPayments.Post("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("loan-payments.create", getRoleService), loanHandler.CreateLoanPayment)

	// Chore routes
	chores := app.Group("/chores")
	chores.Post("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("chores.create", getRoleService), choreHandler.CreateChore)
	chores.Get("/", middleware.AuthMiddleware(cfg), choreHandler.GetChores)
	chores.Get("/with-assignments", middleware.AuthMiddleware(cfg), choreHandler.GetChoresWithAssignments)
	chores.Delete("/:id", middleware.AuthMiddleware(cfg), choreHandler.DeleteChore)
	chores.Post("/assign", middleware.AuthMiddleware(cfg), middleware.RequirePermission("chores.assign", getRoleService), choreHandler.AssignChore)
	chores.Post("/swap", middleware.AuthMiddleware(cfg), middleware.RequirePermission("chores.assign", getRoleService), choreHandler.SwapChoreAssignment)
	chores.Post("/:id/rotate", middleware.AuthMiddleware(cfg), middleware.RequirePermission("chores.assign", getRoleService), choreHandler.RotateChore)
	chores.Post("/:id/auto-assign", middleware.AuthMiddleware(cfg), middleware.RequirePermission("chores.assign", getRoleService), choreHandler.AutoAssignChore)

	// Chore assignment routes
	choreAssignments := app.Group("/chore-assignments")
	choreAssignments.Get("/", middleware.AuthMiddleware(cfg), choreHandler.GetChoreAssignments)
	choreAssignments.Get("/me", middleware.AuthMiddleware(cfg), choreHandler.GetMyChoreAssignments)
	choreAssignments.Patch("/:id", middleware.AuthMiddleware(cfg), choreHandler.UpdateChoreAssignment)

	// Chore leaderboard
	app.Get("/chores/leaderboard", middleware.AuthMiddleware(cfg), choreHandler.GetUserLeaderboard)

	// Supply routes
	supplies := app.Group("/supplies")

	// Settings
	supplies.Get("/settings", middleware.AuthMiddleware(cfg), supplyHandler.GetSettings)
	supplies.Patch("/settings", middleware.AuthMiddleware(cfg), middleware.RequirePermission("supplies.update", getRoleService), supplyHandler.UpdateSettings)
	supplies.Post("/settings/adjust", middleware.AuthMiddleware(cfg), middleware.RequirePermission("supplies.update", getRoleService), supplyHandler.AdjustBudget)

	// Items
	supplies.Get("/items", middleware.AuthMiddleware(cfg), supplyHandler.GetItems)
	supplies.Post("/items", middleware.AuthMiddleware(cfg), supplyHandler.CreateItem)
	supplies.Patch("/items/:id", middleware.AuthMiddleware(cfg), supplyHandler.UpdateItem)
	supplies.Post("/items/:id/restock", middleware.AuthMiddleware(cfg), supplyHandler.RestockItem)
	supplies.Post("/items/:id/consume", middleware.AuthMiddleware(cfg), supplyHandler.ConsumeItem)
	supplies.Patch("/items/:id/quantity", middleware.AuthMiddleware(cfg), supplyHandler.SetQuantity)
	supplies.Post("/items/:id/refund", middleware.AuthMiddleware(cfg), middleware.RequirePermission("supplies.update", getRoleService), supplyHandler.MarkAsRefunded)
	supplies.Delete("/items/:id", middleware.AuthMiddleware(cfg), supplyHandler.DeleteItem)

	// Contributions
	supplies.Get("/contributions", middleware.AuthMiddleware(cfg), supplyHandler.GetContributions)
	supplies.Post("/contributions", middleware.AuthMiddleware(cfg), supplyHandler.CreateContribution)

	// Stats
	supplies.Get("/stats", middleware.AuthMiddleware(cfg), supplyHandler.GetStats)

	// Events/SSE route
	events := app.Group("/events")
	events.Get("/stream", middleware.AuthMiddleware(cfg), eventHandler.StreamEvents)

	// Export routes
	exports := app.Group("/exports")
	exports.Get("/bills", middleware.AuthMiddleware(cfg), exportHandler.ExportBills)

	// Audit log routes
	audit := app.Group("/audit")
	audit.Get("/logs", middleware.AuthMiddleware(cfg), middleware.RequirePermission("audit.read", getRoleService), auditHandler.GetLogs)

	// Role and permission routes
	roles := app.Group("/roles")
	roles.Get("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("roles.read", getRoleService), roleHandler.GetAllRoles)
	roles.Post("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("roles.create", getRoleService), roleHandler.CreateRole)
	roles.Patch("/:id", middleware.AuthMiddleware(cfg), middleware.RequirePermission("roles.update", getRoleService), roleHandler.UpdateRole)
	roles.Delete("/:id", middleware.AuthMiddleware(cfg), middleware.RequirePermission("roles.delete", getRoleService), roleHandler.DeleteRole)

	permissions := app.Group("/permissions")
	permissions.Get("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("roles.read", getRoleService), roleHandler.GetAllPermissions)

	// Approval routes
	approvals := app.Group("/approvals")
	approvals.Get("/pending", middleware.AuthMiddleware(cfg), middleware.RequirePermission("approvals.review", getRoleService), approvalHandler.GetPendingRequests)
	approvals.Get("/", middleware.AuthMiddleware(cfg), middleware.RequirePermission("approvals.review", getRoleService), approvalHandler.GetAllRequests)
	approvals.Post("/:id/approve", middleware.AuthMiddleware(cfg), middleware.RequirePermission("approvals.review", getRoleService), approvalHandler.ApproveRequest)
	approvals.Post("/:id/reject", middleware.AuthMiddleware(cfg), middleware.RequirePermission("approvals.review", getRoleService), approvalHandler.RejectRequest)

	// Backup routes
	backup := app.Group("/backup")
	backup.Get("/export", middleware.AuthMiddleware(cfg), middleware.RequirePermission("backup.export", getRoleService), backupHandler.ExportBackup)
	backup.Post("/import", middleware.AuthMiddleware(cfg), middleware.RequirePermission("backup.import", getRoleService), backupHandler.ImportBackup)
	exports.Get("/balances", middleware.AuthMiddleware(cfg), exportHandler.ExportBalances)
	exports.Get("/chores", middleware.AuthMiddleware(cfg), exportHandler.ExportChores)
	exports.Get("/consumptions", middleware.AuthMiddleware(cfg), exportHandler.ExportConsumptions)

	// Start password reset token cleanup job
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()

		// Run cleanup immediately on startup
		log.Println("Running initial password reset token cleanup...")
		if err := userService.CleanupExpiredResetTokens(context.Background()); err != nil {
			log.Printf("Error during initial cleanup: %v", err)
		}

		// Run cleanup every 6 hours
		for range ticker.C {
			log.Println("Running scheduled password reset token cleanup...")
			if err := userService.CleanupExpiredResetTokens(context.Background()); err != nil {
				log.Printf("Error during scheduled cleanup: %v", err)
			}
		}
	}()

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

	// Log the error with details
	log.Printf("ERROR: %s %s - Status: %d - Error: %v", c.Method(), c.Path(), code, err)

	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}