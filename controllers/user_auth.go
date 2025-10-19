package controllers

import (
	"net/http"
	"time"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"go-postgres-inventory/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserLoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func UserLogin(c *gin.Context) {
	var in UserLoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.DB.Where("username = ? AND is_active = true", in.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak ditemukan"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password salah"})
		return
	}

	// Ambil permissions
	type Row struct{ Code string }
	var rows []Row
	config.DB.Raw(`
		SELECT p.code FROM permissions p
		JOIN user_permissions up ON up.permission_id = p.id
		WHERE up.user_id = ?`, user.ID).Scan(&rows)

	perms := make([]string, 0, len(rows))
	for _, r := range rows {
		perms = append(perms, r.Code)
	}

	token, _ := utils.GenerateUserToken(user.ID, user.Username, perms, 24*time.Hour)

	c.JSON(http.StatusOK, gin.H{
		"message": "Login user sukses",
		"token":   token,
		"perms":   perms,
	})
}

func UserProfile(c *gin.Context) {
	uid, _ := c.Get("user_id")
	var user models.User
	if err := config.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil profil pengguna",
		"data":    user,
	})
}