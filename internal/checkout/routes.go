package checkout

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type RouteHandler struct {
	usecase *CheckoutUseCase
}

func NewRouteHandler(usecase *CheckoutUseCase) *RouteHandler {
	return &RouteHandler{
		usecase: usecase,
	}
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
