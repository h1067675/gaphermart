package authorization

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"

	"github.com/h1067675/gophermart/internal/logger"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

type Autorizator interface {
	UserAuthorization(login string, password string) (userID int, err error)
}

const secretKey = "mysecretkey"

func CheckToken(tokenString string) (int, error) {
	var cl = Claims{}
	token, err := jwt.ParseWithClaims(tokenString, &cl, func(t *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil {
		return -1, err
	}
	if !token.Valid {
		err := fmt.Errorf("token is not valid")
		logger.Log.WithError(err).Info("token is not valid")
		return -1, err
	}
	logger.Log.Info(fmt.Sprintf("user id=%v restore from token", cl.UserID))
	return cl.UserID, nil
}

func SetToken(id int) (string, error) {
	var cl = Claims{
		UserID: id,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		logger.Log.WithError(err).Info("error token generate")
		return "", err
	}
	logger.Log.Info(fmt.Sprintf("create new token, user id=%v", cl.UserID))
	return tokenString, nil
}

// authorization user
func UserAuthorization(s Autorizator, login string, password string) (cookie string, err error) {
	var (
		token  string
		userID int
	)
	userID, err = s.UserAuthorization(login, password)
	if err != nil {
		logger.Log.WithError(err).Info("authorization error")
		return "", err
	}
	token, err = SetToken(userID)
	if err != nil {
		logger.Log.WithError(err).Info("token error")
		return "", err
	}
	return token, nil
}
