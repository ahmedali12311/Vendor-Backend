package data

import (
	"fmt"
	"net/url"
	"time"

	"project/utils"
	"project/utils/validator"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

// ------------------------------------
// StoreType Model and Methods
// ------------------------------------

type StoreType struct {
	ID          int       `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type StoreTypeDB struct {
	db *sqlx.DB
}

func ValidateStoreType(v *validator.Validator, storeType *StoreType) {
	v.Check(storeType.Name != "", "name", "يجب إدخال اسم نوع المتجر")
	v.Check(len(storeType.Name) <= 50, "name", "يجب ألا يزيد اسم نوع المتجر عن 50 حرف")
}

func (st *StoreTypeDB) InsertStoreType(storeType *StoreType) error {
	tx, err := st.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	query, args, err := QB.Insert("store_types").
		Columns("name", "description").
		Values(storeType.Name, storeType.Description).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %v", err)
	}

	err = tx.QueryRowx(query, args...).StructScan(storeType)
	if err != nil {
		return fmt.Errorf("failed to insert store type: %v", err)
	}

	return tx.Commit()
}

func (st *StoreTypeDB) GetStoreType(id int) (*StoreType, error) {
	var storeType StoreType

	query, args, err := QB.Select(store_types_columns...).
		From("store_types").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %v", err)
	}

	err = st.db.Get(&storeType, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store type not found: %w", err)
	}

	return &storeType, nil
}

func (st *StoreTypeDB) UpdateStoreType(storeType *StoreType) error {
	tx, err := st.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	storeType.UpdatedAt = time.Now()
	query, args, err := QB.Update("store_types").
		SetMap(map[string]interface{}{
			"name":        storeType.Name,
			"description": storeType.Description,
			"updated_at":  storeType.UpdatedAt,
		}).
		Where(squirrel.Eq{"id": storeType.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %v", err)
	}

	result, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update store type: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify affected rows: %v", err)
	}
	if rowsAffected == 0 {
		return ErrStoreTypeNotFound
	}

	return tx.Commit()
}

func (st *StoreTypeDB) DeleteStoreType(id int) error {
	tx, err := st.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	query, args, err := QB.Delete("store_types").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %v", err)
	}

	result, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete store type: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify affected rows: %v", err)
	}
	if rowsAffected == 0 {
		return ErrStoreTypeNotFound
	}

	return tx.Commit()
}

func (st *StoreTypeDB) ListStoreTypes(queryParams url.Values) ([]StoreType, *utils.Meta, error) {
	var storeTypes []StoreType
	meta, err := utils.BuildQuery(&storeTypes, "store_types", nil, store_types_columns, []string{"name", "description"}, queryParams, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list store types: %v", err)
	}
	return storeTypes, meta, nil
}
