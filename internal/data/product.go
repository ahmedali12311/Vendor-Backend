package data

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"project/utils"
	"project/utils/validator"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

type Product struct {
	ID            uuid.UUID `db:"id" json:"id"`
	StoreID       uuid.UUID `db:"store_id" json:"store_id"`
	Name          string    `db:"name" json:"name"`
	Description   *string   `db:"description" json:"description,omitempty"`
	Price         float64   `db:"price" json:"price"`
	Discount      float64   `db:"discount" json:"discount"`
	Image         *string   `db:"image" json:"image,omitempty"`
	StockQuantity int       `db:"stock_quantity" json:"stock_quantity"`
	IsAvailable   bool      `db:"is_available" json:"is_available"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type ProductDB struct {
	db DBInterface
}

func ValidateProduct(v *validator.Validator, product *Product) {
	v.Check(product.Name != "", "name", "يجب ادخال الاسم")
	v.Check(len(product.Name) <= 150, "name", "يجب ألا يزيد عن 150 حرف")
	v.Check(product.StoreID != uuid.Nil, "store_id", "يجب ادخال رقم المتجر")
	v.Check(product.Price >= 0, "price", "يجب أن تكون قيمة إيجابية")
	v.Check(product.Discount >= 0, "discount", "يجب أن تكون الخصم قيمة إيجابية")
	v.Check(product.Discount <= product.Price, "discount", "يجب ألا يتجاوز الخصم السعر")
	v.Check(product.StockQuantity >= 0, "stock_quantity", "الكمية يجب أن تكون رقم غير سالب")

}

func (p *ProductDB) Insert(product *Product) error {
	product.ID = uuid.New()
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	query, args, err := QB.Insert("products").
		Columns(
			"id",
			"store_id",
			"name",
			"description",
			"price",
			"discount",
			"image",
			"stock_quantity",
			"is_available",
			"created_at",
			"updated_at",
		).
		Values(
			product.ID,
			product.StoreID,
			product.Name,
			product.Description,
			product.Price,    // Fixed: Price (NUMERIC) goes here
			product.Discount, // Fixed: Discount (NUMERIC) goes here
			product.Image,    // Fixed: Image (VARCHAR) goes here
			product.StockQuantity,
			product.IsAvailable,
			product.CreatedAt,
			product.UpdatedAt,
		).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	return p.db.QueryRow(query, args...).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)
}

func (p *ProductDB) Get(id uuid.UUID) (*Product, error) {
	var product Product
	query, args, err := QB.Select(products_columns...).From("products").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("error creating query: %v", err)
	}

	err = p.db.Get(&product, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("error getting product: %v", err)
	}

	return &product, nil
}
func (p *ProductDB) Update(product *Product) error {
	product.UpdatedAt = time.Now()

	query, args, err := QB.Update("products").
		SetMap(map[string]interface{}{
			"name":           product.Name,
			"description":    product.Description,
			"image":          product.Image,
			"price":          product.Price,
			"discount":       product.Discount,
			"stock_quantity": product.StockQuantity,
			"is_available":   product.IsAvailable,
			"updated_at":     product.UpdatedAt,
		}).
		Where(squirrel.Eq{"id": product.ID}).
		ToSql()

	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	_, err = p.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error updating product: %v", err)
	}

	return nil
}

func (p *ProductDB) Delete(id uuid.UUID) error {
	query, args, err := QB.Delete("products").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("error creating query: %v", err)
	}

	_, err = p.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("error deleting product: %v", err)
	}

	return nil
}

type ProductWithStore struct {
	Product
	StoreName  string  `db:"store_name" json:"store_name"`
	StoreImage *string `db:"store_image" json:"store_image,omitempty"`
}

func (p *ProductDB) List(queryParams url.Values) ([]ProductWithStore, *utils.Meta, error) {
	var products []ProductWithStore

	joins := []string{
		" stores s ON p.store_id = s.id",
	}

	columns := []string{
		"p.*",
		"s.name as store_name",
		"s.image as store_image",
	}

	meta, err := utils.BuildQuery(&products, "products p", joins, columns, []string{"p.name", "p.description"}, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	return products, meta, nil
}

func (p *ProductDB) GetStoreProducts(storeID uuid.UUID) ([]Product, error) {
	var products []Product
	query, args, err := QB.Select("*").From("products").Where(squirrel.Eq{"store_id": storeID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("error creating query: %v", err)
	}

	err = p.db.Select(&products, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error getting store products: %v", err)
	}

	return products, nil
}
