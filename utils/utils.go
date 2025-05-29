package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/dgrijalva/jwt-go"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/rand"
)

type Meta struct {
	Total       int `json:"total"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	FirstPage   int `json:"first_page"`
	LastPage    int `json:"last_page"`
	From        int `json:"from"`
	To          int `json:"to"`
}

type Envelope map[string]interface{}

var QB = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
var (
	Domain = os.Getenv("DOMAIN")
)

var db *sqlx.DB

// Initialize the database connection
func SetDB(database *sqlx.DB) {
	db = database
}

var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrExpiredToken  = errors.New("token has expired")
	ErrMissingToken  = errors.New("missing authorization token")
	ErrInvalidClaims = errors.New("invalid token claims")
)

func SendJSONResponse(w http.ResponseWriter, status int, data Envelope) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		return err
	}
	return nil
}

// In utils/json.go
func ReadJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	// Limit the size of the request body
	maxBytes := 1_048_576 // 1MB
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}

	return nil
}
func SaveFile(file io.Reader, table string, filename string) (string, error) {
	// Use filepath.Join which uses OS-specific separators
	fullPath := filepath.Join("uploads", table)

	if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
		log.Printf("Error creating directory: %v", err)
		return "", err
	}

	// Generate new filename
	randomNumber := rand.Intn(1000)
	timestamp := time.Now().Unix()
	ext := filepath.Ext(filename)
	newFileName := fmt.Sprintf("%s_%d_%d%s", filepath.Base(table), timestamp, randomNumber, ext)
	newFilePath := filepath.Join(fullPath, newFileName)

	destFile, err := os.Create(newFilePath)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return "", err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		log.Printf("Error copying file: %v", err)
		return "", err
	}

	// Return the path with forward slashes for URLs, but keep the actual filesystem path with backslashes
	urlPath := "/" + filepath.ToSlash(filepath.Join("uploads", table, newFileName))
	return urlPath, nil
}

func DeleteFile(filePath string) error {
	filePath = strings.TrimPrefix(filePath, "/")
	filePath = filepath.FromSlash(filePath)

	if err := os.Remove(filePath); err != nil {
		log.Printf("Error deleting file %s: %v", filePath, err)
		return fmt.Errorf("could not delete file: %v", err)
	}
	return nil
}
func HashPassword(password string) (string, error) {
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashPassword), nil
}

// for converting string to float
func NormalizeFloatInput(input string) string {
	if strings.Contains(input, ".") {
		parts := strings.Split(input, ".")
		if len(parts[1]) == 0 {
			return input + "0"
		}
	}
	return input + ".0"
}

var jwtSecret = []byte("ahmedpa55wordforitmajormarjcomputerscience")

func GenerateToken(userID string, userRole []string) (string, error) {
	expirationTime := time.Now().Add(time.Hour * 24).Unix() // 24 hours expiration time

	claims := &jwt.MapClaims{
		"id":        userID,
		"user_role": userRole,
		"exp":       expirationTime,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
func SetTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
	})
}
func ValidateToken(tokenString string) (*jwt.Token, error) {
	segments := strings.Split(tokenString, ".")
	if len(segments) != 3 {
		return nil, fmt.Errorf("token contains an invalid number of segments")
	}

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
}

func CheckPassword(storedHash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	return err == nil
}

// ParseBoolOrDefault parses a string into a boolean, or returns a default value if parsing fails.
func ParseBoolOrDefault(value string, defaultValue bool) (bool, error) {
	if value == "" {
		return defaultValue, nil
	}
	return strconv.ParseBool(value)
}

func BuildQuery(dest interface{}, table string,
	joins []string, columns []string,
	searchCols []string, queryParams url.Values,
	additionalFilters []string) (*Meta, error) {

	q := queryParams.Get("q")
	filters := queryParams.Get("filters")
	sort := queryParams.Get("sort")
	page, _ := strconv.Atoi(queryParams.Get("page"))
	perPage, _ := strconv.Atoi(queryParams.Get("per_page"))

	sb := squirrel.Select().PlaceholderFormat(squirrel.Dollar).From(table)

	for _, join := range joins {
		sb = sb.LeftJoin(join)
	}

	if q != "" {
		orConditions := squirrel.Or{}
		for _, col := range searchCols {
			searchStr := fmt.Sprintf("%v", q)
			orConditions = append(orConditions, squirrel.ILike{col: "%" + searchStr + "%"})
		}
		sb = sb.Where(orConditions)
	}

	if filters != "" {
		pairs := strings.Split(filters, ",")
		for _, pair := range pairs {
			parts := strings.Split(pair, ":")
			if len(parts) == 2 {
				sb = sb.Where(squirrel.Eq{parts[0]: parts[1]})
			}
		}
	}

	for _, filter := range additionalFilters {
		sb = sb.Where(filter)
	}

	countSb := sb.Column("COUNT(*)")

	countSQL, countArgs, err := countSb.ToSql()
	if err != nil {
		return nil, err
	}

	var total int
	if err := db.QueryRow(countSQL, countArgs...).Scan(&total); err != nil {
		return nil, err
	}

	sb = sb.Columns(columns...)

	// Add sorting based on the sort parameter
	if sort != "" {
		if strings.HasPrefix(sort, "-") {
			// Descending order
			sb = sb.OrderBy(strings.TrimPrefix(sort, "-") + " DESC")
		} else {
			// Ascending order
			sb = sb.OrderBy(sort + " ASC")
		}
	}

	var offset, lastPage, from, to int
	if page > 0 && perPage > 0 {
		offset = (page - 1) * perPage
		sb = sb.Limit(uint64(perPage)).Offset(uint64(offset))

		// Calculate pagination metadata
		lastPage = (total + perPage - 1) / perPage
		from = offset + 1
		to = offset + perPage
		if to > total {
			to = total
		}
	} else {
		perPage = total
		page = 1
		lastPage = 1
		from = 1
		to = total
	}

	// Generate the SQL query and arguments
	sql, args, err := sb.ToSql()
	if err != nil {
		return nil, err
	}

	// Execute the query with arguments
	if err := db.Select(dest, sql, args...); err != nil {
		return nil, err
	}

	meta := Meta{
		Total:       total,
		PerPage:     perPage,
		CurrentPage: page,
		FirstPage:   1,
		LastPage:    lastPage,
		From:        from,
		To:          to,
	}

	return &meta, nil
}

func GenerateRandomCode() string {
	const digits = "0123456789"

	rand.Seed(uint64(time.Now().UnixNano()))
	code := make([]byte, 6)

	for i := range code {
		randomIndex := rand.Intn(len(digits))

		code[i] = digits[randomIndex]
	}

	return string(code)
}
func StringPointer(s string) *string {
	return &s
}
