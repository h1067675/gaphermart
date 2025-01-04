package configurer

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v6"

	"github.com/h1067675/gophermart/internal/logger"
)

type Configurer interface {
	InitializeConfigurer(server string, db string, system string, reload bool) *Config
}

// structure of the server settings
type Config struct {
	RunAddress           NetAddress
	DatabaseURI          DatabasePath
	AccrualSystemAddress NetAddress
	ReloadTables         bool
}

// structure of the network address format
type NetAddress struct {
	Host string
	Port int
}

func (c Config) GetRunAddress() string {
	return c.RunAddress.String()
}

func (c Config) GetDatabaseURI() string {
	return c.DatabaseURI.String()
}

func (c Config) GetAccrualSystemAddress() string {
	return c.AccrualSystemAddress.String()
}

func (c Config) GetReloadTables() bool {
	return c.ReloadTables
}

// возвращаем адрес вида host:port
func (n *NetAddress) String() string {
	return fmt.Sprint(n.Host + ":" + strconv.Itoa(n.Port))
}

// устанавливаем значения host и port в переменные
func (n *NetAddress) Set(s string) (err error) {
	n.Host, n.Port, err = checkNetAddress(s, n.Host, n.Port)
	if err != nil {
		logger.Log.WithError(err).Info("error of set NetAdress")
		return err
	}
	return nil
}

type DatabasePath struct {
	Path string
}

// Сохраняет значение переменной среды
func (n *DatabasePath) Set(s string) (err error) {
	n.Path = s
	return nil
}

// возвращаем путь файла
func (n *DatabasePath) String() string {
	return n.Path
}

// structure of environment variables
type EnvConfig struct {
	RunAddress           string `env:"RUN_ADDRESS"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

// сreating server settins
func (c Config) InitializeConfigurer(server string, db string, system string, reload bool) *Config {
	var result = Config{ // set defaul settins
		ReloadTables: reload,
	}
	result.RunAddress.Set(server)
	result.DatabaseURI.Set(db)
	result.AccrualSystemAddress.Set(system)
	result.ParseFlags()
	result.EnvConfigSet()
	return &result
}

// функция проверяющая на корректность указания пары host:port и в случае ошибки передающей значения по умолчанию
func checkNetAddress(s string, h string, p int) (host string, port int, e error) {
	host, port = h, p
	v := strings.Split(s, "://")
	if len(v) < 1 || len(v) > 2 {
		e = errors.New("incorrect net address")
		return
	}
	if len(v) == 2 {
		s = v[1]
	}
	a := strings.Split(s, ":")
	if len(a) < 1 || len(a) > 2 {
		e = errors.New("incorrect net address")
		return
	}
	// ПРОВЕРИТЬ!!! что выдаст если дойдет до сюда по дальше выдаст ошибку
	// Написать тесты
	host = a[0]
	ip := net.ParseIP(host)
	if ip == nil && host != "localhost" {
		e = errors.New("incorrect net address")
		return
	}
	if a[1] != "" {
		port, e = strconv.Atoi(a[1])
		if e != nil || port < 0 || port > 65535 {
			e = errors.New("incorrect net address")
			return
		}
	}
	return
}

// разбираем атрибуты командной строки
func (c *Config) ParseFlags() {
	flag.Var(&c.RunAddress, "a", "Network address for runing server (host:port)")
	flag.Var(&c.DatabaseURI, "d", "Network address for connecting to the database (host:port)")
	flag.Var(&c.AccrualSystemAddress, "r", "Network address of the accrual calculation system (host:port)")
	flag.Parse()
}

// set settins from environment variables
func (c *Config) EnvConfigSet() {
	var envCnf EnvConfig
	err := env.Parse(&envCnf)
	if err != nil {
		logger.Log.WithError(err).Error("error of set settins from env")
	}
	if envCnf.RunAddress != "" {
		c.RunAddress.Set(envCnf.RunAddress)
	}
	if envCnf.DatabaseURI != "" {
		c.DatabaseURI.Set(envCnf.DatabaseURI)
	}
	if envCnf.DatabaseURI != "" {
		c.DatabaseURI.Set(envCnf.DatabaseURI)
	}
}
