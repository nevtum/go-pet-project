package es

import (
	"errors"
	"es/internal/api"
)

type EventsRouteHandler struct {
	eventStream *EventStream
}

func NewEventsRouteHandler(eventStream *EventStream) *EventsRouteHandler {
	return &EventsRouteHandler{
		eventStream: eventStream,
	}
}

func (h *EventsRouteHandler) AggregateEvents(c *api.Context) error {
	aggType := c.StringParam("aggType")

	if aggType == "" {
		return c.BadRequest(errors.New("aggType is required"))
	}

	aggID, err := c.IntParam("aggID")

	if err != nil {
		return c.BadRequest(err)
	}

	events, err := h.eventStream.GetAggregateEvents(c.RequestContext(), AggregateType(aggType), aggID)

	if err != nil {
		return err
	}

	return c.OK().JSON(events)
}
