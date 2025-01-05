package main

import (
	"database/sql"
	"encoding/json"
	"errors"
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

func setAuthirizationCookie(response http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:   "token",
		Value:  token,
		MaxAge: 60 * 60 * 24,
		Path:   "/",
	}
	http.SetCookie(response, cookie)
}

// user resister handler
func (c *Connect) UserRegisterHandler(response http.ResponseWriter, request *http.Request) {
	if !strings.Contains(request.Header.Get("Content-Type"), "application/json") {
		response.WriteHeader(http.StatusBadRequest)
	}
	var register userLoginJSON
	if err := register.parse(*request); err != nil {
		if errors.Is(err, errLoginIsEmpty) || errors.Is(err, errPasswordIsEmpty) || errors.Is(err, errBodyIsEmpty) {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	if c.Depository.UserCheckExistLogin(register.Login) {
		logger.Log.Infof("login %s alredy exist", register.Login)
		response.WriteHeader(http.StatusConflict)
		return
	}
	userID, err := c.Depository.UserRegister(register.Login, register.Password)
	if err != nil || userID < 1 {
		logger.Log.Infof("error registration user %s", register.Login)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	logger.Log.Infof("user %s is register", register.Login)
	token, err := authorization.SetToken(userID)
	if err != nil {
		logger.Log.Infof("error creating token for user %s", register.Login)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	setAuthirizationCookie(response, token)
	logger.Log.Infof("user %s is logined", register.Login)
	response.WriteHeader(http.StatusOK)
}

// user login handler
func (c *Connect) UserLoginHandler(response http.ResponseWriter, request *http.Request) {
	if !strings.Contains(request.Header.Get("Content-Type"), "application/json") {
		response.WriteHeader(http.StatusBadRequest)
	}
	var loginUser userLoginJSON
	if err := loginUser.parse(*request); err != nil {
		logger.Log.WithError(err).Info("error json parsing")
		if errors.Is(err, errLoginIsEmpty) || errors.Is(err, errPasswordIsEmpty) || errors.Is(err, errBodyIsEmpty) {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	token, err := authorization.UserAuthorization(c.Depository, loginUser.Login, loginUser.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	setAuthirizationCookie(response, token)
	logger.Log.Infof("user %s is logined", loginUser.Login)
	response.WriteHeader(http.StatusOK)
}

// POST /api/user/orders
// load user number order
func (c *Connect) UserLoadOrdersHandler(response http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value(KeyUserID)
	if userID == nil || userID.(int) <= 0 {
		response.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !strings.Contains(request.Header.Get("Content-Type"), "text/plain") {
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		logger.Log.WithError(err).Error("error reading http body")
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(body) <= 0 {
		logger.Log.Info("order number cannot be empty")
		response.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	order, err := strconv.Atoi(string(body))
	if err != nil {
		logger.Log.Info("wrong order format")
		response.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if !luhn.Valid(order) {
		logger.Log.Info("wrong order format (is not Luhn)")
		response.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	orderUserID, err := c.Depository.OrderUserCheck(order)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	if orderUserID > 0 {
		if orderUserID == userID {
			response.WriteHeader(http.StatusOK)
			return
		}
		response.WriteHeader(http.StatusConflict)
		return
	}

	if c.Depository.OrderNew(userID.(int), order) {
		response.WriteHeader(http.StatusAccepted)
		return
	}
	response.WriteHeader(http.StatusInternalServerError)
}

// GET /api/user/orders
func (c *Connect) UserGetOrdersHandler(response http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value(KeyUserID)
	if userID == nil || userID.(int) <= 0 {
		response.WriteHeader(http.StatusUnauthorized)
		return
	}
	orders, err := c.Depository.OrderGetUserOrders(userID.(int))
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		response.WriteHeader(http.StatusNoContent)
		return
	}
	body, err := json.Marshal(orders)
	if err != nil {
		logger.Log.WithError(err).Error("error JSON marshal")
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	response.Header().Add("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	logger.Log.Info("response write body:", string(body))
	response.Write(body)
}

// GET /api/user/balance
func (c *Connect) UserGetBalanceHandler(response http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value(KeyUserID)
	if userID == nil || userID.(int) <= 0 {
		response.WriteHeader(http.StatusUnauthorized)
		return
	}
	balance, err := c.Depository.UserGetBalance(userID.(int))
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	body, err := json.Marshal(balance)
	if err != nil {
		logger.Log.WithError(err).Error("error JSON marshal")
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	response.Header().Add("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	logger.Log.Info("response write body:", string(body))
	response.Write(body)
}

type withdrawal struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

// POST /api/user/balance/withdraw
func (c *Connect) UserGetBalanceWithdrawHandler(response http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value(KeyUserID)
	if userID == nil || userID.(int) <= 0 {
		response.WriteHeader(http.StatusUnauthorized)
		return
	}
	js, err := getBodyJs(*request)
	if err != nil {
		logger.Log.WithError(err).Info("body getting error")
		response.WriteHeader(http.StatusInternalServerError)
	}
	var w withdrawal
	err = json.Unmarshal(js, &w)
	if err != nil {
		logger.Log.WithError(err).Info("json parsimg error")
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	var order int
	order, err = strconv.Atoi(w.Order)
	if err != nil {
		logger.Log.Info("wrong order format (is not number)")
		response.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if !luhn.Valid(order) {
		logger.Log.Info("wrong order format (is not Luhn)")
		response.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	err = c.Depository.UserWithdrawal(userID.(int), order, w.Sum)
	if err != nil {
		if errors.Is(err, depository.ErrInsufficientBalance) {
			response.WriteHeader(http.StatusPaymentRequired)
			return
		}
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	response.WriteHeader(http.StatusOK)
}

// GET /api/user/withdrawals
func (c *Connect) UserGetWithdrawalsHandler(response http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value(KeyUserID)
	if userID == nil || userID.(int) <= 0 {
		response.WriteHeader(http.StatusUnauthorized)
		return
	}
	withdrawals, err := c.Depository.UserGetWithdrawals(userID.(int))
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(withdrawals) == 0 {
		response.WriteHeader(http.StatusNoContent)
		return
	}
	body, err := json.Marshal(withdrawals)
	if err != nil {
		logger.Log.WithError(err).Error("error JSON marshal")
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	response.Header().Add("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	logger.Log.Info("response write body:", string(body))
	response.Write(body)
}
