package loader

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/h1067675/gophermart/cmd/client"
	"github.com/h1067675/gophermart/cmd/depository"
	"github.com/h1067675/gophermart/internal/logger"
)

type Loader struct {
	Server      string
	Periodicity time.Duration
	Depository  *depository.Storage
	Client      client.Client
}

func InitializeLoader(depository *depository.Storage, server string, periodicity time.Duration) Loader {
	var loader = Loader{
		Server:      server,
		Periodicity: periodicity,
		Depository:  depository,
	}
	return loader
}

func (l Loader) updateOrder(chIn chan struct {
	order   int
	status  string
	accrual float64
}) {
	tx, err := l.Depository.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Commit()
	for ch := range chIn {
		if ch.status != "" {
			err = l.Depository.OrderStatusUpdate(ch.order, ch.status, ch.accrual, tx)
			if err != nil {
				tx.Rollback()
			}

			var userID int
			userID, err = l.Depository.OrderUserCheck(ch.order)
			if err != nil {
				tx.Rollback()
			}

			err = l.Depository.UserBalanceUpdate(userID, ch.accrual, 0, tx)
			if err != nil {
				tx.Rollback()
			}
		}
	}
	tx.Commit()
}

type responseCalculator struct {
	Order   int     `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

func (l Loader) UpdateOrdersStatuses(orders []int) (err error) {
	chDone := make(chan struct{})
	defer close(chDone)
	inputCh := l.generator(chDone, orders)

	channels := l.fanOut(chDone, inputCh, len(orders))

	collectResultCh := l.fanIn(chDone, channels...)

	l.updateOrder(collectResultCh)

	return errors.New("error of update statuses")
}

func (l Loader) generator(chDone chan struct{}, orders []int) chan struct {
	order int
} {
	chRes := make(chan struct {
		order int
	})
	go func() {
		defer close(chRes)
		for _, e := range orders {
			select {
			case <-chDone:
				return
			case chRes <- struct {
				order int
			}{order: e}:
			}
		}
	}()
	return chRes
}

func (l Loader) fanOut(chDone chan struct{}, chIn chan struct {
	order int
}, nWorkers int) []chan struct {
	order   int
	status  string
	accrual float64
} {
	channels := make([]chan struct {
		order   int
		status  string
		accrual float64
	}, nWorkers)

	for i := 0; i < nWorkers; i++ {
		addRes := l.getOrderStatusFromServerAPI(chDone, chIn)
		channels[i] = addRes
	}
	return channels
}

func (l Loader) fanIn(chDone chan struct{}, resultChs ...chan struct {
	order   int
	status  string
	accrual float64
}) chan struct {
	order   int
	status  string
	accrual float64
} {
	finalCh := make(chan struct {
		order   int
		status  string
		accrual float64
	})
	var wg sync.WaitGroup
	for _, ch := range resultChs {
		chClosure := ch
		wg.Add(1)

		go func() {
			defer wg.Done()
			for data := range chClosure {
				select {
				case <-chDone:
					return
				case finalCh <- data:
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(finalCh)
	}()

	return finalCh
}

func (l Loader) getOrderStatusFromServerAPI(chDone chan struct{}, inChan chan struct {
	order int
}) chan struct {
	order   int
	status  string
	accrual float64
} {
	chResult := make(chan struct {
		order   int
		status  string
		accrual float64
	})
	go func() {
		defer close(chResult)
		select {
		case <-chDone:
			return
		case in := <-inChan:
			logger.Log.Info(fmt.Sprintf("loader: query -> GET %s/api/orders/%v", l.Server, in.order))
			body, status, err := l.Client.GET(l.Server, "/api/orders/", in.order)
			if err != nil {
				logger.Log.WithError(err).Error("error getting status from outer sistem")
				return
			}
			logger.Log.Info(fmt.Sprintf("loader: answer <- get response with status %v and body %s", status, string(body)))
			if status == http.StatusNoContent {
				chResult <- struct {
					order   int
					status  string
					accrual float64
				}{order: in.order, status: depository.OrderInvalid}
			}
			if status == http.StatusTooManyRequests || status == http.StatusInternalServerError {
				chResult <- struct {
					order   int
					status  string
					accrual float64
				}{order: in.order, status: ""}
			}
			if status == http.StatusOK {
				var js responseCalculator
				err := json.Unmarshal(body, &js)
				if err != nil {
					logger.Log.WithError(err).Error("json parsing error")
				}
				chResult <- struct {
					order   int
					status  string
					accrual float64
				}{order: in.order, status: js.Status, accrual: js.Accrual}
			}

		}
	}()
	return chResult
}
func (l Loader) StartLoader() {
	for {
		logger.Log.Info("loader: checking orders to update")
		orders, err := l.Depository.OrderGetOrdersInProcess()
		if err != nil {
			logger.Log.WithError(err).Error("loader: database error")
		} else {
			logger.Log.Info("loader: get orders for upload statuses", orders)
			l.UpdateOrdersStatuses(orders)
		}
		time.Sleep(l.Periodicity)
	}
}
