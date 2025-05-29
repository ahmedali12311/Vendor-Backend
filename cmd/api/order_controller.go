package main

import (
	"errors"
	"fmt"
	"net/http"
	"project/internal/data"
	"project/utils"
	"project/utils/validator"
	"strconv"

	"github.com/google/uuid"
)

func (app *application) CreateOrderFromCartHandler(w http.ResponseWriter, r *http.Request) {
	cartID, err := uuid.Parse(r.FormValue("cart_id"))
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف السلة غير صالح"))
		return
	}

	// Get request values
	deliveryAddress := r.FormValue("delivery_address")
	var deliveryLatitude, deliveryLongitude *float64
	var deliveryNotes *string
	if lat := r.FormValue("delivery_latitude"); lat != "" {
		if val, err := strconv.ParseFloat(lat, 64); err == nil {
			deliveryLatitude = &val
		} else {
			app.badRequestResponse(w, r, errors.New("خطأ في تنسيق خط العرض"))
			return
		}
	}
	if lon := r.FormValue("delivery_longitude"); lon != "" {
		if val, err := strconv.ParseFloat(lon, 64); err == nil {
			deliveryLongitude = &val
		} else {
			app.badRequestResponse(w, r, errors.New("خطأ في تنسيق خط الطول"))
			return
		}
	}
	if notes := r.FormValue("delivery_notes"); notes != "" {
		deliveryNotes = &notes // Fixed typo: was '¬es'
	}

	// Get cart and items
	cart, err := app.Model.CartDB.Get(cartID)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}
	items, err := app.Model.CartItemDB.ListByCart(cartID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	fmt.Printf("Cart items: %+v\n", items) // Debug log
	if len(items) == 0 {
		app.badRequestResponse(w, r, errors.New("السلة فارغة"))
		return
	}

	// If no delivery details provided, fetch from user
	if deliveryAddress == "" && deliveryLatitude == nil && deliveryLongitude == nil {
		user, err := app.Model.UserDB.GetUser(cart.UserID)
		if err != nil {
			app.serverErrorResponse(w, r, fmt.Errorf("خطأ في جلب بيانات المستخدم: %v", err))
			return
		}
		deliveryAddress = ""
		if user.AddressText != nil {
			deliveryAddress = *user.AddressText
		}
		deliveryLatitude = user.Latitude
		deliveryLongitude = user.Longitude
	}

	// Create order
	order := &data.Order{
		UserID:            cart.UserID,
		StoreID:           *cart.StoreID, // From cart, as discussed
		TotalPrice:        0,             // Will be calculated in model
		Status:            "pending",
		DeliveryAddress:   deliveryAddress,
		DeliveryLatitude:  deliveryLatitude,
		DeliveryLongitude: deliveryLongitude,
		DeliveryNotes:     deliveryNotes,
	}

	// Validate order, including address or coordinates constraint
	v := validator.New()
	data.ValidateOrder(v, order)
	// Add custom validation for address or coordinates
	v.Check(order.DeliveryAddress != "" || (order.DeliveryLatitude != nil && order.DeliveryLongitude != nil),
		"delivery_info", "يجب توفير عنوان التوصيل أو كلا من خط العرض وخط الطول")
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Create order from cart via model
	err = app.Model.OrderDB.CreateFromCart(order, cartID, items)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusCreated, utils.Envelope{
		"message": "تم إنشاء الطلب بنجاح",
		"order":   order,
		"items":   items,
	})
}

// Other order handlers remain unchanged
func (app *application) GetOrderHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف الطلب غير صالح"))
		return
	}

	order, err := app.Model.OrderDB.Get(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	items, err := app.Model.OrderItemDB.ListByOrder(order.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"order": order,
		"items": items,
	})
}

func (app *application) UpdateOrderHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف الطلب غير صالح"))
		return
	}

	order, err := app.Model.OrderDB.Get(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	if status := r.FormValue("status"); status != "" {
		order.Status = status
	}
	if deliveryAddress := r.FormValue("delivery_address"); deliveryAddress != "" {
		order.DeliveryAddress = deliveryAddress
	}
	if lat := r.FormValue("delivery_latitude"); lat != "" {
		if val, err := strconv.ParseFloat(lat, 64); err == nil {
			order.DeliveryLatitude = &val
		}
	}
	if lon := r.FormValue("delivery_longitude"); lon != "" {
		if val, err := strconv.ParseFloat(lon, 64); err == nil {
			order.DeliveryLongitude = &val
		}
	}
	if notes := r.FormValue("delivery_notes"); notes != "" {
		order.DeliveryNotes = &notes
	}

	v := validator.New()
	data.ValidateOrder(v, order)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.Model.OrderDB.Update(order)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"message": "تم تحديث الطلب بنجاح",
		"order":   order,
	})
}

func (app *application) ListOrdersHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	orders, meta, err := app.Model.OrderDB.List(queryParams)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"orders": orders,
		"meta":   meta,
	})
}

func (app *application) DeleteOrderHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف الطلب غير صالح"))
		return
	}

	err = app.Model.OrderDB.Delete(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{"message": "تم حذف الطلب بنجاح"})
}
func (app *application) ListStoreOrdersHandler(w http.ResponseWriter, r *http.Request) {
	storeIDStr := r.PathValue("store_id")
	storeID, err := uuid.Parse(storeIDStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المتجر غير صالح"))
		return
	}

	// Get query parameters for filtering, sorting, and pagination
	queryParams := r.URL.Query()
	includeItems := queryParams.Get("include_items") == "true"

	orders, meta, err := app.Model.OrderDB.ListByStore(storeID, queryParams, includeItems)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"orders": orders,
		"meta":   meta,
	})
}
