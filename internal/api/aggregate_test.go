package api_test

import (
	"es/internal/api"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCartAggregateFunctionality(t *testing.T) {
	t.Run("add single item", func(t *testing.T) {
		cart := api.NewCartAggregate(1001)
		assert.NoError(t, cart.Add(42))
		assert.Equal(t, []int{42}, cart.Contents)
	})

	t.Run("add single item multiple times", func(t *testing.T) {
		cart := api.NewCartAggregate(1001)
		assert.NoError(t, cart.Add(42))
		assert.NoError(t, cart.Add(42))
		assert.Equal(t, []int{42, 42}, cart.Contents)
	})

	t.Run("add and remove single item", func(t *testing.T) {
		cart := api.NewCartAggregate(1001)
		assert.NoError(t, cart.Add(42))
		assert.NoError(t, cart.Remove(42))
		assert.Equal(t, []int{}, cart.Contents)
	})

	t.Run("checkout", func(t *testing.T) {
		cart := api.NewCartAggregate(1001)

		assert.Equal(t, false, cart.CheckedOut)

		assert.NoError(t, cart.Checkout())

		assert.Equal(t, true, cart.CheckedOut)
	})

	t.Run("cannot add item to checked out cart", func(t *testing.T) {
		cart := api.NewCartAggregate(1001)
		assert.Equal(t, false, cart.CheckedOut)
		assert.NoError(t, cart.Checkout())

		err := cart.Add(42)
		assert.Error(t, err)
		assert.EqualError(t, err, "cannot add items to a checked out cart")
		assert.Equal(t, []int{}, cart.Contents)
		assert.Equal(t, true, cart.CheckedOut)
	})

	t.Run("cannot remove item from checked out cart", func(t *testing.T) {
		cart := api.NewCartAggregate(1001)
		assert.NoError(t, cart.Add(42))

		assert.NoError(t, cart.Checkout())

		err := cart.Remove(42)
		assert.Error(t, err)
		assert.EqualError(t, err, "cannot remove items from a checked out cart")
		assert.Equal(t, []int{42}, cart.Contents)
	})

	t.Run("remove non-existent item", func(t *testing.T) {
		cart := api.NewCartAggregate(1001)
		assert.NoError(t, cart.Remove(99))
		assert.Equal(t, []int{}, cart.Contents)
	})

	t.Run("multiple unique items", func(t *testing.T) {
		cart := api.NewCartAggregate(1001)
		assert.NoError(t, cart.Add(42))
		assert.NoError(t, cart.Add(43))
		assert.NoError(t, cart.Add(44))

		assert.Equal(t, []int{42, 43, 44}, cart.Contents)
	})

	t.Run("remove item from multiple items", func(t *testing.T) {
		cart := api.NewCartAggregate(1001)
		assert.NoError(t, cart.Add(42))
		assert.NoError(t, cart.Add(43))
		assert.NoError(t, cart.Add(44))

		assert.NoError(t, cart.Remove(43))

		assert.Equal(t, []int{42, 44}, cart.Contents)
	})

	t.Run("cannot checkout multiple times", func(t *testing.T) {
		cart := api.NewCartAggregate(1001)
		assert.NoError(t, cart.Checkout())

		err := cart.Checkout()
		assert.Error(t, err)
		assert.EqualError(t, err, "cart is already checked out")
	})

	t.Run("apply no events", func(t *testing.T) {
		cart := api.NewCartAggregate(1001)

		err := cart.Apply()
		assert.Error(t, err)
		assert.EqualError(t, err, "must apply at least 1 event")
	})
}
