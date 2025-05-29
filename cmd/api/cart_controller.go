package main

import (
	"errors"
	"fmt"
	"net/http"
	"project/internal/data"
	"project/utils"

	"github.com/google/uuid"
)

func (app *application) CreateCartHandler(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from request context
	userIDStr, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		app.badRequestResponse(w, r, errors.New("unauthorized: معرف المستخدم غير متوفر"))
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المستخدم غير صالح"))
		return
	}

	// Parse optional store_id from form
	var storeID *uuid.UUID
	if storeIDStr := r.FormValue("store_id"); storeIDStr != "" {
		parsedID, err := uuid.Parse(storeIDStr)
		if err != nil {
			app.badRequestResponse(w, r, errors.New("معرف المتجر غير صالح"))
			return
		}
		storeID = &parsedID
	}

	cart := &data.Cart{
		UserID:  userID,
		StoreID: storeID,
	}

	// Validate cart

	// Check if user already has a cart (optionally with the same store_id)
	existingCart, err := app.Model.CartDB.GetByUser(userID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if existingCart != nil {
		if storeID == nil || (existingCart.StoreID != nil && *existingCart.StoreID == *storeID) {
			app.badRequestResponse(w, r, errors.New("لدى المستخدم سلة بالفعل"))
			return
		}
	}

	err = app.Model.CartDB.Insert(cart)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusCreated, utils.Envelope{
		"message": "تم إنشاء السلة بنجاح",
		"cart":    cart,
	})
}

func (app *application) GetUserCartHandler(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from request context
	userIDStr, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		app.badRequestResponse(w, r, errors.New("unauthorized: معرف المستخدم غير متوفر"))
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المستخدم غير صالح"))
		return
	}

	// Get user's cart
	cart, err := app.Model.CartDB.GetByUser(userID)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}
	if cart == nil {
		app.badRequestResponse(w, r, errors.New("لا توجد سلة للمستخدم"))
		return
	}

	// Get cart items
	items, err := app.Model.CartItemDB.ListByCart(cart.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	fmt.Println("Cart ID:", cart)

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"user_id": userID,
		"cart":    cart,
		"items":   items,
	})
}

func (app *application) GetCartHandler(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from request context
	userIDStr, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		app.badRequestResponse(w, r, errors.New("unauthorized: معرف المستخدم غير متوفر"))
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المستخدم غير صالح"))
		return
	}

	// Parse cart ID from path
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف السلة غير صالح"))
		return
	}

	// Get cart
	cart, err := app.Model.CartDB.Get(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	// Validate cart ownership
	if cart.UserID != userID {
		app.forbiddenResponse(w, r)
		return
	}

	// Get cart items
	items, err := app.Model.CartItemDB.ListByCart(cart.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"cart":  cart,
		"items": items,
	})
}

func (app *application) DeleteCartHandler(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from request context
	userIDStr, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		app.badRequestResponse(w, r, errors.New("unauthorized: معرف المستخدم غير متوفر"))
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المستخدم غير صالح"))
		return
	}

	// Parse cart ID from path
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف السلة غير صالح"))
		return
	}

	// Get cart to validate ownership
	cart, err := app.Model.CartDB.Get(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}
	if cart.UserID != userID {
		app.forbiddenResponse(w, r)
		return
	}

	// Delete cart
	err = app.Model.CartDB.Delete(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"message": "تم حذف السلة بنجاح",
	})
}
