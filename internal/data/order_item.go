package data

import (
	"database/sql"
	"fmt"
	"time"

	"project/utils/validator"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

type OrderItem struct {
	ID           uuid.UUID `db:"id" json:"id"`
	OrderID      uuid.UUID `db:"order_id" json:"order_id"`
	ProductID    uuid.UUID `db:"product_id" json:"product_id"`
	Quantity     int       `db:"quantity" json:"quantity"`
	PriceAtOrder float64   `db:"price_at_order" json:"price_at_order"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type OrderItemDB struct {
	db DBInterface
}

var orderItemColumns = []string{"id", "order_id", "product_id", "quantity", "price_at_order", "created_at"}

func ValidateOrderItem(v *validator.Validator, item *OrderItem) {
	v.Check(item.OrderID != uuid.Nil, "order_id", "يجب إدخال معرف الطلب")
	v.Check(item.ProductID != uuid.Nil, "product_id", "يجب إدخال معرف المنتج")
	v.Check(item.Quantity > 0, "quantity", "يجب أن تكون الكمية أكبر من 0")
	v.Check(item.PriceAtOrder >= 0, "price_at_order", "يجب أن يكون السعر عند الطلب إيجابيًا")
}

func (oi *OrderItemDB) Insert(item *OrderItem) error {
	item.ID = uuid.New()
	item.CreatedAt = time.Now()

	query, args, err := QB.Insert("order_items").
		Columns(orderItemColumns...).
		Values(item.ID, item.OrderID, item.ProductID, item.Quantity, item.PriceAtOrder, item.CreatedAt).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	return oi.db.QueryRow(query, args...).Scan(&item.ID, &item.CreatedAt)
}

func (oi *OrderItemDB) Get(id uuid.UUID) (*OrderItem, error) {
	var item OrderItem
	query, args, err := QB.Select(orderItemColumns...).From("order_items").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("error creating query: %v", err)
	}

	err = oi.db.Get(&item, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order item not found")
		}
		return nil, fmt.Errorf("error getting order item: %v", err)
	}

	return &item, nil
}

func (oi *OrderItemDB) ListByOrder(orderID uuid.UUID) ([]OrderItem, error) {
	var items []OrderItem
	query, args, err := QB.Select(orderItemColumns...).From("order_items").Where(squirrel.Eq{"order_id": orderID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("error creating query: %v", err)
	}

	err = oi.db.Select(&items, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error getting order items: %v", err)
	}

	return items, nil
}
