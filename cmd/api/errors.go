package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"project/internal/data"
	"project/utils"
)

func (app *application) handleRetrievalError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, data.ErrRecordNotFound):
		app.errorResponse(w, r, http.StatusNotFound, "السجل غير موجود")
	case errors.Is(err, data.ErrUserNotFound):
		app.errorResponse(w, r, http.StatusNotFound, "المستخدم غير موجود")
	case errors.Is(err, data.ErrEmailAlreadyInserted):
		app.errorResponse(w, r, http.StatusConflict, "البريد الإلكتروني موجود بالفعل")
	case errors.Is(err, data.ErrHasRole):
		app.errorResponse(w, r, http.StatusConflict, "المستخدم لديه دور بالفعل")
	case errors.Is(err, data.ErrProductNotFound):
		app.errorResponse(w, r, http.StatusNotFound, "المنتج غير موجود")
	case errors.Is(err, data.ErrCartNotFound):
		app.errorResponse(w, r, http.StatusNotFound, "السلة غير موجود")
	case errors.Is(err, data.ErrDuplicateEntry):
		app.errorResponse(w, r, http.StatusConflict, "إدخال مكرر")
	case errors.Is(err, data.ErrInvalidInput):
		app.errorResponse(w, r, http.StatusBadRequest, "إدخال غير صالح")
	case errors.Is(err, data.ErrStoreNotFound):
		app.errorResponse(w, r, http.StatusNotFound, "المتجر غير موجود")
	case errors.Is(err, data.ErrStoreTypeNotFound):
		app.errorResponse(w, r, http.StatusNotFound, "نوع المتجر غير موجود")
	case errors.Is(err, data.ErrProductUnavailable):
		app.errorResponse(w, r, http.StatusBadRequest, "المنتج غير متاح")
	case errors.Is(err, data.ErrCartItemNotFound):
		app.errorResponse(w, r, http.StatusNotFound, "عنصر السلة غير موجود")
	case errors.Is(err, data.ErrOrderNotFound):
		app.errorResponse(w, r, http.StatusNotFound, "الطلب غير موجود")
	case errors.Is(err, data.ErrOrderItemNotFound):
		app.errorResponse(w, r, http.StatusNotFound, "عنصر الطلب غير موجود")
	case errors.Is(err, data.ErrAdNotFound):
		app.errorResponse(w, r, http.StatusNotFound, "الإعلان غير موجود")
	case errors.Is(err, data.ErrDuplicatedKey):
		app.errorResponse(w, r, http.StatusConflict, "المفتاح مكرر")
	case errors.Is(err, data.ErrDuplicatedRole):
		app.errorResponse(w, r, http.StatusConflict, "الدور مكرر")
	case errors.Is(err, data.ErrHasNoRoles):
		app.errorResponse(w, r, http.StatusBadRequest, "المستخدم ليس لديه أدوار")
	case errors.Is(err, data.ErrForeignKeyViolation):
		app.errorResponse(w, r, http.StatusBadRequest, "انتهاك قيد المفتاح الخارجي")
	case errors.Is(err, data.ErrUserAlreadyhaveatable):
		app.errorResponse(w, r, http.StatusConflict, "المستخدم لديه جدول بالفعل")
	case errors.Is(err, data.ErrUserHasNoTable):
		app.errorResponse(w, r, http.StatusBadRequest, "المستخدم ليس لديه جدول")
	case errors.Is(err, data.ErrInvalidQuantity):
		app.errorResponse(w, r, http.StatusBadRequest, "الكمية غير صالحة")
	case errors.Is(err, data.ErrRecordNotFoundOrders):
		app.errorResponse(w, r, http.StatusNotFound, "لا توجد طلبات متاحة")
	case errors.Is(err, data.ErrDescriptionMissing):
		app.errorResponse(w, r, http.StatusBadRequest, "الوصف مفقود")
	case errors.Is(err, data.ErrDuplicatedPhone):
		app.errorResponse(w, r, http.StatusConflict, "رقم الهاتف مكرر")
	case errors.Is(err, data.ErrInvalidAddressOrCoordinates):
		app.errorResponse(w, r, http.StatusBadRequest, "العنوان أو الإحداثيات غير صالحة")
	case errors.Is(err, data.ErrInvalidDiscount):
		app.errorResponse(w, r, http.StatusBadRequest, "الخصم غير صالح")
	case errors.Is(err, data.ErrSubscriptionNotFound):
		app.errorResponse(w, r, http.StatusNotFound, "نوع الاشتراك غير موجود")
	case errors.Is(err, data.ErrPhoneAlreadyInserted):
		app.errorResponse(w, r, http.StatusConflict, "رقم الهاتف مسجل مسبقاً")

	default:
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
	env := utils.Envelope{"error": message}
	err := utils.SendJSONResponse(w, status, env)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(500)

	}
}

func (app *application) logError(r *http.Request, err error) {
	log.Printf("Error: %v, Method: %s, URL: %s", err, r.Method, r.URL.String())
}
func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	message := "the server encountered a problem and could not process your request"
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

//	func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
//		message := "resources not found"
//		app.errorResponse(w, r, http.StatusNotFound, message)
//	}
func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (app *application) ErrorHandlerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the error
				app.log.Println("Recovered from panic:", err)

				// Send the error response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				response := ErrorResponse{
					Error: "Internal Server Error",
				}
				json.NewEncoder(w).Encode(response)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
// 	message := "rate limit exceeded"
// 	app.errorResponse(w, r, http.StatusTooManyRequests, message)
// }

func (app *application) jwtErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	var message string
	switch {
	case errors.Is(err, utils.ErrInvalidToken):
		message = "invalid token"
	case errors.Is(err, utils.ErrExpiredToken):
		message = "token has expired"
	case errors.Is(err, utils.ErrMissingToken):
		message = "missing authorization token"
	case errors.Is(err, utils.ErrInvalidClaims):
		message = "invalid token claims"
	default:
		app.errorResponse(w, r, http.StatusUnauthorized, "You don't have a premission")
		return
	}
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}
func (app *application) unauthorizedResponse(w http.ResponseWriter, r *http.Request) {
	message := "unauthorized"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (app *application) forbiddenResponse(w http.ResponseWriter, r *http.Request) {
	message := "you do not have permission to access this resource"
	app.errorResponse(w, r, http.StatusForbidden, message)
}
