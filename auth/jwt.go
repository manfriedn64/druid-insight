package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(secret string, username string, isAdmin bool, expirationMinutes int) (string, error) {
	claims := jwt.MapClaims{
		"sub":   username,
		"admin": isAdmin,
		"exp":   time.Now().Add(time.Duration(expirationMinutes) * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ExtractUserAndAdminFromJWT(r *http.Request, secret string) (username string, isAdmin bool, err error) {
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		return "", false, errors.New("no bearer token")
	}
	tokenString := strings.TrimPrefix(auth, "Bearer ")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", false, errors.New("invalid or expired JWT")
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		username, _ = claims["sub"].(string)
		switch v := claims["admin"].(type) {
		case bool:
			isAdmin = v
		case string:
			isAdmin = v == "true" || v == "1"
		case float64:
			isAdmin = v == 1
		}
		return username, isAdmin, nil
	}
	return "", false, errors.New("invalid JWT claims")
}
