package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	AdminSecret = []byte("ADMIN_SUPER_SECRET") // set via env di produksi
	UserSecret  = []byte("USER_SUPER_SECRET")  // set via env di produksi

	AdminIssuer  = "inventory-api"
	UserIssuer   = "inventory-api"
	AdminAudience = "admin-app"
	UserAudience  = "user-app"
)

type AdminClaims struct {
	jwt.RegisteredClaims
	Kind     string `json:"kind"` // "admin"
	AdminID  uint   `json:"admin_id"`
	Username string `json:"username"`
}

type UserClaims struct {
	jwt.RegisteredClaims
	Kind       string   `json:"kind"` // "user"
	UserID     uint     `json:"user_id"`
	Username   string   `json:"username"`
	Permissions []string `json:"perms"`
}

func GenerateAdminToken(adminID uint, username string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := AdminClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    AdminIssuer,
			Audience:  []string{AdminAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Kind:     "admin",
		AdminID:  adminID,
		Username: username,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(AdminSecret)
}

func GenerateUserToken(userID uint, username string, perms []string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    UserIssuer,
			Audience:  []string{UserAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Kind:        "user",
		UserID:      userID,
		Username:    username,
		Permissions: perms,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(UserSecret)
}

func VerifyAdminToken(tokenString string) (*AdminClaims, error) {
	tok, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(t *jwt.Token) (interface{}, error) {
		return AdminSecret, nil
	})
	if err != nil || !tok.Valid {
		return nil, errors.New("token admin tidak valid")
	}
	claims, ok := tok.Claims.(*AdminClaims)
	if !ok {
		return nil, errors.New("claims admin tidak valid")
	}
	// // Opsional: cek audience
	// if !claims.VerifyAudience(AdminAudience, true) {
	// 	return nil, errors.New("audience admin salah")
	// }
	return claims, nil
}

func VerifyUserToken(tokenString string) (*UserClaims, error) {
	tok, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		return UserSecret, nil
	})
	if err != nil || !tok.Valid {
		return nil, errors.New("token user tidak valid")
	}
	claims, ok := tok.Claims.(*UserClaims)
	if !ok {
		return nil, errors.New("claims user tidak valid")
	}
	// if !claims.VerifyAudience(UserAudience, true) {
	// 	return nil, errors.New("audience user salah")
	// }
	return claims, nil
}
