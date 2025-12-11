package checkout

import (
	"encoding/json"
	"errors"
	"es/internal/es"
	"es/internal/util"
	"slices"
	"time"
)

type CartAggregate struct {
	es.EventSourcedAggregate
	now            util.Timestamp
	ID             int   `json:"cart_id"`
	Contents       []int `json:"contents"`
	CheckedOut     bool  `json:"checked_out"`
	currentVersion int
}

type CartOption func(*CartAggregate)

func UseTimestamp(tp util.Timestamp) CartOption {
	return func(c *CartAggregate) {
		c.now = tp
	}
}

func NewCartAggregate(cartID int, options ...CartOption) *CartAggregate {
	c := &CartAggregate{
		now:        time.Now,
		ID:         cartID,
		Contents:   []int{},
		CheckedOut: false,
	}

	for _, opt := range options {
		opt(c)
	}

	return c
}

func (c *CartAggregate) Init() error {
	return c.Apply(c.newCartCreatedEvent(c.ID))
}

func (c *CartAggregate) Add(itemID int) error {
	if c.CheckedOut {
		return errors.New("cannot add items to a checked out cart")
	}
	return c.Apply(c.newItemAddedToCartEvent(itemID))
}

func (c *CartAggregate) Remove(itemID int) error {
	if c.CheckedOut {
		return errors.New("cannot remove items from a checked out cart")
	}
	if slices.Contains(c.Contents, itemID) {
		return c.Apply(c.newItemRemovedFromCartEvent(itemID))
	}
	return nil
}

func (c *CartAggregate) Checkout() error {
	if c.CheckedOut {
		return errors.New("cart is already checked out")
	}
	return c.Apply(c.newCartCheckedOutEvent())
}

func (c *CartAggregate) Apply(events ...es.Event) error {
	if len(events) == 0 {
		return errors.New("must apply at least 1 event")
	}

	for _, event := range events {
		switch event.Type {
		case CartCreated:
			c.ID = event.AggregateID
		case ItemAddedToCart:
			itemID, err := toItemID(event.Data)
			if err != nil {
				return err
			}
			c.Contents = append(c.Contents, itemID)
		case ItemRemovedFromCart:
			itemID, err := toItemID(event.Data)
			if err != nil {
				return err
			}
			for i, id := range c.Contents {
				if id == itemID {
					c.Contents = append(c.Contents[:i], c.Contents[i+1:]...)
					break
				}
			}
		case CartCheckedOut:
			c.CheckedOut = true
		default:
			return errors.New("not implemented")
		}

	}

	c.currentVersion = events[len(events)-1].VersionID
	c.EventSourcedAggregate.Apply(events...)
	return nil
}

func toItemID(data any) (int, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}
	type itemAddedPayload struct {
		ItemID int `json:"item_id"`
	}
	var payload itemAddedPayload
	if err := json.Unmarshal(dataBytes, &payload); err != nil {
		return 0, err
	}
	if payload.ItemID == 0 {
		return 0, errors.New("invalid or missing item_id")
	}
	return payload.ItemID, nil
}
