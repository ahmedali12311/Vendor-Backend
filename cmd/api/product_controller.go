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

func (app *application) CreateProductHandler(w http.ResponseWriter, r *http.Request) {

	storeID, err := uuid.Parse(r.FormValue("store_id"))
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المتجر غير صالح"))
		return
	}

	priceStr := r.FormValue("price")
	if priceStr == "" {
		app.badRequestResponse(w, r, errors.New("يجب إدخال السعر"))
		return
	}
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("السعر يجب أن يكون رقمًا صالحًا (مثال: 30 أو 40.50)"))
		return
	}
	if price < 0 {
		app.badRequestResponse(w, r, errors.New("السعر يجب أن يكون غير سالب"))
		return
	}

	discountStr := r.FormValue("discount")
	discount := 0.0
	if discountStr != "" {
		discount, err = strconv.ParseFloat(discountStr, 64)
		if err != nil {
			app.badRequestResponse(w, r, errors.New("الخصم يجب أن يكون رقمًا صالحًا (مثال: 0 أو 5.50)"))
			return
		}
		if discount < 0 {
			app.badRequestResponse(w, r, errors.New("الخصم يجب أن يكون غير سالب"))
			return
		}
	}

	stockQuantityStr := r.FormValue("stock_quantity")
	stockQuantity := 0
	if stockQuantityStr != "" {
		stockQuantity, err = strconv.Atoi(stockQuantityStr)
		if err != nil {
			app.badRequestResponse(w, r, errors.New("كمية المخزون يجب أن تكون عددًا صحيحًا صالحًا (مثال: 0 أو 100)"))
			return
		}
		if stockQuantity < 0 {
			app.badRequestResponse(w, r, errors.New("كمية المخزون يجب أن تكون غير سالبة"))
			return
		}
	}

	isAvailable := true
	if available := r.FormValue("is_available"); available != "" {
		isAvailable = available == "true"
	}

	product := &data.Product{
		StoreID:       storeID,
		Name:          r.FormValue("name"),
		Description:   utils.StringPointer(r.FormValue("description")),
		Price:         price,
		Discount:      discount,
		StockQuantity: stockQuantity,
		IsAvailable:   isAvailable,
	}

	if file, fileHeader, err := r.FormFile("image"); err == nil {
		defer file.Close()
		imageName, err := utils.SaveFile(file, "products", fileHeader.Filename)
		if err != nil {
			app.errorResponse(w, r, http.StatusBadRequest, "صورة غير صالحة")
			return
		}
		product.Image = &imageName
	}

	v := validator.New()
	data.ValidateProduct(v, product)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.Model.ProductDB.Insert(product)
	if err != nil {
		if product.Image != nil {
			if err := utils.DeleteFile(*product.Image); err != nil {
				log.Printf("Failed to delete image %s: %v", *product.Image, err)
			}
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusCreated, utils.Envelope{
		"message": "تم إنشاء المنتج بنجاح",
		"product": product,
	})
}

func (app *application) GetProductHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المنتج غير صالح"))
		return
	}

	product, err := app.Model.ProductDB.Get(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{"product": product})
}
func (app *application) UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المنتج غير صالح"))
		return
	}

	product, err := app.Model.ProductDB.Get(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	// Store the old image path and trim domain prefix if present
	oldImage := product.Image
	if oldImage != nil {
		*oldImage = strings.TrimPrefix(*oldImage, data.Domain+"/")
	}

	if name := r.FormValue("name"); name != "" {
		product.Name = name
	}
	if description := r.FormValue("description"); description != "" {
		product.Description = &description
	}
	if price := r.FormValue("price"); price != "" {
		val, err := strconv.ParseFloat(price, 64)
		if err != nil {
			app.badRequestResponse(w, r, errors.New("السعر يجب أن يكون رقمًا صالحًا (مثال: 30 أو 40.50)"))
			return
		}
		if val < 0 {
			app.badRequestResponse(w, r, errors.New("السعر يجب أن يكون غير سالب"))
			return
		}
		product.Price = val
	}
	if discount := r.FormValue("discount"); discount != "" {
		val, err := strconv.ParseFloat(discount, 64)
		if err != nil {
			app.badRequestResponse(w, r, errors.New("الخصم يجب أن يكون رقمًا صالحًا (مثال: 0 أو 5.50)"))
			return
		}
		if val < 0 {
			app.badRequestResponse(w, r, errors.New("الخصم يجب أن يكون غير سالب"))
			return
		}
		product.Discount = val
	}
	if stock := r.FormValue("stock_quantity"); stock != "" {
		val, err := strconv.Atoi(stock)
		if err != nil {
			app.badRequestResponse(w, r, errors.New("كمية المخزون يجب أن تكون عددًا صحيحًا صالحًا (مثال: 0 أو 100)"))
			return
		}
		if val < 0 {
			app.badRequestResponse(w, r, errors.New("كمية المخزون يجب أن تكون غير سالبة"))
			return
		}
		product.StockQuantity = val
	}
	if available := r.FormValue("is_available"); available != "" {
		product.IsAvailable = available == "true"
	} else {
		product.IsAvailable = true
	}

	var newImageName string
	removeImage := r.FormValue("remove_image") == "true"
	file, fileHeader, err := r.FormFile("image")
	if removeImage {
		if oldImage != nil {
			if err := utils.DeleteFile(*oldImage); err != nil {
				log.Printf("Failed to delete old image %s: %v", *oldImage, err)
			}
			product.Image = nil
		}
	} else if err == nil && file != nil {
		if oldImage != nil {
			if err := utils.DeleteFile(*oldImage); err != nil {
				log.Printf("Failed to delete old image %s: %v", *oldImage, err)
			}
		}
		defer file.Close()
		newImageName, err = utils.SaveFile(file, "products", fileHeader.Filename)
		if err != nil {
			app.errorResponse(w, r, http.StatusBadRequest, fmt.Sprintf("فشل في حفظ الصورة: %v", err))
			return
		}
		product.Image = &newImageName
	} else if err != nil && !errors.Is(err, http.ErrMissingFile) {
		app.errorResponse(w, r, http.StatusBadRequest, fmt.Sprintf("خطأ في معالجة ملف الصورة: %v", err))
		return
	}

	v := validator.New()
	data.ValidateProduct(v, product)
	if !v.Valid() {
		if newImageName != "" {
			if err := utils.DeleteFile(newImageName); err != nil {
				log.Printf("Failed to delete new image %s: %v", newImageName, err)
			}
		}
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.Model.ProductDB.Update(product)
	if err != nil {
		if newImageName != "" {
			if err := utils.DeleteFile(newImageName); err != nil {
				log.Printf("Failed to delete new image %s: %v", newImageName, err)
			}
		}
		product.Image = oldImage
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"message": "تم تحديث المنتج بنجاح",
		"product": product,
	})
}
func (app *application) ListProductsHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	products, meta, err := app.Model.ProductDB.List(queryParams)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"products": products,
		"meta":     meta,
	})
}
func (app *application) DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المنتج غير صالح"))
		return
	}

	product, err := app.Model.ProductDB.Get(productID)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	err = app.Model.ProductDB.Delete(productID)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	if product.Image != nil {
		*product.Image = strings.TrimPrefix(*product.Image, data.Domain+"/")
	}
	if product.Image != nil {
		if err := utils.DeleteFile(*product.Image); err != nil {
			log.Printf("Failed to delete image %s for product %s: %v", *product.Image, productID, err)
		}
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{"message": "تم حذف المنتج بنجاح"})
}
