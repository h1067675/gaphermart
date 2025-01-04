package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/h1067675/gophermart/cmd/depository"
	"github.com/h1067675/gophermart/internal/configurer"
	"github.com/h1067675/gophermart/internal/logger"
)

type test struct {
	name  string
	login user
	req   req
	want  want
}
type user struct {
	login        string
	password     string
	orders       []order
	balance      float64
	withdrawn    float64
	transactions []transactions
}
type order struct {
	login string
	order int
}
type transactions struct {
	order int
	sum   float64
}
type req struct {
	method      string
	contentType string
	handler     string
	body        string
}

type want struct {
	body       string
	headerCode int
}

type Configurer struct {
	configurer.Configurer
}

func (c Configurer) InitializeConfigurer(server string, db string, system string, reload bool) *configurer.Config {
	var result = configurer.Config{ // set defaul settins
		ReloadTables: reload,
	}
	result.RunAddress.Set(server)
	result.DatabaseURI.Set(db)
	result.AccrualSystemAddress.Set(system)

	return &result
}
func TestUserRegisterHandler(t *testing.T) {
	tests := []test{
		{
			name: "200_1",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/register",
				body: `{
								"login": "first",
								"password": "12345678"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 200,
			},
		},
		{
			name: "409_1",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/register",
				body: `{
								"login": "first",
								"password": "87654321"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 409,
			},
		},
		{
			name: "400_1",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/register",
				body: `{
								"id": "first",
								"password": "12345678"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
		{
			name: "400_2",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/register",
				body: `{
								"login": "first",
								"password": ""
							} `,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
		{
			name: "400_3",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/register",
				body: `{
								"login": "",
								"password": "12345678"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
		{
			name: "400_4",
			req: req{
				method:      "POST",
				contentType: "",
				handler:     "/api/user/register",
				body:        ``,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
		{
			name: "400_5",
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/register",
				body: `{
								"login": "second",
								"password": "87654321"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
	}
	var configurer Configurer
	config := configurer.InitializeConfigurer("127.0.0.1:8080", "host=127.0.0.1 port=5432 dbname=postgres user=postgres password=12345678 connect_timeout=10 sslmode=prefer", "127.0.0.1:8090", true)
	var depositary = depository.InitializeStorager(config)
	var connector = InitializeRouter(depositary, config)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger.Log.Info("test is running", test.name)
			b := strings.NewReader(test.req.body)
			request := httptest.NewRequest(http.MethodPost, test.req.handler, b)
			request.Header.Add("Content-Type", test.req.contentType)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(connector.UserRegisterHandler)
			h(w, request)

			result := w.Result()

			assert.Equal(t, test.want.headerCode, result.StatusCode)

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, test.want.body, string(body))
			err = result.Body.Close()
			require.NoError(t, err)
		})
	}
}

func TestUserLoginHandler(t *testing.T) {
	users := []user{
		{
			login:    "first",
			password: "12345678",
		},
	}
	tests := []test{
		{
			name: "200_1",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/login",
				body: `{
								"login": "first",
								"password": "12345678"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 200,
			},
		},
		{
			name: "401_1",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/login",
				body: `{
								"login": "first",
								"password": "87654321"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 401,
			},
		},
		{
			name: "401_2",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/login",
				body: `{
								"login": "second",
								"password": "12345678"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 401,
			},
		},
		{
			name: "400_1",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/login",
				body: `{
								"id": "first",
								"password": "12345678"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
		{
			name: "400_2",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/login",
				body: `{
								"login": "first",
								"password": ""
							} `,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
		{
			name: "400_3",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/login",
				body: `{
								"login": "",
								"password": "12345678"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
		{
			name: "400_4",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/login",
				body:        ``,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
		{
			name: "400_5",
			req: req{
				method:      "POST",
				contentType: "",
				handler:     "/api/user/login",
				body:        ``,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
		{
			name: "400_6",
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/login",
				body: `{
					"login": "first",
					"password": "12345678"
				} `,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
	}
	var configurer Configurer
	config := configurer.InitializeConfigurer("127.0.0.1:8080", "host=127.0.0.1 port=5432 dbname=postgres user=postgres password=12345678 connect_timeout=10 sslmode=prefer", "127.0.0.1:8090", true)
	var depositary = depository.InitializeStorager(config)
	var connector = InitializeRouter(depositary, config)

	for _, user := range users {
		connector.Depository.UserRegister(user.login, user.password)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := strings.NewReader(test.req.body)
			request := httptest.NewRequest(http.MethodPost, test.req.handler, b)
			request.Header.Add("Content-Type", test.req.contentType)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(connector.UserLoginHandler)
			h(w, request)

			result := w.Result()

			assert.Equal(t, test.want.headerCode, result.StatusCode)

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, test.want.body, string(body))
			err = result.Body.Close()
			require.NoError(t, err)
		})
	}
}

func TestUserLoadOrdersHandler(t *testing.T) {
	users := []user{
		{
			login:    "first",
			password: "12345678",
		},
		{
			login:    "second",
			password: "87654321",
		},
	}
	tests := []test{
		{
			name: "401_1",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/orders",
				body: `{
								"login": "first",
								"password": "12345678"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 401,
			},
		},
		{
			name: "401_2",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/orders",
				body: `{
								"order": "12345678903"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 401,
			},
		},
		{
			name: "400_1",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/orders",
				body: `{
								"login": "first",
								"password": "12345678"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
		{
			name: "400_2",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/orders",
				body: `{
								"order": "12345678903"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 400,
			},
		},
		{
			name: "202_1",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/orders",
				body:        `12345678903`,
			},
			want: want{
				body:       "",
				headerCode: 202,
			},
		},
		{
			name: "202_2",
			login: user{
				login:    "second",
				password: "87654321",
			},
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/orders",
				body:        `12345678911`,
			},
			want: want{
				body:       "",
				headerCode: 202,
			},
		},
		{
			name: "200_1",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/orders",
				body:        `12345678903`,
			},
			want: want{
				body:       "",
				headerCode: 200,
			},
		},
		{
			name: "200_2",
			login: user{
				login:    "second",
				password: "87654321",
			},
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/orders",
				body:        `12345678911`,
			},
			want: want{
				body:       "",
				headerCode: 200,
			},
		},
		{
			name: "409_1",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/orders",
				body:        `12345678911`,
			},
			want: want{
				body:       "",
				headerCode: 409,
			},
		},
		{
			name: "409_2",
			login: user{
				login:    "second",
				password: "87654321",
			},
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/orders",
				body:        `12345678903`,
			},
			want: want{
				body:       "",
				headerCode: 409,
			},
		},
		{
			name: "422_1",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/orders",
				body:        `12345678910`,
			},
			want: want{
				body:       "",
				headerCode: 422,
			},
		},
		{
			name: "422_2",
			login: user{
				login:    "second",
				password: "87654321",
			},
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/orders",
				body:        `12345678900`,
			},
			want: want{
				body:       "",
				headerCode: 422,
			},
		},
		{
			name: "422_3",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/orders",
				body: `{
								"order": ""
							} `,
			},
			want: want{
				body:       "",
				headerCode: 422,
			},
		},
		{
			name: "422_4",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/orders",
				body: `{
								"order": "f112345613"
							} `,
			},
			want: want{
				body:       "",
				headerCode: 422,
			},
		},
		{
			name: "422_5",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "POST",
				contentType: "text/plain; charset=utf-8",
				handler:     "/api/user/orders",
				body:        ``,
			},
			want: want{
				body:       "",
				headerCode: 422,
			},
		},
	}
	var configurer Configurer
	config := configurer.InitializeConfigurer("127.0.0.1:8080", "host=127.0.0.1 port=5432 dbname=postgres user=postgres password=12345678 connect_timeout=10 sslmode=prefer", "127.0.0.1:8090", true)
	var depositary = depository.InitializeStorager(config)
	var connector = InitializeRouter(depositary, config)

	for _, user := range users {
		connector.Depository.UserRegister(user.login, user.password)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// login user
			var userID int
			userID, _ = connector.Depository.UserAuthorization(test.login.login, test.login.password)
			b := strings.NewReader(test.req.body)
			ctx := context.WithValue(context.TODO(), KeyUserID, userID)
			request := httptest.NewRequestWithContext(ctx, http.MethodPost, test.req.handler, b)
			request.Header.Add("Content-Type", test.req.contentType)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(connector.UserLoadOrdersHandler)
			h(w, request)

			result := w.Result()

			assert.Equal(t, test.want.headerCode, result.StatusCode)

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, test.want.body, string(body))
			err = result.Body.Close()
			require.NoError(t, err)
		})
	}
}

