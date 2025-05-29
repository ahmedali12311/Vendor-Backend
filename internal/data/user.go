package data

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"project/utils"
	"project/utils/validator"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type User struct {
	ID                       uuid.UUID      `db:"id" json:"id"`
	Name                     string         `db:"name" json:"name"`
	Email                    string         `db:"email" json:"email"`
	Password                 string         `db:"password" json:"-"`
	PhoneNumber              string         `db:"phone_number" json:"phone_number"`
	AddressText              *string        `db:"address_text" json:"address_text,omitempty"`
	Latitude                 *float64       `db:"latitude" json:"latitude,omitempty"`
	Longitude                *float64       `db:"longitude" json:"longitude,omitempty"`
	Image                    *string        `db:"image" json:"image,omitempty"`
	Verified                 bool           `db:"verified" json:"verified"`
	CreatedAt                time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt                time.Time      `db:"updated_at" json:"updated_at"`
	VerificationCode         string         `db:"verification_code" json:"-"`
	VerificationCodeExpiry   time.Time      `db:"verification_code_expiry" json:"-"`
	LastVerificationCodeSent time.Time      `db:"last_verification_code_sent" json:"-"`
	Roles                    pq.StringArray `db:"roles" json:"roles"` // from github.com/lib/pq
}

type StringArray []string

// Scan implements the sql.Scanner interface.
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = StringArray{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan StringArray: %v", value)
	}

	str := string(bytes)
	str = str[1 : len(str)-1]
	*s = StringArray(strings.Split(str, ","))

	return nil
}

// UserDB handles database operations related to users.
type UserDB struct {
	db *sqlx.DB
}

func ValidateUser(v *validator.Validator, user *User, fields ...string) {
	for _, field := range fields {
		switch field {
		case "name":
			v.Check(len(user.Name) >= 3, "name", "يجب أن يتكون الاسم من 3 أحرف على الأقل")
			v.Check(user.Name != "", "name", "الاسم مطلوب")
			v.Check(len(user.Name) <= 100, "name", "يجب أن يكون الاسم أقل من 100 حرف")
		case "email":
			v.Check(user.Email != "", "email", "البريد الإلكتروني مطلوب")
			v.Check(validator.Matches(user.Email, validator.GeneralEmailRX), "email", "تنسيق البريد الإلكتروني غير صالح")
		case "phone_number":
			v.Check(user.PhoneNumber != "", "phone_number", "رقم الهاتف مطلوب")
			v.Check(validator.Matches(user.PhoneNumber, validator.PhoneRX), "phone_number", "تنسيق رقم الهاتف غير صالح")
		case "address":
			hasAddressText := user.AddressText != nil && *user.AddressText != ""
			hasCoordinates := user.Latitude != nil && user.Longitude != nil
			v.Check(hasAddressText || hasCoordinates, "address", "يجب إدخال العنوان أو الإحداثيات الجغرافية")
			if hasCoordinates {
				v.Check(*user.Latitude >= -90 && *user.Latitude <= 90, "latitude", "خط العرض يجب أن يكون بين -90 و 90")
				v.Check(*user.Longitude >= -180 && *user.Longitude <= 180, "longitude", "خط الطول يجب أن يكون بين -180 و 180")
			}
		case "password":
			if user.Password != "" {
				v.Check(len(user.Password) >= 8, "password", "كلمة المرور قصيرة جداً")
			}
		}
	}
}

func (u *UserDB) InsertUser(user *User) error {
	// Set expiration time for verification code (24 hours from now)
	user.VerificationCodeExpiry = time.Now().Add(24 * time.Hour)
	user.LastVerificationCodeSent = time.Now()

	query, args, err := QB.Insert("users").
		Columns(
			"name", "email", "password", "phone_number",
			"address_text", "latitude", "longitude", "image",
			"verified", "verification_code_expiry", "verification_code",
			"last_verification_code_sent",
		).
		Values(
			user.Name, user.Email, user.Password, user.PhoneNumber,
			user.AddressText, user.Latitude, user.Longitude, user.Image,
			user.Verified, user.VerificationCodeExpiry, user.VerificationCode,
			user.LastVerificationCodeSent,
		).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("خطأ في إنشاء الاستعلام: %v", err)
	}

	err = u.db.QueryRowx(query, args...).StructScan(user)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Constraint {
			case "users_email_key":
				return ErrEmailAlreadyInserted
			case "users_phone_number_key":
				return ErrPhoneAlreadyInserted
			}
		}
		return fmt.Errorf("خطأ في إضافة المستخدم: %v", err)
	}

	return nil
}

func (u *UserDB) GetUserByEmail(email string) (*User, error) {
	var user User
	query, args, err := QB.Select(users_column...).From("users").Where(squirrel.Eq{"email": email}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("خطأ في إنشاء الاستعلام: %v", err)
	}

	err = u.db.Get(&user, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("خطأ في جلب بيانات المستخدم: %v", err)
	}

	return &user, nil
}

func (u *UserDB) GetUser(userID uuid.UUID) (*User, error) {
	var user User

	query, args, err := QB.Select(
		"u.id", "u.name", "u.email", "u.password", "u.phone_number",
		"u.address_text", "u.latitude", "u.longitude", "u.verified",
		"u.verification_code", "u.verification_code_expiry",
		fmt.Sprintf("CASE WHEN NULLIF(u.image, '') IS NOT NULL THEN CONCAT('%s/', u.image) ELSE NULL END AS image", Domain),
		"u.created_at", "u.updated_at", "u.last_verification_code_sent",
		"COALESCE(ARRAY_AGG(r.name), ARRAY[]::text[]) AS roles",
	).
		From("users u").
		LeftJoin("user_roles ur ON u.id = ur.user_id").
		LeftJoin("roles r ON ur.role_id = r.id").
		Where(squirrel.Eq{"u.id": userID}).
		GroupBy("u.id").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("خطأ في إنشاء الاستعلام: %v", err)
	}

	err = u.db.Get(&user, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("خطأ في جلب بيانات المستخدم: %v", err)
	}

	return &user, nil
}

