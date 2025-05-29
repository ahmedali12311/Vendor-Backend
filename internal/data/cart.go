package data

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

type Cart struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	UserID    uuid.UUID  `db:"user_id" json:"user_id"`
	StoreID   *uuid.UUID `db:"store_id" json:"store_id,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

type CartDB struct {
	db DBInterface
}

func (c *CartDB) Insert(cart *Cart) error {
	cart.ID = uuid.New()
	cart.CreatedAt = time.Now()
	cart.UpdatedAt = time.Now()

	query, args, err := QB.Insert("carts").
		Columns("id", "user_id", "store_id", "created_at", "updated_at").
		Values(cart.ID, cart.UserID, cart.StoreID, cart.CreatedAt, cart.UpdatedAt).
		Suffix("RETURNING id, store_id, created_at, updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	return c.db.QueryRow(query, args...).Scan(&cart.ID, &cart.StoreID, &cart.CreatedAt, &cart.UpdatedAt)
}

func (c *CartDB) Get(id uuid.UUID) (*Cart, error) {
	var cart Cart
	query, args, err := QB.Select(cartColumns...).From("carts").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("error creating query: %v", err)
	}

	err = c.db.Get(&cart, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("error getting cart: %v", err)
	}

	return &cart, nil
}

func (c *CartDB) GetByUser(userID uuid.UUID) (*Cart, error) {
	var cart Cart
	query, args, err := QB.Select(cartColumns...).From("carts").Where(squirrel.Eq{"user_id": userID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("error creating query: %v", err)
	}

	err = c.db.Get(&cart, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting cart: %v", err)
	}

	return &cart, nil
}

func (c *CartDB) Delete(id uuid.UUID) error {
	query, args, err := QB.Delete("carts").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	_, err = c.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error deleting cart: %v", err)
	}

	return nil
}
func (c *CartDB) Update(cart *Cart) error {
	cart.UpdatedAt = time.Now()

	query, args, err := QB.Update("carts").
		Set("store_id", cart.StoreID).
		Set("updated_at", cart.UpdatedAt).
		Where(squirrel.Eq{"id": cart.ID}).
		Suffix("RETURNING store_id, updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	return c.db.QueryRow(query, args...).Scan(&cart.StoreID, &cart.UpdatedAt)
}
