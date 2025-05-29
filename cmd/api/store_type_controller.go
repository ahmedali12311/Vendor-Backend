package main

import (
	"errors"
	"net/http"
	"project/internal/data"
	"project/utils"
	"project/utils/validator"
	"strconv"
)

func (app *application) CreateStoreTypeHandler(w http.ResponseWriter, r *http.Request) {

	storeType := &data.StoreType{
		Name:        r.FormValue("name"),
		Description: utils.StringPointer(r.FormValue("description")),
	}

	v := validator.New()
	v.Check(storeType.Name != "", "name", "يجب إدخال اسم نوع المتجر")
	v.Check(len(storeType.Name) <= 100, "name", "يجب ألا يزيد اسم نوع المتجر عن 100 حرف")

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err := app.Model.StoreTypeDB.InsertStoreType(storeType)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusCreated, utils.Envelope{
		"message":    "تم إنشاء نوع المتجر بنجاح",
		"store_type": storeType,
	})
}

func (app *application) GetStoreTypeHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف نوع المتجر غير صالح"))
		return
	}

	storeType, err := app.Model.StoreTypeDB.GetStoreType(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{"store_type": storeType})
}

func (app *application) UpdateStoreTypeHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف نوع المتجر غير صالح"))
		return
	}

	storeType, err := app.Model.StoreTypeDB.GetStoreType(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	if err := r.ParseForm(); err != nil {
		app.badRequestResponse(w, r, errors.New("فشل في معالجة البيانات"))
		return
	}

	if name := r.FormValue("name"); name != "" {
		storeType.Name = name
	}
	if description := r.FormValue("description"); description != "" {
		storeType.Description = utils.StringPointer(description)
	}

	v := validator.New()
	v.Check(storeType.Name != "", "name", "يجب إدخال اسم نوع المتجر")
	v.Check(len(storeType.Name) <= 100, "name", "يجب ألا يزيد اسم نوع المتجر عن 100 حرف")

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.Model.StoreTypeDB.UpdateStoreType(storeType)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"message":    "تم تحديث نوع المتجر بنجاح",
		"store_type": storeType,
	})
}

func (app *application) DeleteStoreTypeHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف نوع المتجر غير صالح"))
		return
	}

	err = app.Model.StoreTypeDB.DeleteStoreType(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{"message": "تم حذف نوع المتجر بنجاح"})
}

func (app *application) ListStoreTypesHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	storeTypes, meta, err := app.Model.StoreTypeDB.ListStoreTypes(queryParams)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"store_types": storeTypes,
		"meta":        meta,
	})
}
