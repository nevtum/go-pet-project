## DISCLAIMER: 🚨 DEMO CODE - NOT FOR PRODUCTION USE 🚨

**IMPORTANT:** This is a demonstration project intended solely for educational and learning purposes. This code is:

- NOT production-ready
- NOT secure for real-world deployment
- PROVIDED WITHOUT ANY WARRANTIES
- MEANT ONLY for understanding event sourcing concepts

**By copying this code, you explicitly acknowledge and agree that:**
- This is a proof-of-concept implementation
- The author takes NO RESPONSIBILITY for any issues arising from its use
- It should NEVER be used in any production environment
- Security, performance, and reliability are NOT guaranteed

## Event Sourcing Shopping Cart API

This project implements a shopping cart API using event sourcing and clean architecture principles in Go. It provides a basic structure for managing shopping carts, allowing items to be added, removed, and checked out while maintaining a record of actions taken on the cart.

## Directory Structure

- `internal/api/`: Contains the main API logic.

## Key Components

### CartAggregate

The `CartAggregate` represents the main business logic of the shopping cart. It controls how items are added or removed and manages the checkout process. The aggregate ensures that business rules are enforced (e.g., preventing changes to a checked-out cart).

#### Key Methods:
- `NewCartAggregate(cartID int)`: Initializes a new cart.
- `Add(itemID int)`: Adds an item to the cart.
- `Remove(itemID int)`: Removes an item from the cart.
- `Checkout()`: Finalizes the cart for checkout.

### Events

The API utilizes various event types that represent the changes in the shopping cart state:
- `cart.created`: Triggered when a new cart is created.
- `cart.item_added`: Triggered when an item is added to the cart.
- `cart.item_removed`: Triggered when an item is removed from the cart.
- `cart.checked_out`: Triggered when the cart is checked out.

### Repositories

The `CartRepository` interface defines methods for cart operations, such as creating a new cart, retrieving an existing cart, and saving changes to the cart.

#### PGCartRepository

The `PGCartRepository` is a PostgreSQL implementation that interacts with the database to persist cart information.

### Route Handlers

The API provides several endpoints to interact with the shopping cart:
- `GET /healthz`: Checks the health of the API.
- `GET /cart/{cartID}`: Retrieves the details of a specific cart.
- `GET /cart/{cartID}/{itemID}`: Adds an item to a specific cart.
- `GET /cart/{cartID}/{itemID}/delete`: Removes an item from a specific cart.
- `GET /checkout/{cartID}`: Completes the checkout process for a specific cart.
- `GET /events/{aggType}/{aggID}`: Retrieves events associated with a specific aggregate type and ID.

### Use Cases

The `ShoppingCartUseCase` struct handles the application logic for cart operations, ensuring that the correct repository methods are called in response to user actions.

### Projections

Projections provide a read-optimized view of the data in the event-sourced architecture. They allow for the efficient querying of data by transforming and storing events into a format that's easy to access. Implementing projections can enhance the performance of read operations.

Some potential projections to consider include:

- **Cart Summary Projection**: Maintains a view of the current state of each cart, including items and checkout status.
- **Sales Analytics Projection**: Aggregates data from completed checkouts to provide insights into sales trends and popular items.

Projections can be updated in near real-time as events are processed, ensuring that the views stay consistent with the underlying data state.

## Further ideas
- [x] Write a round-robin load balancer
- [ ] React front end
- [ ] Write unit tests at every layer of the stack
  - [x] aggregate unit tests
  - [ ] route handler unit tests
  - [ ] use case unit tests
  - [ ] repository integration tests
  - [x] projection integration tests
- [ ] Load testing script to simulate high traffic scenarios
- [ ] CI/CD Github Action workflows
- [x] OAuth login
- [x] OAuth Middleware (Cognito)

## Conclusion

This project provides an example of an event-sourced shopping cart API. It demonstrates how aggregates, events, and repositories can be structured to maintain the integrity of the cart across multiple operations, aligning with modern software development principles.
