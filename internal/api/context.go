package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// ApiContext is a wrapper around http.ResponseWriter and *http.Request that provides
// convenience methods for writing less verbose http handlers.
type ApiContext struct {
	w http.ResponseWriter
	r *http.Request
}

func (c *ApiContext) IntParam(key string) (int, error) {
	value := c.r.PathValue(key)
	return convertToInt(key, value)
}

func (c *ApiContext) RequestContext() context.Context {
	return c.r.Context()
}

func (c *ApiContext) JSON(a any) error {
	return json.NewEncoder(c.w).Encode(a)
}

func (c *ApiContext) OK() *ApiContext {
	c.w.WriteHeader(http.StatusOK)
	return c
}

func (c *ApiContext) BadRequest(err error) error {
	http.Error(c.w, err.Error(), http.StatusBadRequest)
	return nil
}

func (c *ApiContext) InternalServerError(err error) error {
	http.Error(c.w, err.Error(), http.StatusInternalServerError)
	return nil
}

func ToHandleFunc(handler func(c *ApiContext) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &ApiContext{
			w: w,
			r: r,
		}
		if err := handler(ctx); err != nil {
			ctx.InternalServerError(err)
		}
	}
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
