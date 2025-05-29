package main

import (
	"errors"
	"fmt"
	"net/http"
	"project/internal/data"
	"project/utils"
	"project/utils/validator"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (app *application) SigninHandler(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(r.FormValue("email"))
	password := r.FormValue("password")

	if email == "" || password == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "يجب إدخال البريد الإلكتروني وكلمة المرور")
		return
	}

	user, err := app.Model.UserDB.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, data.ErrUserNotFound) {
			app.errorResponse(w, r, http.StatusUnauthorized, "البريد الإلكتروني أو كلمة المرور غير صحيحة")
			return
		}
		app.handleRetrievalError(w, r, err)
		return
	}

	if !utils.CheckPassword(user.Password, password) {
		app.errorResponse(w, r, http.StatusUnauthorized, "البريد الإلكتروني أو كلمة المرور غير صحيحة")
		return
	}

	if !user.Verified {
		if time.Now().After(user.VerificationCodeExpiry) {
			user.VerificationCode = "000000"
			user.VerificationCodeExpiry = time.Now().Add(5 * time.Minute)
			user.LastVerificationCodeSent = time.Now()

			if err := app.Model.UserDB.UpdateUser(user); err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}

			app.errorResponse(w, r, http.StatusForbidden, "انتهت صلاحية رمز التحقق. يرجى إدخال رمز التحقق 000000 لتفعيل حسابك.")
			return
		}

		app.errorResponse(w, r, http.StatusForbidden, "يجب التحقق من حسابك بإدخال رمز التحقق 000000.")
		return
	}

	userRoles, err := app.Model.UserRoleDB.GetUserRoles(user.ID)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	token, err := utils.GenerateToken(user.ID.String(), userRoles)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SetTokenCookie(w, token)

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"expires": "24 ساعة",
		"token":   token,
	})
}
func (app *application) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المستخدم غير صالح"))
		return
	}

	user, err := app.Model.UserDB.GetUser(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}
	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{"user": user})
}

func (app *application) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المستخدم غير صالح"))
		return
	}

	user, err := app.Model.UserDB.GetUser(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	// Check if the requester is an admin
	userRoles, _ := r.Context().Value(UserRoleKey).([]string)
	isAdmin := false
	for _, role := range userRoles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}

	// Update user fields
	if name := r.FormValue("name"); name != "" {
		user.Name = name
	}
	if email := r.FormValue("email"); email != "" {
		user.Email = email
	}
	if phone := r.FormValue("phone_number"); phone != "" {
		user.PhoneNumber = phone
	}
	if address := r.FormValue("address_text"); address != "" {
		user.AddressText = &address
	}
	if latStr := r.FormValue("latitude"); latStr != "" {
		if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
			user.Latitude = &lat
		}
	}
	if longStr := r.FormValue("longitude"); longStr != "" {
		if long, err := strconv.ParseFloat(longStr, 64); err == nil {
			user.Longitude = &long
		}
	}

	if user.Image != nil {
		*user.Image = strings.TrimPrefix(*user.Image, data.Domain+"/")
	}
	if file, fileHeader, err := r.FormFile("image"); err == nil {
		defer file.Close()
		newFileName, err := utils.SaveFile(file, "users", fileHeader.Filename)
		if err != nil {
			app.errorResponse(w, r, http.StatusBadRequest, "ملف غير صالح")
			return
		}

		if user.Image != nil {
			utils.DeleteFile(*user.Image)
		}
		user.Image = &newFileName
	}

	v := validator.New()

	password := r.FormValue("password")
	if password != "" {
		user.Password = password
	}

	// Validate user fields with conditional validation
	fieldsToValidate := []string{"name", "email", "phone_number", "password", "address"}
	if isAdmin {
		fieldsToValidate = append(fieldsToValidate, "email")
	}
	data.ValidateUser(v, user, fieldsToValidate...)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if password != "" {
		hashedPassword, err := utils.HashPassword(password)
		if err != nil {
			app.errorResponse(w, r, http.StatusInternalServerError, "خطأ في تشفير كلمة المرور")
			return
		}
		user.Password = hashedPassword
	}

	err = app.Model.UserDB.UpdateUser(user)
	if err != nil {
		if errors.Is(err, data.ErrEmailAlreadyInserted) {
			app.errorResponse(w, r, http.StatusConflict, "البريد الإلكتروني مسجل مسبقاً")
			return
		}
		if errors.Is(err, data.ErrPhoneAlreadyInserted) {
			app.errorResponse(w, r, http.StatusConflict, "رقم الهاتف مسجل مسبقاً")
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	token, err := utils.GenerateToken(user.ID.String(), userRoles)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SetTokenCookie(w, token)

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"user":    user,
		"token":   token,
		"expires": "24 hours",
	})
}

func (app *application) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.badRequestResponse(w, r, errors.New("معرف المستخدم غير صالح"))
		return
	}

	user, err := app.Model.UserDB.GetUser(id)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	err = app.Model.UserDB.DeleteUser(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if user.Image != nil {
		utils.DeleteFile(*user.Image)
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{"message": "تم حذف المستخدم بنجاح"})
}

