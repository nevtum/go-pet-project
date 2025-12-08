package checkout

import "context"

type ShoppingCartUseCase struct {
	repository CartRepository
}

func NewShoppingCartUseCase(repository CartRepository) *ShoppingCartUseCase {
	return &ShoppingCartUseCase{
		repository: repository,
	}
}

func (u *ShoppingCartUseCase) GetCartDetails(ctx context.Context, cartID int) (*CartAggregate, error) {
	cart, err := u.repository.Get(ctx, cartID)

	if err != nil {
		return nil, err
	}

	if cart == nil {
		newCart, err := u.repository.New(ctx, cartID)
		if err != nil {
			return nil, err
		}
		cart = newCart
	}

	return cart, nil
}

func (u *ShoppingCartUseCase) AddItemToCart(ctx context.Context, cartID int, itemID int) (*CartAggregate, error) {
	cart, err := u.repository.Get(ctx, cartID)

	if err != nil {
		return nil, err
	}

	if err := cart.Add(itemID); err != nil {
		return nil, err
	}

	if err := u.repository.Save(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (u *ShoppingCartUseCase) RemoveItemFromCart(ctx context.Context, cartID int, itemID int) (*CartAggregate, error) {
	cart, err := u.repository.Get(ctx, cartID)

	if err != nil {
		return nil, err
	}

	if err := cart.Remove(itemID); err != nil {
		return nil, err
	}

	if err := u.repository.Save(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (u *ShoppingCartUseCase) Checkout(ctx context.Context, cartID int) (*CartAggregate, error) {
	cart, err := u.repository.Get(ctx, cartID)

	if err != nil {
		return nil, err
	}

	if err := cart.Checkout(); err != nil {
		return nil, err
	}

	if err := u.repository.Save(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}
