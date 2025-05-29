package main

import (
	"net/http"
	"time"

	"github.com/go-michi/michi"
)

func (app *application) Router() *michi.Router {
	r := michi.NewRouter()
	r.Use(app.logRequest)
	r.Use(app.recoverPanic)
	r.Use(secureHeaders)
	r.Use(app.ErrorHandlerMiddleware)
	rateLimiter := NewRateLimiter(RateLimiterConfig{
		Skipper: func(r *http.Request) bool {
			return false
		},
		Rate:      600,
		Burst:     100,
		ExpiresIn: 1 * time.Minute,
		IdentifierExtractor: func(r *http.Request) (string, error) {
			return r.RemoteAddr, nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		},
		DenyHandler: func(w http.ResponseWriter, r *http.Request, identifier string, err error) {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
		},
	})

	r.Use(rateLimiter.Limit)

	r.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	r.Route("/", func(sub *michi.Router) {
		// User endpoints
		sub.HandleFunc("GET users", app.AuthMiddleware(app.AdminOnlyMiddleware(http.HandlerFunc(app.ListUsersHandler))))
		sub.HandleFunc("GET users/{id}", app.AuthMiddleware(app.AdminOrSelfMiddleware(http.HandlerFunc(app.GetUserHandler))))
		sub.HandleFunc("PUT users/{id}", app.AuthMiddleware(app.AdminOrSelfMiddleware(http.HandlerFunc(app.UpdateUserHandler))))
		sub.HandleFunc("DELETE users/{id}", app.AuthMiddleware(app.AdminOnlyMiddleware(http.HandlerFunc(app.DeleteUserHandler))))
		sub.HandleFunc("POST login", http.HandlerFunc(app.SigninHandler))
		sub.HandleFunc("POST signup", app.PassTokenMiddleware(app.SignupHandler))
		sub.HandleFunc("POST verifyemail", app.VerifyEmailHandler)
		sub.HandleFunc("POST resendverification", app.ResendVerificationCodeHandler)
		sub.HandleFunc("POST /password-reset/request", app.RequestPasswordResetHandler)
		sub.HandleFunc("POST /password-reset/verify", app.VerifyPasswordResetCodeHandler)
		sub.HandleFunc("POST /password-reset", app.ResetPasswordHandler)
		sub.HandleFunc("POST roles/grant", app.AuthMiddleware(http.HandlerFunc(app.GrantRoleHandler)))
		sub.HandleFunc("DELETE roles/revoke", app.AuthMiddleware(app.AdminOnlyMiddleware(http.HandlerFunc(app.RevokeRoleHandler))))
		sub.HandleFunc("GET roles/{id}", app.GetUserRolesHandler)
		sub.HandleFunc("GET me", app.AuthMiddleware(http.HandlerFunc(app.MeHandler)))

		// Store endpoints
		sub.HandleFunc("POST stores", app.AuthMiddleware(http.HandlerFunc(app.CreateStoreHandler)))
		sub.HandleFunc("GET stores/{id}", (http.HandlerFunc(app.GetStoreHandler)))
		sub.HandleFunc("PUT stores/{id}", app.AuthMiddleware(app.AdminOnlyMiddleware(http.HandlerFunc(app.UpdateStoreHandler))))
		sub.HandleFunc("DELETE stores/{id}", app.AuthMiddleware(app.AdminOnlyMiddleware(http.HandlerFunc(app.DeleteStoreHandler))))
		sub.HandleFunc("GET stores", (http.HandlerFunc(app.ListStoresHandler)))

		// StoreType endpoints
		sub.HandleFunc("POST store-types", app.AuthMiddleware(app.AdminOnlyMiddleware(http.HandlerFunc(app.CreateStoreTypeHandler))))
		sub.HandleFunc("GET store-types/{id}", (http.HandlerFunc(app.GetStoreTypeHandler)))
		sub.HandleFunc("PUT store-types/{id}", app.AuthMiddleware(app.AdminOnlyMiddleware(http.HandlerFunc(app.UpdateStoreTypeHandler))))
		sub.HandleFunc("DELETE store-types/{id}", app.AuthMiddleware(app.AdminOnlyMiddleware(http.HandlerFunc(app.DeleteStoreTypeHandler))))
		sub.HandleFunc("GET store-types", (http.HandlerFunc(app.ListStoreTypesHandler)))

		// Product endpoints
		sub.HandleFunc("POST products", app.AuthMiddleware(app.StoreOwnerOnlyMiddleware(http.HandlerFunc(app.CreateProductHandler))))
		sub.HandleFunc("GET products/{id}", app.AuthMiddleware(http.HandlerFunc(app.GetProductHandler)))
		sub.HandleFunc("PUT products/{id}", app.AuthMiddleware(app.StoreOwnerOnlyMiddleware(http.HandlerFunc(app.UpdateProductHandler))))
		sub.HandleFunc("DELETE products/{id}", app.AuthMiddleware(app.StoreOwnerOnlyMiddleware(http.HandlerFunc(app.DeleteProductHandler))))
		sub.HandleFunc("GET products", app.AuthMiddleware(http.HandlerFunc(app.ListProductsHandler)))

		// Cart endpoints
		sub.HandleFunc("POST carts", app.AuthMiddleware(http.HandlerFunc(app.CreateCartHandler)))
		sub.HandleFunc("GET carts/{id}", app.AuthMiddleware(app.AdminOrSelfMiddleware(http.HandlerFunc(app.GetCartHandler))))
		sub.HandleFunc("GET usercarts", app.AuthMiddleware(app.AdminOrSelfMiddleware(http.HandlerFunc(app.GetUserCartHandler))))
		sub.HandleFunc("DELETE carts/{id}", app.AuthMiddleware(app.AdminOrSelfMiddleware(http.HandlerFunc(app.DeleteCartHandler))))

		// CartItem endpoints
		sub.HandleFunc("POST cart-items", app.AuthMiddleware(http.HandlerFunc(app.AddCartItemHandler)))
		sub.HandleFunc("PUT cart-items/{id}", app.AuthMiddleware(app.AdminOrSelfMiddleware(http.HandlerFunc(app.UpdateCartItemHandler))))
		sub.HandleFunc("DELETE cart-items/{id}", app.AuthMiddleware(app.AdminOrSelfMiddleware(http.HandlerFunc(app.DeleteCartItemHandler))))

		// Order endpoints
		sub.HandleFunc("POST orders", app.AuthMiddleware(http.HandlerFunc(app.CreateOrderFromCartHandler)))
		sub.HandleFunc("GET orders/{id}", app.AuthMiddleware(app.AdminOrSelfMiddleware(http.HandlerFunc(app.GetOrderHandler))))
		sub.HandleFunc("PUT orders/{id}", app.AuthMiddleware(app.StoreOwnerOnlyMiddleware(http.HandlerFunc(app.UpdateOrderHandler))))
		sub.HandleFunc("DELETE orders/{id}", app.AuthMiddleware(app.AdminOrSelfMiddleware(http.HandlerFunc(app.DeleteOrderHandler))))
		sub.HandleFunc("GET orders", app.AuthMiddleware(http.HandlerFunc(app.ListOrdersHandler)))
		sub.HandleFunc("GET storeorders/{store_id}", app.AuthMiddleware(http.HandlerFunc(app.ListStoreOrdersHandler)))

		// OrderItem endpoints
		sub.HandleFunc("GET order-items/{id}", app.AuthMiddleware(app.AdminOrSelfMiddleware(http.HandlerFunc(app.GetOrderItemHandler))))
		sub.HandleFunc("GET order-items", app.AuthMiddleware(http.HandlerFunc(app.ListOrderItemsHandler)))
	})

	return r
}