func TestUserGetOrdersHandler(t *testing.T) {
	users := []user{
		{
			login:    "first",
			password: "12345678",
			orders: []order{
				{
					login: "first",
					order: 12345678903,
				},
				{
					login: "first",
					order: 7790169416,
				},
				{
					login: "first",
					order: 9233795609,
				},
				{
					login: "first",
					order: 61592875043,
				},
				{
					login: "first",
					order: 99243794064,
				},
			},
		},
		{
			login:    "second",
			password: "87654321",
			orders: []order{
				{
					login: "second",
					order: 51924515235,
				},
				{
					login: "second",
					order: 33424179795,
				},
				{
					login: "second",
					order: 97201097694,
				},
				{
					login: "second",
					order: 74083941588,
				},
				{
					login: "second",
					order: 63321680462,
				},
			},
		},
		{
			login:    "third",
			password: "33333333",
		},
	}
	tests := []test{
		{
			name: "401_1",
			req: req{
				method:  "GET",
				handler: "/api/user/orders",
			},
			want: want{
				headerCode: 401,
			},
		},
		{
			name: "401_2",
			req: req{
				method:      "GET",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/orders",
			},
			want: want{
				headerCode: 401,
			},
		},
		{
			name: "204_1",
			login: user{
				login:    "third",
				password: "33333333",
			},
			req: req{
				method:      "GET",
				contentType: "",
				handler:     "/api/user/orders",
			},
			want: want{
				headerCode: 204,
			},
		},
		{
			name: "200_1",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "GET",
				contentType: "",
				handler:     "/api/user/orders",
			},
			want: want{
				headerCode: 200,
			},
		},
		{
			name: "200_1",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "GET",
				contentType: "",
				handler:     "/api/user/orders",
			},
			want: want{
				headerCode: 200,
			},
		},
	}
	var configurer Configurer
	config := configurer.InitializeConfigurer("127.0.0.1:8080", "host=127.0.0.1 port=5432 dbname=postgres user=postgres password=12345678 connect_timeout=10 sslmode=prefer", "127.0.0.1:8090", true)
	var depositary = depository.InitializeStorager(config)
	var connector = InitializeRouter(depositary, config)

	for _, user := range users {
		connector.Depository.UserRegister(user.login, user.password)
		userID, _ := connector.Depository.UserAuthorization(user.login, user.password)
		for _, order := range user.orders {
			connector.Depository.OrderNew(userID, order.order)
		}
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// login user
			var userID int
			userID, _ = connector.Depository.UserAuthorization(test.login.login, test.login.password)
			ctx := context.WithValue(context.TODO(), KeyUserID, userID)
			request := httptest.NewRequestWithContext(ctx, http.MethodPost, test.req.handler, nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(connector.UserGetOrdersHandler)
			h(w, request)

			result := w.Result()

			var b []byte
			if assert.Equal(t, test.want.headerCode, result.StatusCode) && result.StatusCode == 200 {
				var o []depository.UserOrders
				var err error
				o, err = connector.Depository.OrderGetUserOrders(userID)
				require.NoError(t, err)

				b, err = json.Marshal(o)
				require.NoError(t, err)

				body, err := io.ReadAll(result.Body)
				require.NoError(t, err)

				assert.JSONEq(t, string(b), string(body))
				err = result.Body.Close()
				require.NoError(t, err)
			}

		})
	}
}