func (app *application) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	users, meta, err := app.Model.UserDB.ListUsers(queryParams)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"users": users,
		"meta":  meta,
	})
}
func (app *application) SignupHandler(w http.ResponseWriter, r *http.Request) {
	userRoles, _ := r.Context().Value(UserRoleKey).([]string)

	isAdmin := false
	for _, role := range userRoles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}

	v := validator.New()
	user := &data.User{
		Name:        r.FormValue("name"),
		Email:       strings.ToLower(r.FormValue("email")),
		PhoneNumber: r.FormValue("phone_number"),
		Password:    r.FormValue("password"),
		Verified:    isAdmin, // Admins are auto-verified
	}

	// Handle address fields
	if address := r.FormValue("address_text"); address != "" {
		user.AddressText = &address
	}
	if latStr := r.FormValue("latitude"); latStr != "" {
		if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
			user.Latitude = &lat
		}
	}
	if longStr := r.FormValue("longitude"); longStr != "" {
		if long, err := strconv.ParseFloat(longStr, 64); err == nil {
			user.Longitude = &long
		}
	}

	if user.Name == "" || user.Email == "" || user.Password == "" || user.PhoneNumber == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "يجب ملء جميع الحقول المطلوبة")
		return
	}

	// Check if email or phone number already exists
	_, err := app.Model.UserDB.GetUserByEmail(user.Email)
	if err == nil {
		app.errorResponse(w, r, http.StatusConflict, "البريد الإلكتروني مسجل مسبقا")
		return
	}
	if !errors.Is(err, data.ErrUserNotFound) {
		app.serverErrorResponse(w, r, err)
		return
	}

	_, err = app.Model.UserDB.GetUserByPhoneNumber(user.PhoneNumber)
	if err == nil {
		app.errorResponse(w, r, http.StatusConflict, "رقم الهاتف مسجل مسبقا")
		return
	}
	if !errors.Is(err, data.ErrUserNotFound) {
		app.serverErrorResponse(w, r, err)
		return
	}

	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		app.errorResponse(w, r, http.StatusInternalServerError, "خطأ في تشفير كلمة المرور")
		return
	}
	user.Password = hashedPassword

	if file, fileHeader, err := r.FormFile("img"); err == nil {
		defer file.Close()
		imageName, err := utils.SaveFile(file, "users", fileHeader.Filename)
		if err != nil {
			app.errorResponse(w, r, http.StatusBadRequest, "صورة غير صالحة")
			return
		}
		user.Image = &imageName
	}

	var role int
	roleStr := r.FormValue("role")
	if !isAdmin {
		role = 3
	} else {
		if roleStr == "" {
			role = 3
		} else {
			var err error
			role, err = strconv.Atoi(roleStr)
			if err != nil {
				app.errorResponse(w, r, http.StatusBadRequest, "الدور غير صالح")
				return
			}
		}
	}

	// Set fixed verification code for non-admin users
	if !isAdmin {
		user.VerificationCode = "000000"
		user.VerificationCodeExpiry = time.Now().Add(5 * time.Minute)
	}

	data.ValidateUser(v, user, "name", "email", "phone_number", "password", "address")
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Store the user in the database
	if err := app.Model.UserDB.InsertUser(user); err != nil {
		if errors.Is(err, data.ErrEmailAlreadyInserted) {
			app.errorResponse(w, r, http.StatusConflict, "البريد الإلكتروني مسجل مسبقا")
			return
		}
		if errors.Is(err, data.ErrPhoneAlreadyInserted) {
			app.errorResponse(w, r, http.StatusConflict, "رقم الهاتف مسجل مسبقا")
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.Model.UserRoleDB.GrantRole(user.ID, role)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	message := "تم التسجيل بنجاح"
	if !isAdmin {
		message = "تم التسجيل بنجاح. يرجى إدخال رمز التحقق 000000 لتفعيل حسابك."
	}

	utils.SendJSONResponse(w, http.StatusCreated, utils.Envelope{
		"message": message,
		"user":    user,
	})
}
func (app *application) MeHandler(w http.ResponseWriter, r *http.Request) {
	idstr := r.Context().Value(UserIDKey).(string)
	userid := uuid.MustParse(idstr)
	user, err := app.Model.UserDB.GetUser(userid)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"user": user,
	})
}

