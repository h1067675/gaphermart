package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/h1067675/gophermart/cmd/depository"
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
	conf := config.InitializeConfigurer("localhost:8080", "host=127.0.0.1 port=5432 dbname=postgres user=postgres password=12345678 connect_timeout=10 sslmode=prefer", "127.0.0.1:8090", true)
	var depositary = depository.InitializeStorager(conf)
	var connector = InitializeRouter(depositary, conf)
	// var loader = loader.InitializeLoader(depositary, conf.GetAccrualSystemAddress(), time.Second*5)
	// go loader.StartLoader()
	logger.Info("loader is running")
	connector.StartServer()

}

// 03/12/2024 from 17-40 to 19-06 1.26
// 04/12/2024 from 17-52 to 18-47 0.55
// 06/12/2024 from 17-10 to 18-59 1.49
// 06/12/2024 from 22-59 to 23-59 1.00
// 03/12/2024 from 20-00 to 23-02 3.02
// 15/12/2024 from 12:40 to 17:00 4.20
// 22/12/2024 from 18:36 to 23:17 4.41
// 17.13
// 22/12/2024 from 14:12 to
