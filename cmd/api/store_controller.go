package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"project/internal/data"
	"project/utils"
	"project/utils/validator"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Store Handlers
func (app *application) CreateStoreHandler(w http.ResponseWriter, r *http.Request) {

	ownerEmail := r.FormValue("owner_email")
	user, err := app.Model.UserDB.GetUserByEmail(ownerEmail)
	if err != nil {
		if errors.Is(err, data.ErrUserNotFound) {
			app.badRequestResponse(w, r, errors.New("البريد الإلكتروني للمالك غير موجود"))
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	storeTypeID, err := strconv.Atoi(r.FormValue("store_type_id"))
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف نوع المتجر غير صالح"))
		return
	}

	store := &data.Store{
		OwnerID:      user.ID,
		StoreTypeID:  storeTypeID,
		Name:         r.FormValue("name"),
		Description:  utils.StringPointer(r.FormValue("description")),
		ContactPhone: r.FormValue("contact_phone"),
		ContactEmail: utils.StringPointer(r.FormValue("contact_email")),
		IsActive:     true,
	}
	if address := r.FormValue("address_text"); address != "" {
		store.AddressText = &address
	}
	if latStr := r.FormValue("latitude"); latStr != "" {
		if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
			store.Latitude = &lat
		}
	}
	if longStr := r.FormValue("longitude"); longStr != "" {
		if long, err := strconv.ParseFloat(longStr, 64); err == nil {
			store.Longitude = &long
		}
	}

	if file, fileHeader, err := r.FormFile("image"); err == nil {
		defer file.Close()
		imageName, err := utils.SaveFile(file, "stores", fileHeader.Filename)
		if err != nil {
			app.errorResponse(w, r, http.StatusBadRequest, "صورة غير صالحة")
			return
		}
		store.Image = &imageName
	}

	v := validator.New()
	data.ValidateStore(v, store)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.Model.StoreDB.InsertStore(store)
	if err != nil {
		_, storeTypeErr := app.Model.StoreTypeDB.GetStoreType(storeTypeID)
		if storeTypeErr != nil {
			app.badRequestResponse(w, r, errors.New("معرف نوع المتجر غير موجود"))
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusCreated, utils.Envelope{
		"message": "تم إنشاء المتجر بنجاح",
		"store":   store,
	})
}

func (app *application) GetStoreHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المتجر غير صالح"))
		return
	}

	store, err := app.Model.StoreDB.GetStore(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{"store": store})
}
func (app *application) UpdateStoreHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المتجر غير صالح"))
		return
	}

	store, err := app.Model.StoreDB.GetStore(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	// Parse form data
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, "فشل في معالجة البيانات")
		return
	}

	// Update store fields if provided
	if name := r.FormValue("name"); name != "" {
		store.Name = name
	}
	if description := r.FormValue("description"); description != "" {
		store.Description = &description
	} else if r.FormValue("description") == "" {
		store.Description = nil
	}
	if contactPhone := r.FormValue("contact_phone"); contactPhone != "" {
		store.ContactPhone = contactPhone

	} else if r.FormValue("contact_phone") == "" {
		store.ContactPhone = ""
	}
	if contactEmail := r.FormValue("contact_email"); contactEmail != "" {
		store.ContactEmail = &contactEmail
	} else if r.FormValue("contact_email") == "" {
		store.ContactEmail = nil
	}
	if address := r.FormValue("address_text"); address != "" {
		store.AddressText = &address
	} else if r.FormValue("address_text") == "" {
		store.AddressText = nil
	}
	if latStr := r.FormValue("latitude"); latStr != "" {
		if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
			store.Latitude = &lat
		}
	} else if r.FormValue("latitude") == "" {
		store.Latitude = nil
	}
	if longStr := r.FormValue("longitude"); longStr != "" {
		if long, err := strconv.ParseFloat(longStr, 64); err == nil {
			store.Longitude = &long
		}
	} else if r.FormValue("longitude") == "" {
		store.Longitude = nil
	}
	if isActiveStr := r.FormValue("is_active"); isActiveStr != "" {
		store.IsActive = isActiveStr == "true"
	}
	if storeTypeIDStr := r.FormValue("store_type_id"); storeTypeIDStr != "" {
		storeTypeID, err := strconv.Atoi(storeTypeIDStr)
		if err != nil || storeTypeID <= 0 {
			app.badRequestResponse(w, r, errors.New("معرف نوع المتجر غير صالح"))
			return
		}
		store.StoreTypeID = storeTypeID
	}
	if ownerEmail := r.FormValue("owner_email"); ownerEmail != "" {
		user, err := app.Model.UserDB.GetUserByEmail(ownerEmail)
		if err != nil {
			if errors.Is(err, data.ErrUserNotFound) {
				app.badRequestResponse(w, r, errors.New("البريد الإلكتروني غير موجود"))
				return
			}
			app.serverErrorResponse(w, r, err)
			return
		}
		store.OwnerID = user.ID
	}

	if store.Image != nil {
		*store.Image = strings.TrimPrefix(*store.Image, data.Domain+"/")
	}
	removeImage := r.FormValue("remove_image") == "true"
	file, fileHeader, err := r.FormFile("image")
	if removeImage {
		if store.Image != nil {
			if err := utils.DeleteFile(strings.TrimPrefix(*store.Image, data.Domain+"/")); err != nil {
				app.serverErrorResponse(w, r, fmt.Errorf("فشل في حذف الصورة: %v", err))
				return
			}
			store.Image = nil
		}
	} else if err == nil && file != nil {
		defer file.Close()
		newFileName, err := utils.SaveFile(file, "stores", fileHeader.Filename)
		if err != nil {
			app.errorResponse(w, r, http.StatusBadRequest, fmt.Sprintf("فشل في حفظ الصورة: %v", err))
			return
		}
		if store.Image != nil {
			if err := utils.DeleteFile(strings.TrimPrefix(*store.Image, data.Domain+"/")); err != nil {
				log.Printf("failed to delete old image: %v", err)
			}
		}
		store.Image = &newFileName
	} else if err != nil && !errors.Is(err, http.ErrMissingFile) {
		app.errorResponse(w, r, http.StatusBadRequest, fmt.Sprintf("خطأ في معالجة ملف الصورة: %v", err))
		return
	}

	v := validator.New()
	data.ValidateStore(v, store)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.Model.StoreDB.UpdateStore(store)
	if err != nil {
		// Check for invalid store_type_id
		_, storeTypeErr := app.Model.StoreTypeDB.GetStoreType(store.StoreTypeID)
		if storeTypeErr != nil {
			app.badRequestResponse(w, r, errors.New("معرف نوع المتجر غير موجود"))
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"message": "تم تحديث المتجر بنجاح",
		"store":   store,
	})
}
func (app *application) DeleteStoreHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المتجر غير صالح"))
		return
	}

	store, err := app.Model.StoreDB.GetStore(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	err = app.Model.StoreDB.DeleteStore(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if store.Image != nil {
		utils.DeleteFile(*store.Image)
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{"message": "تم حذف المتجر بنجاح"})
}

func (app *application) ListStoresHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	stores, meta, err := app.Model.StoreDB.ListStores(queryParams)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"stores": stores,
		"meta":   meta,
	})
}
