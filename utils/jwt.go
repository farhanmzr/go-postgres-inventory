package utils

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey = []byte("rahasia-super-kuat")

func GenerateToken(userID uint, nama string, role string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"nama":    nama,
		"role":    role,
	})
	return token.SignedString(secretKey)
}

func VerifyToken(tokenString string) (jwt.MapClaims, error) {
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("token tidak valid")
}
