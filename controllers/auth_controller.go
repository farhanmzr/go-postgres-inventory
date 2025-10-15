package controllers

import (
	"net/http"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"go-postgres-inventory/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *gin.Context) {
	var input struct {
		Nama     string `json:"nama"`
		Password string `json:"password"`
		Role     string `json:"role"` // admin atau user
	}
	c.BindJSON(&input)

	if input.Role == "" {
		input.Role = "user" // default role
	}

	var existing models.User
	if err := config.DB.Where("nama = ?", input.Nama).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama sudah terdaftar"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(input.Password), 10)
	user := models.User{Nama: input.Nama, Password: string(hash), Role: input.Role}
	config.DB.Create(&user)

	c.JSON(http.StatusOK, gin.H{
		"message": "Registrasi berhasil",
		"user":    user.Nama,
		"role":    user.Role,
	})
}

func Login(c *gin.Context) {
	var input struct {
		Nama     string `json:"nama"`
		Password string `json:"password"`
	}
	c.BindJSON(&input)

	var user models.User
	if err := config.DB.Where("nama = ?", input.Nama).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Nama tidak ditemukan"})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password salah"})
		return
	}

	token, _ := utils.GenerateToken(user.ID, user.Nama, user.Role)

	c.JSON(http.StatusOK, gin.H{
		"message": "Login sukses",
		"token":   token,
		"role":    user.Role,
	})
}

func GetAllUsers(c *gin.Context) {
	// Ambil ID admin yang sedang login dari JWT middleware
	adminID, _ := c.Get("user_id")

	// var users []models.User
	// if err := config.DB.Find(&users).Error; err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pengguna"})
	// 	return
	// }
	var users []models.User
	// Cari semua user kecuali ID admin sendiri
	if err := config.DB.Where("id != ?", adminID).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pengguna"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_users": len(users),
		"data":        users,
	})
}

func Profile(c *gin.Context) {
	// Ambil token dari header Authorization: Bearer <token>
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak ditemukan"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	claims, err := utils.VerifyToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid"})
		return
	}

	// Ambil user dari DB
	var user models.User
	if err := config.DB.First(&user, claims["user_id"]).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nama": user.Nama,
		"role": user.Role,
	})
}

