package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/theplant/luhn"

	"github.com/h1067675/gophermart/cmd/depository"
	"github.com/h1067675/gophermart/internal/authorization"
	"github.com/h1067675/gophermart/internal/logger"
)

type userLoginJSON struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

var errLoginIsEmpty = errors.New("login cannot be empty")
var errPasswordIsEmpty = errors.New("password cannot be empty")
var errBodyIsEmpty = errors.New("password cannot be empty")

func getBodyJs(request http.Request) (js []byte, err error) {
	js, err = io.ReadAll(request.Body)
	if err != nil {
		logger.Log.WithError(err).Info("can't read request body")
		return nil, err
	}
	if len(js) == 0 {
		return nil, errBodyIsEmpty
	}
	return js, nil
}
func (u *userLoginJSON) parse(request http.Request) error {
	js, err := getBodyJs(request)
	if err != nil {
		return err
	}
	err = json.Unmarshal(js, &u)
	if err != nil {
		logger.Log.WithError(err).Info("error json parsing")
		return err
	}
	if u.Login == "" {
		err = errLoginIsEmpty
		logger.Log.WithError(err).Info("login is empty")
	}
	if u.Password == "" {
		err = errors.Join(err, errPasswordIsEmpty)
		logger.Log.WithError(err).Info("password is empty")
	}
	return err
}

