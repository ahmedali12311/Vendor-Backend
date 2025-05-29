package data

import (
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	_ "github.com/joho/godotenv/autoload"
)

var (
	ErrDuplicateEntry              = errors.New("duplicate entry")
	ErrInvalidInput                = errors.New("invalid input")
	ErrStoreNotFound               = errors.New("store not found")
	ErrStoreTypeNotFound           = errors.New("store type not found")
	ErrProductNotFound             = errors.New("product not found")
	ErrProductUnavailable          = errors.New("product unavailable")
	ErrCartNotFound                = errors.New("cart not found")
	ErrCartItemNotFound            = errors.New("cart item not found")
	ErrOrderNotFound               = errors.New("order not found")
	ErrOrderItemNotFound           = errors.New("order item not found")
	ErrAdNotFound                  = errors.New("ad not found")
	ErrRecordNotFound              = errors.New("السجل غير موجود")
	ErrDuplicatedKey               = errors.New("المستخدم لديه القيمة بالفعل")
	ErrDuplicatedRole              = errors.New("المستخدم لديه الدور بالفعل")
	ErrHasRole                     = errors.New("المستخدم لديه دور بالفعل")
	ErrHasNoRoles                  = errors.New("المستخدم ليس لديه أدوار")
	ErrForeignKeyViolation         = errors.New("انتهاك قيد المفتاح الخارجي")
	ErrUserNotFound                = errors.New("المستخدم غير موجود")
	ErrUserAlreadyhaveatable       = errors.New("المستخدم لديه جدول بالفعل")
	ErrUserHasNoTable              = errors.New("المستخدم ليس لديه جدول")
	ErrEmailAlreadyInserted        = errors.New("البريد الإلكتروني موجود بالفعل")
	ErrInvalidQuantity             = errors.New("الكمية المطلوبة غير متاحة")
	ErrRecordNotFoundOrders        = errors.New("لا توجد طلبات متاحة")
	ErrDescriptionMissing          = errors.New("الوصف مطلوب")
	ErrDuplicatedPhone             = errors.New("رقم الهاتف موجود بالفعل")
	QB                             = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	Domain                         = "http://localhost:8080"
	ErrInvalidAddressOrCoordinates = errors.New("either address or coordinates must be provided")
	ErrInvalidDiscount             = errors.New("discount must be between 0 and product price")
	ErrSubscriptionNotFound        = errors.New("نوع الاشتراك غير موجود")
	ErrPhoneAlreadyInserted        = errors.New("رقم الهاتف مسجل مسبقاً")

	users_column = []string{
		"id", "name", "email", "password", "phone_number",
		"address_text", "latitude", "longitude",
		fmt.Sprintf("CASE WHEN NULLIF(image, '') IS NOT NULL THEN CONCAT('%s/', image) ELSE NULL END AS image", Domain),
		"verified", "created_at", "updated_at",
		"verification_code", "verification_code_expiry",
		"last_verification_code_sent",
	}

	store_types_columns = []string{
		"id", "name", "description", "created_at", "updated_at",
	}

	stores_columns = []string{
		"id", "owner_id", "store_type_id", "name", "description", "contact_phone", "contact_email",
		fmt.Sprintf("CASE WHEN NULLIF(image, '') IS NOT NULL THEN FORMAT('%s/%%s', image) ELSE NULL END AS image", Domain),
		"address_text", "latitude", "longitude", "is_active",
		"created_at", "updated_at",
	}

	products_columns = []string{
		"id", "store_id", "name", "description", "price", "discount",
		fmt.Sprintf("CASE WHEN NULLIF(image, '') IS NOT NULL THEN FORMAT('%s/%%s', image) ELSE NULL END AS image", Domain),
		"stock_quantity", "is_available", "created_at", "updated_at",
	}

	cartColumns = []string{"id", "user_id", "store_id", "created_at", "updated_at"}

	cart_items_columns = []string{
		"id", "cart_id", "product_id", "quantity", "created_at", "updated_at",
	}
)

type Model struct {
	db          *sqlx.DB
	UserDB      UserDB
	UserRoleDB  UserRoleDB
	StoreTypeDB StoreTypeDB
	StoreDB     StoreDB
	ProductDB   ProductDB
	CartDB      CartDB
	CartItemDB  CartItemDB
	OrderDB     OrderDB
	OrderItemDB OrderItemDB
}

func NewModels(db *sqlx.DB) Model {
	return Model{
		db:          db,
		UserDB:      UserDB{db},
		UserRoleDB:  UserRoleDB{db},
		StoreTypeDB: StoreTypeDB{db},
		StoreDB:     StoreDB{db},
		ProductDB:   ProductDB{db},
		CartDB:      CartDB{db},
		CartItemDB:  CartItemDB{db},
		OrderDB:     OrderDB{db},
		OrderItemDB: OrderItemDB{db},
	}
}
