package main

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/h1067675/gophermart/cmd/depository"
	"github.com/h1067675/gophermart/cmd/loader"
	"github.com/h1067675/gophermart/internal/configurer"
	"github.com/h1067675/gophermart/internal/logger"
)

type key int

const (
	KeyUserID key = iota
	KeyNewUser
)

func main() {
	var logger = logger.InitializeLogger(&log.JSONFormatter{}, log.InfoLevel, os.Stdout)
	var config configurer.Config
	conf := config.InitializeConfigurer("localhost:8080", "host=127.0.0.1 port=5432 dbname=postgres user=postgres password=12345678 connect_timeout=10 sslmode=prefer", "127.0.0.1:8090", false)
	var depositary = depository.InitializeStorager(conf)
	var connector = InitializeRouter(depositary, conf)
	var loader = loader.InitializeLoader(depositary, conf.GetAccrualSystemAddress(), time.Second*1)
	go loader.StartLoader()
	logger.Info("loader is running. Outer server: ", loader.Server)
	connector.StartServer()

}