func (u *UserDB) GetUserByPhoneNumber(phoneNumber string) (*User, error) {
	var user User

	query, args, err := QB.Select(
		"u.id", "u.name", "u.email", "u.password", "u.phone_number",
		"u.address_text", "u.latitude", "u.longitude", "u.verified",
		"u.verification_code", "u.verification_code_expiry",
		fmt.Sprintf("CASE WHEN NULLIF(u.image, '') IS NOT NULL THEN CONCAT('%s/', u.image) ELSE NULL END AS image", Domain),
		"u.created_at", "u.updated_at", "u.last_verification_code_sent",
		"COALESCE(ARRAY_AGG(r.name), ARRAY[]::text[]) AS roles",
	).
		From("users u").
		LeftJoin("user_roles ur ON u.id = ur.user_id").
		LeftJoin("roles r ON ur.role_id = r.id").
		Where(squirrel.Eq{"u.phone_number": phoneNumber}).
		GroupBy("u.id").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("خطأ في إنشاء الاستعلام: %v", err)
	}

	err = u.db.Get(&user, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("خطأ في جلب بيانات المستخدم بواسطة رقم الهاتف: %v", err)
	}

	return &user, nil
}

func (u *UserDB) UpdateUser(user *User) error {
	user.UpdatedAt = time.Now()

	query, args, err := QB.Update("users").
		SetMap(map[string]interface{}{
			"name":                        user.Name,
			"email":                       user.Email,
			"password":                    user.Password,
			"phone_number":                user.PhoneNumber,
			"address_text":                user.AddressText,
			"latitude":                    user.Latitude,
			"longitude":                   user.Longitude,
			"image":                       user.Image,
			"updated_at":                  user.UpdatedAt,
			"verified":                    user.Verified,
			"last_verification_code_sent": user.LastVerificationCodeSent,
			"verification_code":           user.VerificationCode,
			"verification_code_expiry":    user.VerificationCodeExpiry,
		}).
		Where(squirrel.Eq{"id": user.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("خطأ في إنشاء استعلام التحديث: %v", err)
	}

	result, err := u.db.Exec(query, args...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Constraint {
			case "users_email_key":
				return ErrEmailAlreadyInserted
			case "users_phone_number_key":
				return ErrPhoneAlreadyInserted
			}
		}
		return fmt.Errorf("خطأ في تحديث بيانات المستخدم: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("خطأ في التحقق من الصفوف المتأثرة: %v", err)
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (u *UserDB) DeleteUser(userID uuid.UUID) error {
	query, args, err := QB.Delete("users").Where(squirrel.Eq{"id": userID}).ToSql()
	if err != nil {
		return fmt.Errorf("خطأ في إنشاء استعلام الحذف: %v", err)
	}

	_, err = u.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("خطأ في حذف المستخدم: %v", err)
	}
	return nil
}

func (u *UserDB) ListUsers(queryParams url.Values) ([]User, *utils.Meta, error) {
	var users []User

	// Base columns without roles
	baseColumns := []string{
		"u.id", "u.name", "u.email", "u.phone_number",
		"u.address_text", "u.latitude", "u.longitude",
		fmt.Sprintf("CASE WHEN NULLIF(u.image, '') IS NOT NULL THEN CONCAT('%s/', u.image) ELSE NULL END AS image", Domain),
		"u.verified", "u.created_at", "u.updated_at",
	}

	// Columns including roles (using array_agg)
	columnsWithRoles := append(baseColumns,
		"(SELECT ARRAY_AGG(r.name) FROM user_roles ur JOIN roles r ON ur.role_id = r.id WHERE ur.user_id = u.id) AS roles")

	// Searchable columns
	searchCols := []string{"u.name", "u.email", "u.phone_number"}

	// Get users with their roles using a subquery approach
	meta, err := utils.BuildQuery(
		&users,
		"users u",
		nil, // No joins needed since we're using subquery
		columnsWithRoles,
		searchCols,
		queryParams,
		nil, // No additional filters
	)

	if err != nil {
		return nil, nil, fmt.Errorf("خطأ في جلب قائمة المستخدمين: %v", err)
	}

	return users, meta, nil
}
func (u *UserDB) CheckVerificationCodeExpiry(userID uuid.UUID) (bool, error) {
	var user User
	query, args, err := QB.Select("verification_code_expiry").From("users").Where(squirrel.Eq{"id": userID}).ToSql()
	if err != nil {
		return false, fmt.Errorf("خطأ في إنشاء الاستعلام: %v", err)
	}

	err = u.db.Get(&user, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrUserNotFound
		}
		return false, fmt.Errorf("خطأ في التحقق من صلاحية رمز التأكيد: %v", err)
	}

	return !user.VerificationCodeExpiry.Before(time.Now()), nil
}

func (u *UserDB) VerifyUser(userID uuid.UUID, code string) error {
	user, err := u.GetUser(userID)
	if err != nil {
		return err
	}

	if user.VerificationCodeExpiry.Before(time.Now()) {
		return fmt.Errorf("رمز التأكيد انتهت صلاحيته")
	}

	if user.VerificationCode != code {
		return fmt.Errorf("رمز التأكيد خاطئ")
	}

	query, args, err := QB.Update("users").
		Set("verified", true).
		Where(squirrel.Eq{"id": userID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("خطأ في إنشاء استعلام التحقق: %v", err)
	}

	_, err = u.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("خطأ في التحقق من المستخدم: %v", err)
	}

	return nil
}
