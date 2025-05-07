package jwt

import (
	"crypto/sha512"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	Guid string
	Exp  string
	Type string
}

func NewAccessToken(GUID string, duration time.Duration) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	token := jwt.New(jwt.SigningMethodHS512)

	claims := token.Claims.(jwt.MapClaims)
	claims["guid"] = GUID
	claims["exp"] = time.Now().Add(duration).Unix()
	claims["type"] = "access"
	claims["created_at"] = time.Now().Minute()

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT:%w", err)
	}

	return tokenString, nil
}

func NewRefreshToken(GUID string, duration time.Duration) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	token := jwt.New(jwt.SigningMethodHS512)

	claims := token.Claims.(jwt.MapClaims)
	claims["guid"] = GUID
	claims["exp"] = time.Now().Add(duration).Unix()
	claims["type"] = "refresh"
	claims["created_at"] = time.Now().Minute()

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT:%w", err)
	}

	return tokenString, nil
}
func VerifyToken(tokenString string, expectedType string) (*jwt.Token, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	// Парсим токен с проверкой подписи
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Проверяем тип токена
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims format")
	}

	// Проверяем наличие обязательных полей
	if claims["type"] != expectedType {
		return nil, fmt.Errorf("invalid token type, expected %s", expectedType)
	}

	if claims["guid"] == nil {
		return nil, fmt.Errorf("missing guid claim")
	}

	// Дополнительная проверка срока действия (jwt.Parse уже проверяет exp)
	// Но можно добавить кастомные проверки:
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().After(time.Unix(int64(exp), 0)) {
			return nil, fmt.Errorf("token expired")
		}
	}

	return token, nil
}

func HashJWTbcrypt(jwt string) (string, error) {
	shaHash := sha512.Sum512([]byte(jwt))
	shaHashStr := string(shaHash[:])

	hashedJwtToken, err := bcrypt.GenerateFromPassword([]byte(shaHashStr), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedJwtToken), nil
}