// user resister handler
func (c *Connect) UserRegisterHandler(responce http.ResponseWriter, request *http.Request) {
	if !strings.Contains(request.Header.Get("Content-Type"), "application/json") {
		responce.WriteHeader(http.StatusBadRequest)
	}
	var register userLoginJSON
	if err := register.parse(*request); err != nil {
		if errors.Is(err, errLoginIsEmpty) || errors.Is(err, errPasswordIsEmpty) || errors.Is(err, errBodyIsEmpty) {
			responce.WriteHeader(http.StatusBadRequest)
			return
		}
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	if c.Depository.UserCheckExistLogin(register.Login) {
		logger.Log.Info(fmt.Sprintf("login %s alredy exist", register.Login))
		responce.WriteHeader(http.StatusConflict)
		return
	}
	if c.Depository.UserRegister(register.Login, register.Password) {
		logger.Log.Info(fmt.Sprintf("user %s is register", register.Login))
		responce.WriteHeader(http.StatusOK)
		return
	}
	responce.WriteHeader(http.StatusInternalServerError)
}

// user login handler
func (c *Connect) UserLoginHandler(responce http.ResponseWriter, request *http.Request) {
	if !strings.Contains(request.Header.Get("Content-Type"), "application/json") {
		responce.WriteHeader(http.StatusBadRequest)
	}
	var loginUser userLoginJSON
	if err := loginUser.parse(*request); err != nil {
		logger.Log.WithError(err).Info("error json parsing")
		if errors.Is(err, errLoginIsEmpty) || errors.Is(err, errPasswordIsEmpty) || errors.Is(err, errBodyIsEmpty) {
			responce.WriteHeader(http.StatusBadRequest)
			return
		}
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	token, err := authorization.UserAuthorization(c.Depository, loginUser.Login, loginUser.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			responce.WriteHeader(http.StatusUnauthorized)
			return
		}
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	cookie := &http.Cookie{
		Name:   "token",
		Value:  token,
		MaxAge: 60 * 60 * 24,
		Path:   "/",
	}
	http.SetCookie(responce, cookie)
	responce.WriteHeader(http.StatusOK)
}

// POST /api/user/orders
// load user number order
func (c *Connect) UserLoadOrdersHandler(responce http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value(KeyUserID)
	if userID == nil || userID.(int) <= 0 {
		responce.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !strings.Contains(request.Header.Get("Content-Type"), "text/plain") {
		responce.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		logger.Log.WithError(err).Error("error reading http body")
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(body) <= 0 {
		logger.Log.Info("order number cannot be empty")
		responce.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	order, err := strconv.Atoi(string(body))
	if err != nil {
		logger.Log.Info("wrong order format")
		responce.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if !luhn.Valid(order) {
		logger.Log.Info("wrong order format (is not Luhn)")
		responce.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	orderUserID, err := c.Depository.OrderUserCheck(order)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	if orderUserID > 0 {
		if orderUserID == userID {
			responce.WriteHeader(http.StatusOK)
			return
		}
		responce.WriteHeader(http.StatusConflict)
		return
	}

	if c.Depository.OrderNew(userID.(int), order) {
		responce.WriteHeader(http.StatusAccepted)
		return
	}
	responce.WriteHeader(http.StatusInternalServerError)
}

// GET /api/user/orders
func (c *Connect) UserGetOrdersHandler(responce http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value(KeyUserID)
	if userID == nil || userID.(int) <= 0 {
		responce.WriteHeader(http.StatusUnauthorized)
		return
	}
	orders, err := c.Depository.OrderGetUserOrders(userID.(int))
	if err != nil {
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		responce.WriteHeader(http.StatusNoContent)
		return
	}
	body, err := json.Marshal(orders)
	if err != nil {
		logger.Log.WithError(err).Error("error JSON marshal")
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	responce.Header().Add("Content-Type", "application/json")
	responce.WriteHeader(http.StatusOK)
	responce.Write(body)
}

// GET /api/user/balance
func (c *Connect) UserGetBalanceHandler(responce http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value(KeyUserID)
	if userID == nil || userID.(int) <= 0 {
		responce.WriteHeader(http.StatusUnauthorized)
		return
	}
	balance, err := c.Depository.UserGetBalance(userID.(int))
	if err != nil {
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	body, err := json.Marshal(balance)
	if err != nil {
		logger.Log.WithError(err).Error("error JSON marshal")
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	responce.Header().Add("Content-Type", "application/json")
	responce.WriteHeader(http.StatusOK)
	responce.Write(body)
}

type withdrawal struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

// POST /api/user/balance/withdraw
func (c *Connect) UserGetBalanceWithdrawHandler(responce http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value(KeyUserID)
	if userID == nil || userID.(int) <= 0 {
		responce.WriteHeader(http.StatusUnauthorized)
		return
	}
	js, err := getBodyJs(*request)
	if err != nil {
		logger.Log.WithError(err).Info("body getting error")
		responce.WriteHeader(http.StatusInternalServerError)
	}
	var w withdrawal
	err = json.Unmarshal(js, &w)
	if err != nil {
		logger.Log.WithError(err).Info("json parsimg error")
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	var order int
	order, err = strconv.Atoi(w.Order)
	if err != nil {
		logger.Log.Info("wrong order format (is not number)")
		responce.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if !luhn.Valid(order) {
		logger.Log.Info("wrong order format (is not Luhn)")
		responce.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	err = c.Depository.UserWithdrawal(userID.(int), order, w.Sum)
	if err != nil {
		if errors.Is(err, depository.ErrInsufficientBalance) {
			responce.WriteHeader(http.StatusPaymentRequired)
			return
		}
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	responce.WriteHeader(http.StatusOK)
}

// GET /api/user/withdrawals
func (c *Connect) UserGetWithdrawalsHandler(responce http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value(KeyUserID)
	if userID == nil || userID.(int) <= 0 {
		responce.WriteHeader(http.StatusUnauthorized)
		return
	}
	withdrawals, err := c.Depository.UserGetWithdrawals(userID.(int))
	if err != nil {
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(withdrawals) == 0 {
		responce.WriteHeader(http.StatusNoContent)
		return
	}
	body, err := json.Marshal(withdrawals)
	if err != nil {
		logger.Log.WithError(err).Error("error JSON marshal")
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	responce.Header().Add("Content-Type", "application/json")
	responce.WriteHeader(http.StatusOK)
	responce.Write(body)
}
