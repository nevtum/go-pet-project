package v2

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type RouteHandler struct {
	repo ItemCountRepository
}

func NewRouteHandler(repo ItemCountRepository) *RouteHandler {
	return &RouteHandler{
		repo: repo,
	}
}

func (h *RouteHandler) Get(c *fiber.Ctx) error {
	res, err := h.repo.GetItemCounts(c.Context())

	if err != nil {
		return err
	}

	return c.Status(http.StatusOK).JSON(res)
}