func TestUserGetBalanceHandler(t *testing.T) {
	users := []user{
		{
			login:     "first",
			password:  "12345678",
			balance:   100.1,
			withdrawn: 50.2,
		},
		{
			login:     "second",
			password:  "87654321",
			balance:   200,
			withdrawn: 0,
		},
		{
			login:     "third",
			password:  "33333333",
			balance:   0,
			withdrawn: 0,
		},
	}
	tests := []test{
		{
			name: "401_1",
			req: req{
				method:  "GET",
				handler: "/api/user/balance",
			},
			want: want{
				headerCode: 401,
			},
		},
		{
			name: "401_2",
			req: req{
				method:      "GET",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/balance",
			},
			want: want{
				headerCode: 401,
			},
		},
		{
			name: "200_1",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "GET",
				contentType: "",
				handler:     "/api/user/balance",
			},
			want: want{
				headerCode: 200,
				body:       `{"current":100.1,"withdrawn":50.2}`,
			},
		},
		{
			name: "200_2",
			login: user{
				login:    "second",
				password: "87654321",
			},
			req: req{
				method:      "GET",
				contentType: "",
				handler:     "/api/user/balance",
			},
			want: want{
				headerCode: 200,
				body:       `{"current":200,"withdrawn":0}`,
			},
		},
		{
			name: "200_3",
			login: user{
				login:    "third",
				password: "33333333",
			},
			req: req{
				method:      "GET",
				contentType: "",
				handler:     "/api/user/balance",
			},
			want: want{
				headerCode: 200,
				body:       `{"current":0,"withdrawn":0}`,
			},
		},
	}
	var configurer Configurer
	config := configurer.InitializeConfigurer("127.0.0.1:8080", "host=127.0.0.1 port=5432 dbname=postgres user=postgres password=12345678 connect_timeout=10 sslmode=prefer", "127.0.0.1:8090", true)
	var depositary = depository.InitializeStorager(config)
	var connector = InitializeRouter(depositary, config)

	tx, err := depositary.DB.Begin()
	if err != nil {
		logger.Log.WithError(err).Error("database error")
		return
	}

	for _, user := range users {
		connector.Depository.UserRegister(user.login, user.password)
		userID, _ := connector.Depository.UserAuthorization(user.login, user.password)
		err = connector.Depository.UserBalanceUpdate(userID, user.balance, user.withdrawn, tx)
		if err != nil {
			tx.Rollback()
		}
	}

	tx.Commit()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// login user
			var userID int
			userID, _ = connector.Depository.UserAuthorization(test.login.login, test.login.password)
			ctx := context.WithValue(context.TODO(), KeyUserID, userID)
			request := httptest.NewRequestWithContext(ctx, http.MethodPost, test.req.handler, nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(connector.UserGetBalanceHandler)
			h(w, request)

			result := w.Result()

			assert.Equal(t, test.want.headerCode, result.StatusCode)

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, test.want.body, string(body))
			err = result.Body.Close()
			require.NoError(t, err)

		})
	}
}

