package main

import (
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

	// Делаем маршрутизацию
	c.Router.Route("/", func(r chi.Router) {
		r.Route("/api/", func(r chi.Router) {
			r.Route("/user", func(r chi.Router) {
				r.Post("/register", c.UserRegisterHandler) // POST request for registeration user
				r.Post("/login", c.UserLoginHandler)       // POST request for login user
				r.Post("/orders", c.UserLoadOrdersHandler) // POST request for load user order to calculate
				r.Get("/orders", c.UserGetOrdersHandler)   // GET request for load user order to show list
				r.Get("/balance", c.UserGetBalanceHandler) // GET request for get user balance
				r.Route("/balance", func(r chi.Router) {
					r.Post("/withdraw", c.UserGetBalanceWithdrawHandler) // POST request for withdrawals to new order
				})
				r.Get("/withdrawals", c.UserGetWithdrawalsHandler) // GET request for get user withdrawals
			})
			r.Route("/orders", func(r chi.Router) {
				r.Get("/{number}", c.SystemGetOrdersCalcHandler) // GET request for get information about calculation of balance
			})
		})
	})
	// logger.Log.Debug("Server is running", zap.String("server address", c.Config.GetConfig().ServerAddress))
	return c.Router
}

func (c *Connect) StartServer() error {
	if err := http.ListenAndServe(c.Config.RunAddress.String(), c.Route()); err != nil {
		logger.Log.WithError(err).Errorf("error starting the server with a network address %s", c.Config.RunAddress.String())
		return err
	}
	return nil
}
