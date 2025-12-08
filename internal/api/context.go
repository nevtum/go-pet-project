package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// Context is a wrapper around http.ResponseWriter and *http.Request that provides
// convenience methods for writing less verbose http handlers.
type Context struct {
	w http.ResponseWriter
	r *http.Request
}

func (c *Context) IntParam(key string) (int, error) {
	value := c.r.PathValue(key)
	return convertToInt(key, value)
}

func (c *Context) RequestContext() context.Context {
	return c.r.Context()
}

func (c *Context) JSON(a any) error {
	return json.NewEncoder(c.w).Encode(a)
}

func (c *Context) OK() *Context {
	c.w.WriteHeader(http.StatusOK)
	return c
}

func (c *Context) BadRequest(err error) error {
	http.Error(c.w, err.Error(), http.StatusBadRequest)
	return nil
}

func (c *Context) InternalServerError(err error) error {
	http.Error(c.w, err.Error(), http.StatusInternalServerError)
	return nil
}

func ToHandleFunc(handler func(c *Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{
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
