package checkout

import "context"

type CheckoutUseCase struct {
	repository CartRepository
}

func NewCheckoutUseCase(repository CartRepository) *CheckoutUseCase {
	return &CheckoutUseCase{
		repository: repository,
	}
}

func (u *CheckoutUseCase) GetCartDetails(ctx context.Context, cartID int) (*CartAggregate, error) {
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

func (u *CheckoutUseCase) AddItemToCart(ctx context.Context, cartID int, itemID int) (*CartAggregate, error) {
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

func (u *CheckoutUseCase) RemoveItemFromCart(ctx context.Context, cartID int, itemID int) (*CartAggregate, error) {
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

func (u *CheckoutUseCase) Checkout(ctx context.Context, cartID int) (*CartAggregate, error) {
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
