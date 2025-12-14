package es

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type RouteHandler struct {
	eventStream *EventStream
}

func NewRouteHandler(eventStream *EventStream) *RouteHandler {
	return &RouteHandler{
		eventStream: eventStream,
	}
}

func (h *RouteHandler) AggregateEvents(c *fiber.Ctx) error {
	aggType := c.Params("aggType")

	if aggType == "" {
		return c.Status(http.StatusBadRequest).SendString("aggType is required")
	}

	aggID, err := c.ParamsInt("aggID")

	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}

	events, err := h.eventStream.GetAggregateEvents(c.Context(), AggregateType(aggType), aggID)

	if err != nil {
		return err
	}

	return c.Status(http.StatusOK).JSON(events)
}
