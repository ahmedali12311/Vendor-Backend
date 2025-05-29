package data

import (
	"fmt"
	"net/url"
	"time"

	"project/utils"
	"project/utils/validator"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Store struct {
	ID           uuid.UUID `db:"id" json:"id"`
	OwnerID      uuid.UUID `db:"owner_id" json:"owner_id"`
	StoreTypeID  int       `db:"store_type_id" json:"store_type_id"`
	Name         string    `db:"name" json:"name"`
	Description  *string   `db:"description" json:"description,omitempty"`
	ContactPhone string    `db:"contact_phone" json:"contact_phone"`           // New field for contact phone
	ContactEmail *string   `db:"contact_email" json:"contact_email,omitempty"` // New field for contact email
	Image        *string   `db:"image" json:"image,omitempty"`
	AddressText  *string   `db:"address_text" json:"address_text,omitempty"`
	Latitude     *float64  `db:"latitude" json:"latitude,omitempty"`
	Longitude    *float64  `db:"longitude" json:"longitude,omitempty"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}
type StoreDB struct {
	db *sqlx.DB
}

func ValidateStore(v *validator.Validator, store *Store) {
	v.Check(store.Name != "", "name", "يجب إدخال الاسم")
	v.Check(len(store.Name) <= 150, "name", "يجب ألا يزيد عن 150 حرف")
	v.Check(store.OwnerID != uuid.Nil, "owner_id", "يجب إدخال معرف المستخدم")
	v.Check(store.StoreTypeID > 0, "store_type_id", "يجب إدخال نوع المتجر")
	v.Check(validator.Matches(store.ContactPhone, validator.PhoneRX), "contact_phone", "يجب أن يكون رقم هاتف صالح")

	if store.ContactEmail != nil {
		v.Check(validator.Matches(*store.ContactEmail, validator.EmailRX), "contact_email", "يجب أن تكون عنوان بريد إلكتروني صالح")
	}
}

func (s *StoreDB) InsertStore(store *Store) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	query, args, err := QB.Insert("stores").
		Columns(
			"owner_id",
			"store_type_id",
			"name",
			"description",
			"image",
			"contact_phone",
			"contact_email",
			"address_text",
			"latitude",
			"longitude",
			"is_active",
		).
		Values(
			store.OwnerID,
			store.StoreTypeID,
			store.Name,
			store.Description,
			store.Image,
			store.ContactPhone,
			store.ContactEmail,
			store.AddressText,
			store.Latitude,
			store.Longitude,
			store.IsActive,
		).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %v", err)
	}

	err = tx.QueryRowx(query, args...).StructScan(store)
	if err != nil {
		return fmt.Errorf("failed to insert store: %v", err)
	}

	return tx.Commit()
}
func (s *StoreDB) GetStore(id uuid.UUID) (*Store, error) {
	var store Store

	query, args, err := QB.Select(stores_columns...).
		From("stores").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %v", err)
	}

	err = s.db.Get(&store, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store not found: %w", err)
	}

	return &store, nil
}
func (s *StoreDB) UpdateStore(store *Store) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	store.UpdatedAt = time.Now()
	query, args, err := QB.Update("stores").
		SetMap(map[string]interface{}{
			"owner_id":      store.OwnerID,
			"store_type_id": store.StoreTypeID,
			"name":          store.Name,
			"description":   store.Description,
			"image":         store.Image,
			"contact_phone": store.ContactPhone,
			"contact_email": store.ContactEmail,
			"address_text":  store.AddressText,
			"latitude":      store.Latitude,
			"longitude":     store.Longitude,
			"is_active":     store.IsActive,
			"updated_at":    store.UpdatedAt,
		}).
		Where(squirrel.Eq{"id": store.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %v", err)
	}

	result, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update store: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify affected rows: %v", err)
	}
	if rowsAffected == 0 {
		return ErrStoreNotFound
	}

	return tx.Commit()
}
func (s *StoreDB) DeleteStore(id uuid.UUID) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	query, args, err := QB.Delete("stores").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %v", err)
	}

	result, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete store: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify affected rows: %v", err)
	}
	if rowsAffected == 0 {
		return ErrStoreNotFound
	}

	return tx.Commit()
}

func (s *StoreDB) ListStores(queryParams url.Values) ([]Store, *utils.Meta, error) {
	additionalFilters := []string{}
	if ownerID := queryParams.Get("owner_id"); ownerID != "" {
		additionalFilters = append(additionalFilters, fmt.Sprintf("owner_id = '%s'", ownerID))
	}

	var stores []Store
	meta, err := utils.BuildQuery(&stores, "stores", nil, stores_columns, []string{"name", "description"}, queryParams, additionalFilters)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list stores: %v", err)
	}
	return stores, meta, nil
}
