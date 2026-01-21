package internal

import (
	"fmt"

	"es/internal/authentication"
	"es/internal/checkout"
	"es/internal/es"
	v2 "es/internal/inventory/v2"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewApi(pool *pgxpool.Pool) *fiber.App {
	eventStream := es.NewEventStream(pool)

	app := fiber.New()

	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	app.Use(healthcheck.New(healthcheck.Config{
		LivenessProbe: func(c *fiber.Ctx) bool {
			return true
		},
		LivenessEndpoint: "/livez",
		ReadinessProbe: func(c *fiber.Ctx) bool {
			return true
		},
		ReadinessEndpoint: "/readyz",
	}))
	app.Use(logger.New())

	cfg := authentication.LoadConfig()
	authHandlers := authentication.NewRouteHandler(cfg)

	app.Get("/login", authHandlers.Login)
	app.Get("/logout", authHandlers.Logout)
	app.Get("/callback", authHandlers.Callback)

	authMW, err := authentication.AuthMiddleware(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize auth middleware: %v", err))
	}
	repo := checkout.NewPGCartRepository(pool)
	usecase := checkout.NewCheckoutUseCase(repo)
	h := checkout.NewRouteHandler(usecase)

	api := app.Group("/cart", authMW)
	api.Get("/:cartID", h.GetCartDetails)
	api.Get("/:cartID/:itemID", h.AddItem)
	api.Get("/:cartID/:itemID/delete", h.RemoveItem)
	api.Post("/:cartID/checkout", h.Checkout)

	invRepo := v2.NewPGItemCountRepository(pool)
	invHandler := v2.NewRouteHandler(invRepo)

	inventoryApi := app.Group("/inventory/v2", authMW)
	inventoryApi.Get("/", invHandler.Get)

	eventsApi := app.Group("/events")

	eHandler := es.NewRouteHandler(eventStream)
	eventsApi.Get("/:aggType/:aggID", eHandler.AggregateEvents)

	return app
}
