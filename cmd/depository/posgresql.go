package depository

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/h1067675/gophermart/internal/logger"
)

type SQLDB struct {
	*sql.DB
	Connected bool
}

type dbTable struct {
	table    string
	pgxQuery string
}

var dbTables = []dbTable{
	{
		table: "users",
		pgxQuery: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				login TEXT UNIQUE, 
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
				hash_password TEXT
				);`,
	},
	{
		table: "orders",
		pgxQuery: `CREATE TABLE orders (
				id SERIAL PRIMARY KEY,
				order_number NUMERIC UNIQUE,
				uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
				accrual DOUBLE PRECISION DEFAULT 0.0, 
				status TEXT
				);`,
	},
	{
		table: "users_orders",
		pgxQuery: `CREATE TABLE users_orders (
				order_id INTEGER UNIQUE, 
				user_id INTEGER
				);`,
	},
	{
		table: "users_balance",
		pgxQuery: `CREATE TABLE users_balance (
				user_id INTEGER UNIQUE, 
				balance DOUBLE PRECISION DEFAULT 0.0, 
				withdrawal DOUBLE PRECISION DEFAULT 0.0
				);`,
	},
	{
		table: "users_transactions",
		pgxQuery: `CREATE TABLE users_transactions (
				user_id INTEGER, 
				order_id NUMERIC,
				processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
				sum DOUBLE PRECISION DEFAULT 0.0,
				withdrawal BOOL,
				balance DOUBLE PRECISION DEFAULT 0.0
				);`,
	},
}

// create DB
func newDB(dbPath string) *SQLDB {
	r, err := sql.Open("pgx", dbPath)
	if err != nil {
		logger.Log.WithError(err).Info("error of connect to DB")
		return &SQLDB{Connected: false}
	}
	return &SQLDB{DB: r, Connected: true}
}

// create tables in DB
func (s *Storage) createDBTables() error {
	var err error
	for _, t := range dbTables {
		if !s.checkLinksDBTable(t.table) {
			logger.Log.WithError(err).Infof("try to create %s table", t.table)
			_, err = s.DB.Exec(t.pgxQuery)
			if err != nil {
				logger.Log.WithError(err).Infof("the %s table could not be created", t.table)
				return err
			}
			logger.Log.Infof("the %s table id created", t.table)
		}
	}
	return err
}

// check exist table in DB
func (s *Storage) checkLinksDBTable(table string) bool {
	rows, err := s.DB.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 1;", table))
	if err != nil {
		logger.Log.WithError(err).Infof("the %s does not exist", table)
		return false
	}
	defer func() {
		_ = rows.Close()
		err = rows.Err() // or modify return value
		if err != nil {
			logger.Log.WithError(err).Info("error of close rows")
		}
	}()
	return true
}

// drop tables in DB
func (s *Storage) dropDBTables() error {
	var err error
	for _, t := range dbTables {
		_, err = s.DB.Query(fmt.Sprintf("DROP TABLE %s;", t.table))
		if err != nil {
			logger.Log.WithError(err).Infof("the %s table could not be droped", t.table)
			return err
		}
		logger.Log.Infof("the %s table is droped", t.table)
	}
	return err
}
