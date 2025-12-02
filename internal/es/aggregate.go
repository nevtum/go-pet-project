package es

type EventSourcedAggregate struct {
	comittedEvents    []Event
	uncommittedEvents []Event
}

func (c *EventSourcedAggregate) Apply(events ...Event) error {
	c.uncommittedEvents = append(c.uncommittedEvents, events...)
	return nil
}

func (c *EventSourcedAggregate) UncommittedEvents() []Event {
	return c.uncommittedEvents
}

func (c *EventSourcedAggregate) Commit() {
	c.comittedEvents = append(c.comittedEvents, c.uncommittedEvents...)
	c.uncommittedEvents = nil
}
