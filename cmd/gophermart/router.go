package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/h1067675/gophermart/cmd/depository"
	"github.com/h1067675/gophermart/internal/compress"
	"github.com/h1067675/gophermart/internal/configurer"
	"github.com/h1067675/gophermart/internal/logger"
)

// General structure
type Connect struct {
	Router     chi.Router
	Depository *depository.Storage
	Config     *configurer.Config
}

// Initialized general structure with a repositary and config
func InitializeRouter(dep *depository.Storage, conf *configurer.Config) Connect {
	var c = Connect{
		Router:     chi.NewRouter(),
		Depository: dep,
		Config:     conf,
	}
	return c
}

// Routing http requests to edpoints
func (c *Connect) Route() chi.Router {
	// Use all middleware-functions
	c.Router.Use(c.CookieAuthorizationMiddleware)
	c.Router.Use(compress.CompressHandler)
	c.Router.Use(logger.ResponseLogging)

	// Делаем маршрутизацию
	c.Router.Route("/", func(r chi.Router) {
		r.Route("/api/user", func(r chi.Router) {
			r.Route("/register", func(r chi.Router) {
				r.Post("/", c.UserRegisterHandler) // POST request for registeration user
			})
			r.Route("/login", func(r chi.Router) {
				r.Post("/", c.UserLoginHandler) // POST request for login user
			})
			r.Route("/orders", func(r chi.Router) {
				r.Post("/", c.UserLoadOrdersHandler) // POST request for load user order to calculate
				r.Get("/", c.UserGetOrdersHandler)   // GET request for load user order to show list
			})
			r.Route("/balance", func(r chi.Router) {
				r.Post("/", c.UserGetBalanceHandler) // GET request for get user balance
			})
			r.Route("/withdrawals", func(r chi.Router) {
				r.Post("/", c.UserGetWithdrawalsHandler) // GET request for get user withdrawals
			})
		})
		r.Route("/api/user/balance", func(r chi.Router) {
			r.Route("/withdraw", func(r chi.Router) {
				r.Post("/", c.UserGetBalanceWithdrawHandler) // POST request for withdrawals to new order
			})
		})
	})
	logger.Log.Info(fmt.Sprintf("Server is running %s", c.Config.GetRunAddress()))
	return c.Router
}

func (c *Connect) StartServer() error {
	if err := http.ListenAndServe(c.Config.GetRunAddress(), c.Route()); err != nil {
		logger.Log.WithError(err).Errorf("error starting the server with a network address %s", c.Config.GetRunAddress())
		return err
	}
	return nil
}
