package jwt

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)

type Claims struct {
	Guid string
	Exp  string
	Type string
}

var secret = []byte(os.Getenv("JWT_SECRET"))

func NewAccessToken(GUID string, duration time.Duration) (string, error) {
	token := jwt.New(jwt.SigningMethodHS512)

	claims := token.Claims.(jwt.MapClaims)
	claims["guid"] = GUID
	claims["exp"] = time.Now().Add(duration).Unix()
	claims["type"] = "access"

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT:%w", err)
	}

	return tokenString, nil
}

func NewRefreshToken(GUID string, duration time.Duration) (string, error) {
	token := jwt.New(jwt.SigningMethodHS512)

	claims := token.Claims.(jwt.MapClaims)
	claims["guid"] = GUID
	claims["exp"] = time.Now().Add(duration).Unix()
	claims["type"] = "refresh"

	secret := os.Getenv("JWT_SECRET")

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT:%w", err)
	}

	return tokenString, nil
}

func VerifyToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) { return secret, nil })
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil

}
