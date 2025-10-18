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

type AdminRegisterInput struct {
	Username string `json:"username" binding:"required"`
	FullName string `json:"full_name" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

func AdminRegister(c *gin.Context) {
	var in AdminRegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	var exists models.Admin
	if err := config.DB.Where("username = ?", in.Username).First(&exists).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username sudah dipakai"}); return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	admin := models.Admin{
		Username:     in.Username,
		FullName:     in.FullName,
		PasswordHash: string(hash),
		IsActive:     true,
	}

	if err := config.DB.Create(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat admin"}); return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Admin dibuat", "username": admin.Username})
}

type AdminLoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func AdminLogin(c *gin.Context) {
	var in AdminLoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	var admin models.Admin
	if err := config.DB.Where("username = ? AND is_active = true", in.Username).First(&admin).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin tidak ditemukan"}); return
	}

	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(in.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password salah"}); return
	}

	token, _ := utils.GenerateAdminToken(admin.ID, admin.Username, 24*time.Hour)
	c.JSON(http.StatusOK, gin.H{
		"message": "Login admin sukses",
		"token":   token,
	})
}

// Admin: lihat semua user (contoh endpoint admin)
func AdminGetAllUsers(c *gin.Context) {
	var users []models.User
	if err := config.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal ambil data pengguna"}); return
	}
	c.JSON(http.StatusOK, gin.H{"total": len(users), "data": users})
}

// Admin: buat user operasional
type CreateUserInput struct {
	Username     string `json:"username" binding:"required"`
	FullName     string `json:"full_name" binding:"required"`
	Password     string `json:"password" binding:"required,min=6"`
	UserCode     string `json:"user_code"`
	Position     string `json:"position"`
	WorkLocation string `json:"work_location"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	AvatarURL    string `json:"avatar_url"`
}

func AdminCreateUser(c *gin.Context) {
	var in CreateUserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	var exists models.User
	if err := config.DB.Where("username = ?", in.Username).First(&exists).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username sudah dipakai"}); return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	user := models.User{
		Username:     in.Username,
		FullName:     in.FullName,
		UserCode:     in.UserCode,
		Position:     in.Position,
		WorkLocation: in.WorkLocation,
		Phone:        in.Phone,
		Address:      in.Address,
		AvatarURL:    in.AvatarURL,
		PasswordHash: string(hash),
		IsActive:     true,
	}
	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat user"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User dibuat", "username": user.Username})
}

// Admin: set permissions user
type SetUserPermissionsInput struct {
	PermissionCodes []string `json:"permission_codes"`
}

func AdminSetUserPermissions(c *gin.Context) {
	userID := c.Param("userID") // string
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"}); return
	}

	var in SetUserPermissionsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	// Ambil permission ID dari codes
	var perms []models.Permission
	if len(in.PermissionCodes) > 0 {
		if err := config.DB.Where("code IN ?", in.PermissionCodes).Find(&perms).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kode permission tidak valid"}); return
		}
	}

	// Replace all permissions user -> teknik sederhana: delete then insert
	if err := config.DB.Where("user_id = ?", user.ID).Delete(&models.UserPermission{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal reset permission"}); return
	}
	for _, p := range perms {
		config.DB.Create(&models.UserPermission{
			UserID:       user.ID,
			PermissionID: p.ID,
			GrantedAt:    time.Now(),
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permissions disimpan", "applied": len(perms)})
}
