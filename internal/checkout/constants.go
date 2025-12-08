package checkout

import (
	"es/internal/es"
)

const (
	CartType es.AggregateType = "cart"

	CartCreated         es.EventType = "cart.created"
	ItemAddedToCart     es.EventType = "cart.item_added"
	ItemRemovedFromCart es.EventType = "cart.item_removed"
	CartCheckedOut      es.EventType = "cart.checked_out"
)
