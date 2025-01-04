package depository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/h1067675/gophermart/internal/logger"
)

const (
	OrderNew        = "NEW"
	OrderRegistred  = "REGISTERED"
	OrderProcessing = "PROCESSING"
	OrderInvalid    = "INVALID"
	OrderProcessed  = "PROCESSED"
)

type UserOrders struct {
	Number     int     `json:"number"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual,omitempty"`
	UploadedAt RFCDate `json:"uploaded_at"`
}

type RFCDate struct {
	time.Time
}

func (d RFCDate) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return nil, nil
	}
	return []byte(fmt.Sprintf(`"%s"`, d.Time.Format(time.RFC3339))), nil
}

func (s *Storage) OrderUserCheck(order int) (userID int, err error) {
	row := s.DB.QueryRow("SELECT user_id FROM users_orders WHERE order_id = (SELECT id FROM orders WHERE order_number = $1)", order)
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

func (s *Storage) OrderNew(user int, order int) bool {
	var err error
	row := s.DB.QueryRow("INSERT INTO orders (order_number, status) VALUES ($1,$2) RETURNING id;", order, OrderNew)
	var id string
	err = row.Scan(&id)
	if err != nil {
		logger.Log.WithError(err).Info("error insert new order into db")
		return false
	}
	_, err = s.DB.Exec("INSERT INTO users_orders (user_id, order_id) VALUES ($1, $2);", user, id)
	if err != nil {
		logger.Log.WithError(err).Error("error inserting order to orders")
		return false
	}
	return true
}

func (s *Storage) OrderGetUserOrders(user int) (orders []UserOrders, err error) {
	var rows *sql.Rows
	var order UserOrders
	rows, err = s.DB.Query("SELECT order_number, uploaded_at, accrual, status FROM orders WHERE id IN (SELECT order_id FROM users_orders WHERE user_id = $1) ORDER BY uploaded_at DESC", user)
	if err != nil {
		logger.Log.WithError(err).Error("error getting data from the database")
		return nil, err
	}
	defer func() {
		rows.Close()
		rows.Err()
	}()
	for rows.Next() {
		err = rows.Scan(&order.Number, &order.UploadedAt.Time, &order.Accrual, &order.Status)
		if err != nil {
			logger.Log.WithError(err).Error("error scanning sql.Rows")
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (s Storage) OrderGetOrdersInProcess() (orders []int, err error) {
	var rows *sql.Rows
	//statuses := []string{`'` + OrderNew + `'`, `'` + OrderProcessing + `'`}

	rows, err = s.DB.Query("SELECT order_number FROM orders WHERE status IN ($1, $2)", OrderNew, OrderProcessing)
	if err != nil {
		logger.Log.WithError(err).Error("error getting data from the database")
		return nil, err
	}
	defer func() {
		rows.Close()
		rows.Err()
	}()
	for rows.Next() {
		var order int
		err = rows.Scan(&order)
		if err != nil {
			logger.Log.WithError(err).Error("error scanning sql.Rows")
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (s Storage) OrderStatusUpdate(order int, status string, accrual float64, tx *sql.Tx) (err error) {
	_, err = tx.Exec("UPDATE orders SET status = $2, accrual = $3 WHERE Id = $1", order, status, accrual)
	if err != nil {
		logger.Log.WithError(err).Error("error updating the order in the database")
		return err
	}

	return nil
}