func TestUserGetBalanceWithdrawHandler(t *testing.T) {
	users := []user{
		{
			login:     "first",
			password:  "12345678",
			balance:   1000.1,
			withdrawn: 50.2,
		},
		{
			login:     "second",
			password:  "87654321",
			balance:   2000,
			withdrawn: 0,
		},
		{
			login:     "third",
			password:  "33333333",
			balance:   0,
			withdrawn: 0,
		},
	}
	tests := []test{
		{
			name: "401_1",
			req: req{
				method:  "POST",
				handler: "/api/user/balance/withdraw",
				body: `{
					"order": "12345678903",
					"sum": 100
				} `,
			},
			want: want{
				headerCode: 401,
			},
		},
		{
			name: "401_2",
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/balance/withdraw",
				body: `{
					"order": "12345678903",
					"sum": 100
				} `,
			},
			want: want{
				headerCode: 401,
			},
		},
		{
			name: "402_1",
			login: user{
				login:    "third",
				password: "33333333",
			},
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/balance/withdraw",
				body: `{
					"order": "12345678903",
					"sum": 100
				} `,
			},
			want: want{
				headerCode: 402,
			},
		},
		{
			name: "422_1",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/balance/withdraw",
				body: `{
					"order": "12345678900",
					"sum": 100
				} `,
			},
			want: want{
				headerCode: 422,
			},
		},
		{
			name: "422_2",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/balance/withdraw",
				body: `{
					"order": "a12345678900",
					"sum": 100
				} `,
			},
			want: want{
				headerCode: 422,
			},
		},
		{
			name: "200_1",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/balance/withdraw",
				body: `{
					"order": "12345678903",
					"sum": 100
				} `,
			},
			want: want{
				headerCode: 200,
			},
		},
		{
			name: "200_2",
			login: user{
				login:    "second",
				password: "87654321",
			},
			req: req{
				method:      "POST",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/balance/withdraw",
				body: `{
					"order": "12345678903",
					"sum": 1000
				} `,
			},
			want: want{
				headerCode: 200,
			},
		},
	}
	var configurer Configurer
	config := configurer.InitializeConfigurer("127.0.0.1:8080", "host=127.0.0.1 port=5432 dbname=postgres user=postgres password=12345678 connect_timeout=10 sslmode=prefer", "127.0.0.1:8090", true)
	var depositary = depository.InitializeStorager(config)
	var connector = InitializeRouter(depositary, config)

	tx, err := depositary.DB.Begin()
	if err != nil {
		logger.Log.WithError(err).Error("database error")
		return
	}

	for _, user := range users {
		connector.Depository.UserRegister(user.login, user.password)
		userID, _ := connector.Depository.UserAuthorization(user.login, user.password)
		err = connector.Depository.UserBalanceUpdate(userID, user.balance, user.withdrawn, tx)
		if err != nil {
			tx.Rollback()
		}
	}

	tx.Commit()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// login user
			var userID int
			userID, _ = connector.Depository.UserAuthorization(test.login.login, test.login.password)
			b := strings.NewReader(test.req.body)
			ctx := context.WithValue(context.TODO(), KeyUserID, userID)
			request := httptest.NewRequestWithContext(ctx, http.MethodPost, test.req.handler, b)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(connector.UserGetBalanceWithdrawHandler)
			h(w, request)

			result := w.Result()

			assert.Equal(t, test.want.headerCode, result.StatusCode)

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, test.want.body, string(body))
			err = result.Body.Close()
			require.NoError(t, err)

		})
	}
}

