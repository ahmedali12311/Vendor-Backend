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

func (app *application) AddCartItemHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		app.badRequestResponse(w, r, errors.New("معرف المستخدم غير متوفر في السياق"))
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المستخدم غير صالح"))
		return
	}

	productID, err := uuid.Parse(r.FormValue("product_id"))
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المنتج غير صالح"))
		return
	}

	quantity, err := strconv.Atoi(r.FormValue("quantity"))
	if err != nil {
		app.badRequestResponse(w, r, errors.New("الكمية غير صالحة"))
		return
	}

	product, err := app.Model.ProductDB.Get(productID)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}
	if !product.IsAvailable || product.StockQuantity < quantity {
		app.badRequestResponse(w, r, errors.New("المنتج غير متوفر أو الكمية غير كافية"))
		return
	}
	if product.StoreID == uuid.Nil {
		app.badRequestResponse(w, r, errors.New("معرف المتجر للمنتج غير صالح"))
		return
	}

	var cartID uuid.UUID
	cartIDStr := r.FormValue("cart_id")
	var cart *data.Cart
	if cartIDStr != "" {
		cartID, err = uuid.Parse(cartIDStr)
		if err != nil {
			app.badRequestResponse(w, r, errors.New("معرف السلة غير صالح"))
			return
		}
		cart, err = app.Model.CartDB.Get(cartID)
		if err != nil {
			app.handleRetrievalError(w, r, err)
			return
		}
		if cart.UserID != userID {
			app.badRequestResponse(w, r, errors.New("السلة لا تخص المستخدم"))
			return
		}
		if cart.StoreID != nil && *cart.StoreID != product.StoreID {
			app.badRequestResponse(w, r, errors.New("لا يمكن إضافة منتج من متجر مختلف إلى السلة"))
			return
		}
		if cart.StoreID == nil {
			cart.StoreID = &product.StoreID
			err = app.Model.CartDB.Update(cart)
			if err != nil {
				app.serverErrorResponse(w, r, fmt.Errorf("فشل في تحديث معرف المتجر للسلة: %v", err))
				return
			}
		}
	} else {
		cart, err = app.Model.CartDB.GetByUser(userID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		if cart == nil {
			storeID := product.StoreID
			cart = &data.Cart{
				UserID:  userID,
				StoreID: &storeID,
			}
			err = app.Model.CartDB.Insert(cart)
			if err != nil {
				app.serverErrorResponse(w, r, fmt.Errorf("فشل في إنشاء السلة: %v", err))
				return
			}
		} else {
			if cart.StoreID != nil && *cart.StoreID != product.StoreID {
				app.badRequestResponse(w, r, errors.New("لا يمكن إضافة منتج من متجر مختلف إلى السلة"))
				return
			}
			if cart.StoreID == nil {
				cart.StoreID = &product.StoreID
				err = app.Model.CartDB.Update(cart)
				if err != nil {
					app.serverErrorResponse(w, r, fmt.Errorf("فشل في تحديث معرف المتجر للسلة: %v", err))
					return
				}
			}
		}
		cartID = cart.ID
	}

	existingItems, err := app.Model.CartItemDB.ListByCart(cartID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	for _, item := range existingItems {
		if item.ProductID == productID {
			app.badRequestResponse(w, r, errors.New("المنتج موجود بالفعل في السلة"))
			return
		}
	}

	item := &data.CartItem{
		CartID:    cartID,
		ProductID: productID,
		Quantity:  quantity,
	}

	v := validator.New()
	data.ValidateCartItem(v, item)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.Model.CartItemDB.Insert(item)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusCreated, utils.Envelope{
		"message":   "تم إضافة العنصر إلى السلة بنجاح",
		"cart_item": item,
		"cart_id":   cartID,
	})
}

func (app *application) UpdateCartItemHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف عنصر السلة غير صالح"))
		return
	}

	item, err := app.Model.CartItemDB.Get(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	if quantity := r.FormValue("quantity"); quantity != "" {
		if val, err := strconv.Atoi(quantity); err == nil {
			item.Quantity = val
		} else {
			app.badRequestResponse(w, r, errors.New("الكمية غير صالحة"))
			return
		}
	}

	v := validator.New()
	data.ValidateCartItem(v, item)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Verify product stock
	product, err := app.Model.ProductDB.Get(item.ProductID)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}
	if !product.IsAvailable || product.StockQuantity < item.Quantity {
		app.badRequestResponse(w, r, errors.New("المنتج غير متوفر أو الكمية غير كافية"))
		return
	}

	err = app.Model.CartItemDB.Update(item)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"message":   "تم تحديث عنصر السلة بنجاح",
		"cart_item": item,
	})
}

func (app *application) DeleteCartItemHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف عنصر السلة غير صالح"))
		return
	}

	err = app.Model.CartItemDB.Delete(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{"message": "تم حذف عنصر السلة بنجاح"})
}
