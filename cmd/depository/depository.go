package depository

import (
	"context"
	"time"

	"github.com/h1067675/gophermart/internal/configurer"
	"github.com/h1067675/gophermart/internal/logger"
)

type Storager interface {
}

type Storage struct {
	DB *SQLDB
}

// create new storage
func InitializeStorager(config configurer.Config) *Storage {
	var r = Storage{
		DB: newDB(config.DatabaseURI.String()),
	}
	if r.DB.Connected {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := r.DB.DB.PingContext(ctx); err != nil {
			return nil
		}
		logger.Log.Info("connection to the database has been established and verified")
		if config.ReloadTables {
			if err := r.dropDBTables(); err != nil {
				logger.Log.WithError(err).Info("error droping tables in DB")
			}
		}
		if err := r.createDBTables(); err != nil {
			logger.Log.WithError(err).Error("error creating tables in DB")
			return nil
		}
		return &r
	}
	return nil
}
