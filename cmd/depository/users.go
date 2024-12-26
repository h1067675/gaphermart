package depository

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/h1067675/gophermart/internal/logger"
)

type UserBalance struct {
	Balance   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type UserWithDrawals struct {
	Order     int     `json:"order"`
	Sum       float64 `json:"sum"`
	Processed RFCDate `json:"processed_at"`
}

var ErrInsufficientBalance = errors.New("insufficient balance")

// check exist login
func (s Storage) UserCheckLogin(login string) bool {
	var id int
	row := s.DB.QueryRow("SELECT id FROM users WHERE login = $1", login)
	err := row.Scan(id)
	return err != nil
}

// get crypto password
func cryptPassword(pass string) (string, error) {
	h := sha256.New()
	_, err := h.Write([]byte(pass))
	if err != nil {
		logger.Log.WithError(err).Info("error crypto password")
		return "", err
	}
	return string(h.Sum(nil)), nil
}

// register new user
func (s Storage) UserRegister(login string, pass string) bool {
	cryptPass, err := cryptPassword(pass)
	if err == nil && cryptPass != "" {
		_, err := s.DB.Exec("INSERT INTO users_balance (user_id, balance, withdrawal) VALUES ((INSERT INTO users (login, create_at, hash_password) VALUES ($1,$2,$3) RETURNING id), 0,0);", login, time.Now, cryptPass)
		if err != nil {
			logger.Log.WithError(err).Info("error insert new user into db")
			return false
		}
		return true
	}
	return false
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

// withdrawal operation
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
	defer tx.Commit()
	_, err = tx.Exec("INSERT users_transactions (user_id, order_id,	processed_at, sum, withdrawal, balance) VALUES ($1, $2, $3, $4, $5, $6)", userID, order, time.Now, sum, true, balance.Balance-sum)
	if err != nil {
		tx.Rollback()
		logger.Log.WithError(err).Error("database writing error")
		return err
	}
	_, err = tx.Exec("UPDATE users_balance SET balance = $1, withdrawal = $2 WHERE Id = $3", balance.Balance-sum, balance.Withdrawn+sum)
	if err != nil {
		tx.Rollback()
		logger.Log.WithError(err).Error("database writing error")
		return err
	}
	tx.Commit()
	return nil
}

// user withdrawals
func (s Storage) UserGetWithdrawals(userID int) (withdrawals []UserWithDrawals, err error) {
	var rows *sql.Rows
	var withdrawal UserWithDrawals
	rows, err = s.DB.Query("SELECT order_id, sum, processed_at FROM users_transactions WHERE user_id = $1 ORDER BY uploaded_at DESC", userID)
	if err != nil {
		logger.Log.WithError(err).Error("error getting data from the database")
		return nil, err
	}
	defer func() {
		rows.Close()
		rows.Err()
	}()
	for rows.Next() {
		err = rows.Scan(&withdrawal.Order, &withdrawal.Sum, &withdrawal.Processed)
		if err != nil {
			logger.Log.WithError(err).Error("error scanning sql.Rows")
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return withdrawals, nil
}
