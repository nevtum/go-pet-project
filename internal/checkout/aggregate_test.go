package checkout_test

import (
	"es/internal/checkout"
	"es/internal/util"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"es/internal/es"
)

func TestCartAggregateCommands(t *testing.T) {
	t.Run("add single item", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)
		assert.NoError(t, cart.Add(42))
		assert.Equal(t, []int{42}, cart.Contents)
	})

	t.Run("add single item multiple times", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)
		assert.NoError(t, cart.Add(42))
		assert.NoError(t, cart.Add(42))
		assert.Equal(t, []int{42, 42}, cart.Contents)
	})

	t.Run("add and remove single item", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)
		assert.NoError(t, cart.Add(42))
		assert.NoError(t, cart.Remove(42))
		assert.Equal(t, []int{}, cart.Contents)
	})

	t.Run("checkout", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)

		assert.Equal(t, false, cart.CheckedOut)

		assert.NoError(t, cart.Checkout())

		assert.Equal(t, true, cart.CheckedOut)
	})

	t.Run("cannot add item to checked out cart", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)
		assert.Equal(t, false, cart.CheckedOut)
		assert.NoError(t, cart.Checkout())

		err := cart.Add(42)
		assert.Error(t, err)
		assert.EqualError(t, err, "cannot add items to a checked out cart")
		assert.Equal(t, []int{}, cart.Contents)
		assert.Equal(t, true, cart.CheckedOut)
	})

	t.Run("cannot remove item from checked out cart", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)
		assert.NoError(t, cart.Add(42))

		assert.NoError(t, cart.Checkout())

		err := cart.Remove(42)
		assert.Error(t, err)
		assert.EqualError(t, err, "cannot remove items from a checked out cart")
		assert.Equal(t, []int{42}, cart.Contents)
	})

	t.Run("remove non-existent item", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)
		assert.NoError(t, cart.Remove(99))
		assert.Equal(t, []int{}, cart.Contents)
	})

	t.Run("multiple unique items", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)
		assert.NoError(t, cart.Add(42))
		assert.NoError(t, cart.Add(43))
		assert.NoError(t, cart.Add(44))

		assert.Equal(t, []int{42, 43, 44}, cart.Contents)
	})

	t.Run("remove item from multiple items", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)
		assert.NoError(t, cart.Add(42))
		assert.NoError(t, cart.Add(43))
		assert.NoError(t, cart.Add(44))

		assert.NoError(t, cart.Remove(43))

		assert.Equal(t, []int{42, 44}, cart.Contents)
	})

	t.Run("cannot checkout multiple times", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)
		assert.NoError(t, cart.Checkout())

		err := cart.Checkout()
		assert.Error(t, err)
		assert.EqualError(t, err, "cart is already checked out")
	})
}

func TestCartAggregateEvents(t *testing.T) {
	t.Run("cart aggregate initial events", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)
		assert.NoError(t, cart.Add(42))

		assert.Equal(t, []es.Event{
			{
				Type:          checkout.CartCreated,
				At:            atTimeDelta(0),
				VersionID:     1,
				AggregateType: checkout.CartType,
				AggregateID:   1001,
				Data:          map[string]any{},
			},
			{
				Type:          checkout.ItemAddedToCart,
				At:            atTimeDelta(1),
				VersionID:     2,
				AggregateType: checkout.CartType,
				AggregateID:   1001,
				Data:          map[string]int{"item_id": 42},
			},
		}, cart.UncommittedEvents())
	})

	t.Run("apply no events", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)

		err := cart.Apply()
		assert.Error(t, err)
		assert.EqualError(t, err, "must apply at least 1 event")
	})

	t.Run("apply unknown event type returns error", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)

		// Create an event with an unimplemented type to trigger the default case
		unknownEvent := es.Event{
			Type: "UnknownEventType",
			Data: nil,
		}

		err := cart.Apply(unknownEvent)
		assert.Error(t, err)
		assert.EqualError(t, err, "not implemented")
	})

	t.Run("apply event with invalid item ID data", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)

		// Create an event with invalid item ID data
		invalidItemEvent := es.Event{
			Type: checkout.ItemAddedToCart,
			Data: map[string]string{"wrong_key": "123"},
		}

		err := cart.Apply(invalidItemEvent)
		assert.Error(t, err)
	})

	t.Run("apply multiple different events", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)

		// Prepare multiple events to apply in a single call
		events := []es.Event{
			{Type: checkout.ItemAddedToCart, Data: map[string]int{"item_id": 42}},
			{Type: checkout.ItemAddedToCart, Data: map[string]int{"item_id": 43}},
			{Type: checkout.CartCheckedOut},
		}

		err := cart.Apply(events...)
		assert.NoError(t, err)
		assert.Equal(t, []int{42, 43}, cart.Contents)
		assert.True(t, cart.CheckedOut)
	})

	t.Run("remove item from non-consecutive position", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)

		// Add multiple items and remove a non-first, non-last item
		events := []es.Event{
			{Type: checkout.ItemAddedToCart, Data: map[string]int{"item_id": 10}},
			{Type: checkout.ItemAddedToCart, Data: map[string]int{"item_id": 20}},
			{Type: checkout.ItemAddedToCart, Data: map[string]int{"item_id": 30}},
			{Type: checkout.ItemRemovedFromCart, Data: map[string]int{"item_id": 20}},
		}

		err := cart.Apply(events...)
		assert.NoError(t, err)
		assert.Equal(t, []int{10, 30}, cart.Contents)
	})

	t.Run("apply event with unmarshalable data", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)

		// Create an event with data that cannot be JSON marshaled
		invalidEvent := es.Event{
			Type: checkout.ItemAddedToCart,
			Data: make(chan int), // Channels cannot be JSON marshaled
		}

		err := cart.Apply(invalidEvent)
		assert.Error(t, err)
	})

	t.Run("commit events", func(t *testing.T) {
		cart := newTestCartAggregate(t, 1001)

		events := []es.Event{
			{Type: checkout.ItemAddedToCart, Data: map[string]int{"item_id": 10}},
			{Type: checkout.ItemAddedToCart, Data: map[string]int{"item_id": 20}},
			{Type: checkout.ItemRemovedFromCart, Data: map[string]int{"item_id": 10}},
		}

		assert.NoError(t, cart.Apply(events...))

		assert.Equal(t, append(
			[]es.Event{
				{
					Type:          checkout.CartCreated,
					At:            atTimeDelta(0),
					VersionID:     1,
					AggregateType: checkout.CartType,
					AggregateID:   1001,
					Data:          map[string]any{},
				},
			}, events...), cart.UncommittedEvents())

		cart.Commit()
		assert.Equal(t, []es.Event{}, cart.UncommittedEvents())

		// Ensure idempotent by committing again
		cart.Commit()
		assert.Equal(t, []es.Event{}, cart.UncommittedEvents())
	})
}

func newTestCartAggregate(t *testing.T, cartID int) *checkout.CartAggregate {
	t.Helper()

	cart := checkout.NewCartAggregate(cartID, checkout.UseTimestamp(
		util.SequencedTime(atTimeDelta(0)),
	))

	assert.NoError(t, cart.Init())
	return cart
}

func atTimeDelta(ns int) time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, ns, time.UTC)
}