func TestUserGetWithdrawalsHandler(t *testing.T) {
	users := []user{
		{
			login:     "first",
			password:  "12345678",
			balance:   1000.1,
			withdrawn: 50.2,
			transactions: []transactions{
				{
					order: 12345678903,
					sum:   100,
				},
				{
					order: 7790169416,
					sum:   101,
				},
				{
					order: 9233795609,
					sum:   102,
				},
			},
		},
		{
			login:     "second",
			password:  "87654321",
			balance:   2000,
			withdrawn: 0,
			transactions: []transactions{
				{
					order: 61592875043,
					sum:   200,
				},
				{
					order: 33424179795,
					sum:   300,
				},
				{
					order: 74083941588,
					sum:   400,
				},
				{
					order: 63321680462,
					sum:   500,
				},
			},
		},
		{
			login:     "third",
			password:  "33333333",
			balance:   0,
			withdrawn: 0,
		},
	}
	tests := []test{
		{
			name: "401_1",
			req: req{
				method:  "GET",
				handler: "/api/user/withdrawals",
			},
			want: want{
				headerCode: 401,
			},
		},
		{
			name: "401_2",
			req: req{
				method:      "GET",
				contentType: "application/json; charset=utf-8",
				handler:     "/api/user/withdrawals",
			},
			want: want{
				headerCode: 401,
			},
		},
		{
			name: "204_1",
			login: user{
				login:    "third",
				password: "33333333",
			},
			req: req{
				method:  "GET",
				handler: "/api/user/withdrawals",
			},
			want: want{
				headerCode: 204,
			},
		},
		{
			name: "200_1",
			login: user{
				login:    "first",
				password: "12345678",
			},
			req: req{
				method:  "GET",
				handler: "/api/user/withdrawals",
			},
			want: want{
				headerCode: 200,
			},
		},
		{
			name: "200_2",
			login: user{
				login:    "second",
				password: "87654321",
			},
			req: req{
				method:  "GET",
				handler: "/api/user/withdrawals",
			},
			want: want{
				headerCode: 200,
			},
		},
	}
	var configurer Configurer
	config := configurer.InitializeConfigurer("127.0.0.1:8080", "host=127.0.0.1 port=5432 dbname=postgres user=postgres password=12345678 connect_timeout=10 sslmode=prefer", "127.0.0.1:8090", true)
	var depositary = depository.InitializeStorager(config)
	var connector = InitializeRouter(depositary, config)

	tx, err := depositary.DB.Begin()
	if err != nil {
		logger.Log.WithError(err).Error("database error")
		return
	}

	for _, user := range users {
		connector.Depository.UserRegister(user.login, user.password)
		userID, _ := connector.Depository.UserAuthorization(user.login, user.password)
		err = connector.Depository.UserBalanceUpdate(userID, user.balance, user.withdrawn, tx)
		if err != nil {
			tx.Rollback()
		}

	}

	tx.Commit()

	for _, user := range users {
		userID, _ := connector.Depository.UserAuthorization(user.login, user.password)
		for _, transaction := range user.transactions {
			depositary.UserWithdrawal(userID, transaction.order, transaction.sum)
		}
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// login user
			var userID int
			userID, _ = connector.Depository.UserAuthorization(test.login.login, test.login.password)
			ctx := context.WithValue(context.TODO(), KeyUserID, userID)
			request := httptest.NewRequestWithContext(ctx, http.MethodPost, test.req.handler, nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(connector.UserGetWithdrawalsHandler)
			h(w, request)

			result := w.Result()

			var b []byte
			if assert.Equal(t, test.want.headerCode, result.StatusCode) && result.StatusCode == 200 {
				var withdrawals []depository.UserWithDrawals
				withdrawals, err = connector.Depository.UserGetWithdrawals(userID)
				require.NoError(t, err)

				b, err = json.Marshal(withdrawals)
				require.NoError(t, err)

				body, err := io.ReadAll(result.Body)
				require.NoError(t, err)

				assert.JSONEq(t, string(b), string(body))
				err = result.Body.Close()
				require.NoError(t, err)
			}
		})
	}
}
