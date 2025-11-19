package jwts

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	UserID string `json:"userID"`
	jwt.RegisteredClaims
}

func GetToken(claims *CustomClaims, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(token, secret string) (string, error) {
	parse, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := parse.Claims.(jwt.MapClaims); ok && parse.Valid {
		return fmt.Sprintf("%v", claims["userID"]), nil
	}

	return "", errors.New("token not valid")
}
