package checkout

import (
	"es/internal/es"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

type RouteHandler struct {
	usecase *ShoppingCartUseCase
}

func NewShoppingCartHandler(repository CartRepository, eventStream *es.EventStream) *fiber.App {
	h := &RouteHandler{
		usecase: NewShoppingCartUseCase(repository),
	}

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

	api := app.Group("/cart")
	api.Get("/:cartID", h.GetCartDetails)
	api.Get("/:cartID/:itemID", h.AddItem)
	api.Get("/:cartID/:itemID/delete", h.RemoveItem)
	api.Post("/:cartID/checkout", h.Checkout)

	eventsApi := app.Group("/events")

	eHandler := es.NewEventsRouteHandler(eventStream)
	eventsApi.Get("/:aggType/:aggID", eHandler.AggregateEvents)

	return app
}

func (h *RouteHandler) GetCartDetails(c *fiber.Ctx) error {
	cartID, err := c.ParamsInt("cartID")

	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}

	cart, err := h.usecase.GetCartDetails(c.Context(), cartID)
	if err != nil {
		return err
	}

	return c.Status(http.StatusOK).JSON(cart)
}

func (h *RouteHandler) AddItem(c *fiber.Ctx) error {
	cartID, err := c.ParamsInt("cartID")

	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}

	itemID, err := c.ParamsInt("itemID")

	if err != nil {
		return err
	}

	cart, err := h.usecase.AddItemToCart(c.Context(), cartID, itemID)

	if err != nil {
		return err
	}

	return c.Status(http.StatusOK).JSON(cart)
}

func (h *RouteHandler) RemoveItem(c *fiber.Ctx) error {
	cartID, err := c.ParamsInt("cartID")

	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}

	itemID, err := c.ParamsInt("itemID")

	if err != nil {
		return err
	}

	cart, err := h.usecase.RemoveItemFromCart(c.Context(), cartID, itemID)

	if err != nil {
		return err
	}

	return c.Status(http.StatusOK).JSON(cart)
}

func (h *RouteHandler) Checkout(c *fiber.Ctx) error {
	cartID, err := c.ParamsInt("cartID")

	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}

	cart, err := h.usecase.Checkout(c.Context(), cartID)

	if err != nil {
		return err
	}

	return c.Status(http.StatusOK).JSON(cart)
}
