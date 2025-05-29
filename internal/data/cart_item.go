package data

import (
	"database/sql"
	"fmt"
	"time"

	"project/utils/validator"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

type CartItem struct {
	ID        uuid.UUID `db:"id" json:"id"`
	CartID    uuid.UUID `db:"cart_id" json:"cart_id"`
	ProductID uuid.UUID `db:"product_id" json:"product_id"`
	Quantity  int       `db:"quantity" json:"quantity"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CartItemDB struct {
	db DBInterface
}

var cartItemColumns = []string{"id", "cart_id", "product_id", "quantity", "created_at", "updated_at"}

func ValidateCartItem(v *validator.Validator, item *CartItem) {
	v.Check(item.CartID != uuid.Nil, "cart_id", "يجب إدخال معرف السلة")
	v.Check(item.ProductID != uuid.Nil, "product_id", "يجب إدخال معرف المنتج")
	v.Check(item.Quantity > 0, "quantity", "يجب أن تكون الكمية أكبر من 0")
}

func (ci *CartItemDB) Insert(item *CartItem) error {
	item.ID = uuid.New()
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()

	query, args, err := QB.Insert("cart_items").
		Columns(cartItemColumns...).
		Values(item.ID, item.CartID, item.ProductID, item.Quantity, item.CreatedAt, item.UpdatedAt).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	return ci.db.QueryRow(query, args...).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
}

func (ci *CartItemDB) Get(id uuid.UUID) (*CartItem, error) {
	var item CartItem
	query, args, err := QB.Select(cartItemColumns...).From("cart_items").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("error creating query: %v", err)
	}

	err = ci.db.Get(&item, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cart item not found")
		}
		return nil, fmt.Errorf("error getting cart item: %v", err)
	}

	return &item, nil
}

func (ci *CartItemDB) Update(item *CartItem) error {
	item.UpdatedAt = time.Now()

	query, args, err := QB.Update("cart_items").
		SetMap(map[string]interface{}{
			"quantity":   item.Quantity,
			"updated_at": item.UpdatedAt,
		}).
		Where(squirrel.Eq{"id": item.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	_, err = ci.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error updating cart item: %v", err)
	}

	return nil
}

func (ci *CartItemDB) Delete(id uuid.UUID) error {
	query, args, err := QB.Delete("cart_items").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	_, err = ci.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error deleting cart item: %v", err)
	}

	return nil
}

func (ci *CartItemDB) ListByCart(cartID uuid.UUID) ([]CartItem, error) {
	var items []CartItem
	query, args, err := QB.Select(cartItemColumns...).From("cart_items").Where(squirrel.Eq{"cart_id": cartID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("error creating query: %v", err)
	}

	err = ci.db.Select(&items, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error getting cart items: %v", err)
	}

	return items, nil
}
