package data

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"time"

	"project/utils"
	"project/utils/validator"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Order struct {
	ID                uuid.UUID `db:"id" json:"id"`
	UserID            uuid.UUID `db:"user_id" json:"user_id"`
	StoreID           uuid.UUID `db:"store_id" json:"store_id"`
	TotalPrice        float64   `db:"total_price" json:"total_price"`
	Status            string    `db:"status" json:"status"`
	DeliveryAddress   string    `db:"delivery_address" json:"delivery_address"`
	DeliveryLatitude  *float64  `db:"delivery_latitude" json:"delivery_latitude,omitempty"`
	DeliveryLongitude *float64  `db:"delivery_longitude" json:"delivery_longitude,omitempty"`
	DeliveryNotes     *string   `db:"delivery_notes" json:"delivery_notes,omitempty"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

type OrderDB struct {
	db DBInterface
}

var orderColumns = []string{
	"id", "user_id", "store_id", "total_price", "status",
	"delivery_address", "delivery_latitude", "delivery_longitude", "delivery_notes",
	"created_at", "updated_at",
}

func ValidateOrder(v *validator.Validator, order *Order) {
	v.Check(order.UserID != uuid.Nil, "user_id", "يجب إدخال معرف المستخدم")
	v.Check(order.StoreID != uuid.Nil, "store_id", "يجب إدخال معرف المتجر")
	v.Check(order.TotalPrice >= 0, "total_price", "يجب أن يكون السعر الإجمالي إيجابيًا")
	v.Check(order.Status != "", "status", "يجب إدخال حالة الطلب")
	v.Check(validator.In(order.Status, "pending", "processing", "shipped", "delivered", "cancelled"), "status", "حالة الطلب غير صالحة")
	v.Check(order.DeliveryAddress != "", "delivery_address", "يجب إدخال عنوان التوصيل")
}

func (o *OrderDB) Insert(order *Order) error {
	order.ID = uuid.New()
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	query, args, err := QB.Insert("orders").
		Columns(orderColumns...).
		Values(order.ID, order.UserID, order.StoreID, order.TotalPrice, order.Status,
			order.DeliveryAddress, order.DeliveryLatitude, order.DeliveryLongitude, order.DeliveryNotes,
			order.CreatedAt, order.UpdatedAt).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	return o.db.QueryRow(query, args...).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
}

func (o *OrderDB) Get(id uuid.UUID) (*Order, error) {
	var order Order
	query, args, err := QB.Select(orderColumns...).From("orders").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("error creating query: %v", err)
	}

	err = o.db.Get(&order, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("error getting order: %v", err)
	}

	return &order, nil
}

func (o *OrderDB) Update(order *Order) error {
	order.UpdatedAt = time.Now()

	query, args, err := QB.Update("orders").
		SetMap(map[string]interface{}{
			"status":             order.Status,
			"delivery_address":   order.DeliveryAddress,
			"delivery_latitude":  order.DeliveryLatitude,
			"delivery_longitude": order.DeliveryLongitude,
			"delivery_notes":     order.DeliveryNotes,
			"updated_at":         order.UpdatedAt,
		}).
		Where(squirrel.Eq{"id": order.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	_, err = o.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error updating order: %v", err)
	}

	return nil
}

func (o *OrderDB) Delete(id uuid.UUID) error {
	query, args, err := QB.Delete("orders").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	_, err = o.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error deleting order: %v", err)
	}

	return nil
}

func (o *OrderDB) List(queryParams url.Values) ([]OrderWithItems, *utils.Meta, error) {
	var orders []OrderWithItems
	joins := []string{
		" stores s ON orders.store_id = s.id",
	}
	columns := []string{
		"orders.*",
		"s.name as store_name",
	}
	meta, err := utils.BuildQuery(&orders, "orders", joins, columns, []string{"delivery_address"}, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	return orders, meta, nil
}

func (o *OrderDB) ListByUser(userID uuid.UUID) ([]Order, error) {
	var orders []Order
	query, args, err := QB.Select(orderColumns...).From("orders").Where(squirrel.Eq{"user_id": userID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("error creating query: %v", err)
	}

	err = o.db.Select(&orders, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error getting orders: %v", err)
	}

	return orders, nil
}

type OrderWithItems struct {
	ID                uuid.UUID   `db:"id" json:"id"`
	UserID            uuid.UUID   `db:"user_id" json:"user_id"`
	StoreID           uuid.UUID   `db:"store_id" json:"store_id"`
	TotalPrice        float64     `db:"total_price" json:"total_price"`
	Status            string      `db:"status" json:"status"`
	DeliveryAddress   string      `db:"delivery_address" json:"delivery_address"`
	DeliveryLatitude  *float64    `db:"delivery_latitude" json:"delivery_latitude,omitempty"`
	DeliveryLongitude *float64    `db:"delivery_longitude" json:"delivery_longitude,omitempty"`
	DeliveryNotes     *string     `db:"delivery_notes" json:"delivery_notes,omitempty"`
	CreatedAt         time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time   `db:"updated_at" json:"updated_at"`
	StoreName         string      `db:"store_name" json:"store_name,omitempty"`
	Items             []OrderItem `db:"-" json:"items,omitempty"`
}

func (o *OrderDB) ListByStore(storeID uuid.UUID, queryParams url.Values, includeItems bool) ([]OrderWithItems, *utils.Meta, error) {
	var orders []OrderWithItems
	joins := []string{
		" stores s ON orders.store_id = s.id",
	}
	columns := []string{
		"orders.id",
		"orders.user_id",
		"orders.store_id",
		"orders.total_price",
		"orders.status",
		"orders.delivery_address",
		"orders.delivery_latitude",
		"orders.delivery_longitude",
		"orders.delivery_notes",
		"orders.created_at",
		"orders.updated_at",
		"s.name as store_name",
	}
	searchCols := []string{"orders.delivery_address"}
	additionalFilters := []string{}

	meta, err := utils.BuildQuery(&orders, "orders", joins, columns, searchCols, queryParams, additionalFilters)
	if err != nil {
		return nil, nil, fmt.Errorf("error building query for store %s: %v", storeID, err)
	}

	if includeItems {
		var orderItems []OrderItem
		query, args, err := QB.Select(
			"oi.id",
			"oi.order_id",
			"oi.product_id",
			"oi.quantity",
			"oi.price_at_order",
			"oi.created_at",
		).From("order_items oi").
			Join("orders o ON oi.order_id = o.id").
			Where(squirrel.Eq{"o.store_id": storeID}).
			ToSql()

		if err != nil {
			return nil, nil, fmt.Errorf("error creating items query: %v", err)
		}
		err = o.db.Select(&orderItems, query, args...)
		if err != nil {
			log.Printf("Error getting order items: %v", err)
			return nil, nil, fmt.Errorf("error getting order items: %v", err)
		}

		itemMap := make(map[uuid.UUID][]OrderItem)
		for _, item := range orderItems {
			itemMap[item.OrderID] = append(itemMap[item.OrderID], item)
		}

		for i := range orders {
			orders[i].Items = itemMap[orders[i].ID]
			log.Printf("Order %s Items: %v", orders[i].ID, orders[i].Items)
		}
	}
	return orders, meta, nil
}
func (o *OrderDB) CreateFromCart(order *Order, cartID uuid.UUID, cartItems []CartItem) error {
	// Calculate total price and validate stock
	var totalPrice float64
	productDB := &ProductDB{db: o.db}
	for _, item := range cartItems {
		product, err := productDB.Get(item.ProductID)
		if err != nil {
			return fmt.Errorf("error getting product %s: %v", item.ProductID, err)
		}
		if !product.IsAvailable || product.StockQuantity < item.Quantity {
			return fmt.Errorf("product %s is not available or insufficient stock", item.ProductID)
		}
		totalPrice += (product.Price - product.Discount) * float64(item.Quantity)
	}
	order.TotalPrice = totalPrice

	// Start transaction
	tx, err := o.db.(*sqlx.DB).Beginx()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback()

	// Create transaction-bound model instances
	orderDB := &OrderDB{db: tx}
	orderItemDB := &OrderItemDB{db: tx}
	productDBTx := &ProductDB{db: tx}
	cartItemDB := &CartItemDB{db: tx}
	cartDB := &CartDB{db: tx}

	// Insert order
	err = orderDB.Insert(order)
	if err != nil {
		return fmt.Errorf("error inserting order: %v", err)
	}

	// Create order items
	for _, item := range cartItems {
		product, err := productDB.Get(item.ProductID) // Use non-tx productDB for read-only
		if err != nil {
			return fmt.Errorf("error getting product %s: %v", item.ProductID, err)
		}
		orderItem := &OrderItem{
			ID:           uuid.New(),
			OrderID:      order.ID,
			ProductID:    item.ProductID,
			Quantity:     item.Quantity,
			PriceAtOrder: product.Price - product.Discount,
			CreatedAt:    time.Now(),
		}
		err = orderItemDB.Insert(orderItem)
		if err != nil {
			return fmt.Errorf("error inserting order item: %v", err)
		}

		// Update product stock
		product.StockQuantity -= item.Quantity
		err = productDBTx.Update(product)
		if err != nil {
			return fmt.Errorf("error updating product stock: %v", err)
		}
	}

	// Clear cart items
	for _, item := range cartItems {
		err = cartItemDB.Delete(item.ID)
		if err != nil {
			return fmt.Errorf("error deleting cart item %s: %v", item.ID, err)
		}
	}

	// Delete the cart
	err = cartDB.Delete(cartID)
	if err != nil {
		return fmt.Errorf("error deleting cart %s: %v", cartID, err)
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}
