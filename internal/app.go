package internal

import (
	"es/internal/authentication"
	"es/internal/checkout"
	"es/internal/es"
	v2 "es/internal/inventory/v2"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewApi(pool *pgxpool.Pool) *fiber.App {
	eventStream := es.NewEventStream(pool)

	app := fiber.New()
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

	authMW := authentication.AuthMiddleware()
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
