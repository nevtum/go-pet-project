package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type RouteHandler struct {
	usecase *ShoppingCartUseCase
}

func NewShoppingCartHandler(repository CartRepository) http.Handler {
	h := &RouteHandler{
		usecase: NewShoppingCartUseCase(repository),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("GET /cart/{cartID}", h.GetCartDetails)
	mux.HandleFunc("GET /cart/{cartID}/{itemID}", h.AddItem)
	mux.HandleFunc("GET /cart/{cartID}/{itemID}/delete", h.RemoveItem)
	mux.HandleFunc("GET /checkout/{cartID}", h.Checkout)

	return mux
}

func (h *RouteHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Healthy!")
}

func (h *RouteHandler) GetCartDetails(w http.ResponseWriter, r *http.Request) {
	cartID, err := convertToInt("cartID", r.PathValue("cartID"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cart, err := h.usecase.GetCartDetails(r.Context(), cartID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(cart)
}

func (h *RouteHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	cartID, err := convertToInt("cartID", r.PathValue("cartID"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	itemID, err := convertToInt("itemID", r.PathValue("itemID"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cart, err := h.usecase.AddItemToCart(r.Context(), cartID, itemID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(cart)
}

func (h *RouteHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	cartID, err := convertToInt("cartID", r.PathValue("cartID"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	itemID, err := convertToInt("itemID", r.PathValue("itemID"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cart, err := h.usecase.RemoveItemFromCart(r.Context(), cartID, itemID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(cart)
}

func (h *RouteHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	cartID, err := convertToInt("cartID", r.PathValue("cartID"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cart, err := h.usecase.Checkout(r.Context(), cartID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(cart)
}

func convertToInt(key string, value string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("%s is required", key)
	}

	intVal, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return intVal, nil
}
