package main

import (
	"errors"
	"net/http"
	"project/utils"

	"github.com/google/uuid"
)

func (app *application) GetOrderItemHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف عنصر الطلب غير صالح"))
		return
	}

	item, err := app.Model.OrderItemDB.Get(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{"order_item": item})
}

func (app *application) ListOrderItemsHandler(w http.ResponseWriter, r *http.Request) {
	orderIDStr := r.URL.Query().Get("order_id")
	if orderIDStr == "" {
		app.badRequestResponse(w, r, errors.New("يجب إدخال معرف الطلب"))
		return
	}
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف الطلب غير صالح"))
		return
	}

	items, err := app.Model.OrderItemDB.ListByOrder(orderID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{"order_items": items})
}