func (app *application) VerifyEmailHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	verificationCode := r.FormValue("verification_code")
	if email == "" || verificationCode == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "البريد الإلكتروني ورمز التحقق مطلوبان")
		return
	}

	user, err := app.Model.UserDB.GetUserByEmail(email)
	if err != nil {
		app.errorResponse(w, r, http.StatusNotFound, "المستخدم غير موجود")
		return
	}

	err = app.Model.UserDB.VerifyUser(user.ID, verificationCode)
	if err != nil {
		app.errorResponse(w, r, http.StatusForbidden, err.Error())
		return
	}

	userRoles, err := app.Model.UserRoleDB.GetUserRoles(user.ID)
	if err != nil {
		app.handleRetrievalError(w, r, err)
		return
	}

	token, err := utils.GenerateToken(user.ID.String(), userRoles)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"message": "تم التحقق من البريد الإلكتروني بنجاح",
		"token":   token,
	})
}
func (app *application) ResendVerificationCodeHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")

	if email == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "يجب إدخال البريد الإلكتروني")
		return
	}

	user, err := app.Model.UserDB.GetUserByEmail(email)
	if err != nil {
		app.errorResponse(w, r, http.StatusNotFound, "لم يتم العثور على المستخدم")
		return
	}
	if user.Verified {
		app.errorResponse(w, r, http.StatusBadRequest, "المستخدم موثق بالفعل")
		return
	}

	if time.Since(user.LastVerificationCodeSent) < 5*time.Minute {
		timeLeft := 5*time.Minute - time.Since(user.LastVerificationCodeSent)
		minutes := int(timeLeft.Minutes())
		seconds := int(timeLeft.Seconds()) % 60

		app.errorResponse(w, r, http.StatusTooManyRequests,
			fmt.Sprintf("يرجى الانتظار %d دقيقة و %d ثانية قبل طلب رمز التحقق", minutes, seconds))
		return
	}

	user.VerificationCode = "000000"
	user.VerificationCodeExpiry = time.Now().Add(5 * time.Minute)
	user.LastVerificationCodeSent = time.Now()

	if err := app.Model.UserDB.UpdateUser(user); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"message": "رمز التحقق هو: 000000",
		"code":    "000000",
	})
}
func (app *application) RequestPasswordResetHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")

	// Validate email
	if email == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "البريد الإلكتروني مطلوب")
		return
	}

	// Find user by email
	user, err := app.Model.UserDB.GetUserByEmail(email)
	if err != nil {
		app.errorResponse(w, r, http.StatusNotFound, "المستخدم غير موجود")
		return
	}
	if time.Since(user.LastVerificationCodeSent) < 5*time.Minute {
		timeLeft := 5*time.Minute - time.Since(user.LastVerificationCodeSent)
		minutes := int(timeLeft.Minutes())
		seconds := int(timeLeft.Seconds()) % 60

		app.errorResponse(w, r, http.StatusTooManyRequests,
			fmt.Sprintf("يرجى الانتظار %d دقيقة و %d ثانية قبل طلب رمز التحقق", minutes, seconds))
		return
	}

	user.VerificationCode = "000000"
	user.VerificationCodeExpiry = time.Now().Add(5 * time.Minute)
	user.LastVerificationCodeSent = time.Now()

	// Update the user with the new verification code and expiry time
	if err := app.Model.UserDB.UpdateUser(user); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"message": "رمز التحقق لإعادة تعيين كلمة المرور هو: 000000",
		"code":    "000000",
	})
}
func (app *application) VerifyPasswordResetCodeHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	verificationCode := r.FormValue("verification_code")

	// Validate input
	if email == "" || verificationCode == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "البريد الإلكتروني ورمز التحقق مطلوبين")
		return
	}

	// Find user by email
	user, err := app.Model.UserDB.GetUserByEmail(email)
	if err != nil {
		app.errorResponse(w, r, http.StatusNotFound, "المستخدم غير موجود")
		return
	}

	// Check if the verification code matches and hasn't expired
	if user.VerificationCode != verificationCode {
		app.errorResponse(w, r, http.StatusForbidden, "رمز التحقق غير صالح")
		return
	}

	if time.Now().After(user.VerificationCodeExpiry) {
		app.errorResponse(w, r, http.StatusForbidden, "رمز التحقق منتهي الصلاحية")
		return
	}

	// Respond with success
	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"message": "تم التحقق بنجاح، يمكنك الآن إعادة تعيين كلمة المرور",
	})
}

func (app *application) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	verificationCode := r.FormValue("verification_code")
	newPassword := r.FormValue("new_password")

	// Validate input
	if email == "" || verificationCode == "" || newPassword == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "البريد الإلكتروني ورمز التحقق وكلمة المرور الجديدة مطلوبين")
		return
	}

	// Find user by email
	user, err := app.Model.UserDB.GetUserByEmail(email)
	if err != nil {
		app.errorResponse(w, r, http.StatusNotFound, "المستخدم غير موجود")
		return
	}

	// Check if the verification code matches and hasn't expired
	if user.VerificationCode != verificationCode {
		app.errorResponse(w, r, http.StatusForbidden, "رمز التحقق غير صالح")
		return
	}

	if time.Now().After(user.VerificationCodeExpiry) {
		app.errorResponse(w, r, http.StatusForbidden, "رمز التحقق منتهي الصلاحية")
		return
	}

	// Hash the new password
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Update user's password
	user.Password = hashedPassword
	user.VerificationCode = ""
	user.VerificationCodeExpiry = time.Time{}

	// Save the updated user in the database
	if err := app.Model.UserDB.UpdateUser(user); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Respond to the client
	utils.SendJSONResponse(w, http.StatusOK, utils.Envelope{
		"message": "تم إعادة تعيين كلمة المرور بنجاح",
	})
}
