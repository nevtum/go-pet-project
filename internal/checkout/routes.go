package checkout

import (
	"es/internal/api"
	"es/internal/es"
	"fmt"
	"net/http"
)

type RouteHandler struct {
	usecase *ShoppingCartUseCase
}

func NewShoppingCartHandler(repository CartRepository, eventStream *es.EventStream) http.Handler {
	h := &RouteHandler{
		usecase: NewShoppingCartUseCase(repository),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Healthy!")
	})
	mux.HandleFunc("GET /cart/{cartID}", api.ToHandleFunc(h.GetCartDetails))
	mux.HandleFunc("GET /cart/{cartID}/{itemID}", api.ToHandleFunc(h.AddItem))
	mux.HandleFunc("GET /cart/{cartID}/{itemID}/delete", api.ToHandleFunc(h.RemoveItem))
	mux.HandleFunc("GET /checkout/{cartID}", api.ToHandleFunc(h.Checkout))

	eHandler := es.NewEventsRouteHandler(eventStream)
	mux.Handle("/events/{aggType}/{aggID}", api.ToHandleFunc(eHandler.AggregateEvents))

	return mux
}

func (h *RouteHandler) GetCartDetails(c *api.Context) error {
	cartID, err := c.IntParam("cartID")

	if err != nil {
		return c.BadRequest(err)
	}

	cart, err := h.usecase.GetCartDetails(c.RequestContext(), cartID)
	if err != nil {
		return err
	}

	return c.OK().JSON(cart)
}

func (h *RouteHandler) AddItem(c *api.Context) error {
	cartID, err := c.IntParam("cartID")

	if err != nil {
		return c.BadRequest(err)
	}

	itemID, err := c.IntParam("itemID")

	if err != nil {
		return err
	}

	cart, err := h.usecase.AddItemToCart(c.RequestContext(), cartID, itemID)

	if err != nil {
		return err
	}

	return c.OK().JSON(cart)
}

func (h *RouteHandler) RemoveItem(c *api.Context) error {
	cartID, err := c.IntParam("cartID")

	if err != nil {
		return c.BadRequest(err)
	}

	itemID, err := c.IntParam("itemID")

	if err != nil {
		return err
	}

	cart, err := h.usecase.RemoveItemFromCart(c.RequestContext(), cartID, itemID)

	if err != nil {
		return err
	}

	return c.OK().JSON(cart)
}

func (h *RouteHandler) Checkout(c *api.Context) error {
	cartID, err := c.IntParam("cartID")

	if err != nil {
		return c.BadRequest(err)
	}

	cart, err := h.usecase.Checkout(c.RequestContext(), cartID)

	if err != nil {
		return err
	}

	return c.OK().JSON(cart)
}
