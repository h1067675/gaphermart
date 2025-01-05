package depository

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/h1067675/gophermart/internal/logger"
)

type UserBalance struct {
	Balance   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type UserWithDrawals struct {
	Order     string  `json:"order"`
	Sum       float64 `json:"sum"`
	Processed RFCDate `json:"processed_at"`
}

var ErrInsufficientBalance = errors.New("insufficient balance")

// check exist login
func (s Storage) UserCheckExistLogin(login string) bool {
	var id int
	row := s.DB.QueryRow("SELECT id FROM users WHERE login = $1", login)
	err := row.Scan(&id)
	return err == nil
}

// get crypto password
func cryptPassword(pass string) (string, error) {
	h := sha256.New()
	_, err := h.Write([]byte(pass))
	if err != nil {
		logger.Log.WithError(err).Info("error crypto password")
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(h.Sum(nil))), nil
}

// register new user
func (s Storage) UserRegister(login string, pass string) (int, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		logger.Log.WithError(err).Info("error creating DB transaction")
		return -1, err
	}
	cryptPass, err := cryptPassword(pass)
	if err == nil && cryptPass != "" {
		row := tx.QueryRow("INSERT INTO users (login, hash_password) VALUES ($1, $2) RETURNING id;", login, cryptPass)
		var id int
		err = row.Scan(&id)
		if err != nil {
			logger.Log.WithError(err).Info("error insert new user into db")
			tx.Rollback()
			return -1, err
		}
		_, err = s.DB.Exec("INSERT INTO users_balance (user_id, balance, withdrawal) VALUES ($1, 0, 0);", id)
		if err != nil {
			logger.Log.WithError(err).Info("error insert user into balance table")
			tx.Rollback()
			return -1, err
		}
		tx.Commit()
		return id, nil
	}
	return -1, err
}

func (s *Storage) UserAuthorization(login string, password string) (userID int, err error) {
	var cryptPass string
	cryptPass, err = cryptPassword(password)
	if err != nil {
		logger.Log.WithError(err).Error("password encryption error")
		return -1, err
	}
	if cryptPass == "" {
		logger.Log.Error("unexpected response from the crypto module")
		return -1, fmt.Errorf("an encrypted password cannot be empty")
	}
	row := s.DB.QueryRow("SELECT id FROM users WHERE login = $1 AND hash_password = $2", login, cryptPass)
	err = row.Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		logger.Log.WithError(err).Error("error getting data from the database")
		return -1, err
	}
	return userID, nil
}

// get user balance
func (s Storage) UserGetBalance(userID int) (UserBalance, error) {
	var balance UserBalance
	row := s.DB.QueryRow("SELECT balance, withdrawal FROM users_balance WHERE user_id = $1", userID)
	err := row.Scan(&balance.Balance, &balance.Withdrawn)
	if err != nil {
		logger.Log.WithError(err).Error("error getting user balance from the database")
		return balance, err
	}
	return balance, nil
}

// transaction
func (s Storage) UserWithdrawal(userID int, order int, sum float64) error {
	balance, err := s.UserGetBalance(userID)
	if err != nil {
		logger.Log.WithError(err).Error("balance getting error")
		return err
	}
	if balance.Balance < sum {
		logger.Log.Info("balance is insufficient to debit funds")
		return ErrInsufficientBalance
	}
	tx, err := s.DB.Begin()
	if err != nil {
		logger.Log.WithError(err).Error("database error")
		return err
	}
	_, err = tx.Exec("INSERT INTO users_transactions (user_id, order_id, sum, withdrawal, balance) VALUES ($1, $2, $3, $4, $5)", userID, order, sum, true, balance.Balance-sum)
	if err != nil {
		tx.Rollback()
		logger.Log.WithError(err).Error("database writing error")
		return err
	}
	err = s.UserBalanceUpdate(userID, -sum, sum, tx)
	if err != nil {
		tx.Rollback()
	}
	tx.Commit()
	return nil
}

// user withdrawals
func (s Storage) UserGetWithdrawals(userID int) (withdrawals []UserWithDrawals, err error) {
	var rows *sql.Rows
	var withdrawal UserWithDrawals
	rows, err = s.DB.Query("SELECT order_id, sum, processed_at FROM users_transactions WHERE user_id = $1 ORDER BY processed_at DESC", userID)
	if err != nil {
		logger.Log.WithError(err).Error("error getting data from the database")
		return nil, err
	}
	defer func() {
		rows.Close()
		rows.Err()
	}()
	for rows.Next() {
		err = rows.Scan(&withdrawal.Order, &withdrawal.Sum, &withdrawal.Processed.Time)
		if err != nil {
			logger.Log.WithError(err).Error("error scanning sql.Rows")
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return withdrawals, nil
}

func (s Storage) UserBalanceUpdate(user int, balance float64, withdrawal float64, tx *sql.Tx) (err error) {
	_, err = tx.Exec("UPDATE users_balance SET balance = balance + $2, withdrawal = withdrawal + $3 WHERE user_id = $1", user, balance, withdrawal)
	if err != nil {
		logger.Log.WithError(err).Error("error updating user balance in the database")
		return err
	}
	logger.Log.Infof("DB query: UPDATE users_balance SET balance = balance + %v, withdrawal = withdrawal + %v WHERE user_id = %v", balance, withdrawal, user)
	return err
}
