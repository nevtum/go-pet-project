package api

import (
	"es/internal/es"
)

func (c *CartAggregate) newCartCreatedEvent(cartID int) es.Event {
	return es.Event{
		AggregateType: CartType,
		At:            c.now(),
		Type:          CartCreated,
		AggregateID:   cartID,
		Data:          map[string]any{},
		VersionID:     1,
	}
}

func (c *CartAggregate) newItemAddedToCartEvent(itemID int) es.Event {
	return es.Event{
		Type:          ItemAddedToCart,
		AggregateType: CartType,
		AggregateID:   c.ID,
		At:            c.now(),
		VersionID:     c.currentVersion + 1,
		Data: map[string]int{
			"item_id": itemID,
		},
	}
}

func (c *CartAggregate) newItemRemovedFromCartEvent(itemID int) es.Event {
	return es.Event{
		Type:          ItemRemovedFromCart,
		AggregateType: CartType,
		AggregateID:   c.ID,
		At:            c.now(),
		VersionID:     c.currentVersion + 1,
		Data: map[string]int{
			"item_id": itemID,
		},
	}
}

func (c *CartAggregate) newCartCheckedOutEvent() es.Event {
	return es.Event{
		Type:          CartCheckedOut,
		AggregateType: CartType,
		AggregateID:   c.ID,
		At:            c.now(),
		VersionID:     c.currentVersion + 1,
		Data:          map[string]any{},
	}
}
