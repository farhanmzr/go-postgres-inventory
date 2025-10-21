package controllers

import (
	"net/http"
	"strconv"
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

// Admin: Update data profile
type UserUpdateProfileInput struct {
	FullName  *string `json:"full_name,omitempty"`
	UserCode *string `json:"user_code,omitempty"`
	Position  *string `json:"position,omitempty"`
	WorkLocation  *string `json:"work_location,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Address   *string `json:"address,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

func UserUpdateProfile(c *gin.Context) {
	// --- normalisasi user_id dari context ---
	rawID, _ := c.Get("user_id")
	var userID uint
	switch v := rawID.(type) {
	case uint:
		userID = v
	case int:
		userID = uint(v)
	case int64:
		userID = uint(v)
	case float64:
		userID = uint(v)
	case string:
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			userID = uint(n)
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"message": "admin_id tidak valid"})
		return
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User tidak ditemukan",
			"error":   err.Error(),
		})
		return
	}

	var in UserUpdateProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Payload tidak valid",
			"error":   err.Error(),
		})
		return
	}

	updates := map[string]any{}
	if in.FullName != nil {
		updates["full_name"] = *in.FullName
	}
	if in.UserCode != nil {
		updates["user_code"] = *in.UserCode
	}
	if in.Position != nil {
		updates["position"] = *in.Position
	}
	if in.WorkLocation != nil {
		updates["work_location"] = *in.WorkLocation
	}
	if in.Phone != nil {
		updates["phone"] = *in.Phone
	}
	if in.Address != nil {
		updates["address"] = *in.Address
	}
	if in.AvatarURL != nil {
		updates["avatar_url"] = *in.AvatarURL
	}

	// tolak kalau memang tidak ada perubahan
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Tidak ada data yang diubah"})
		return
	}

	// baru set updated_at setelah ada changes
	updates["updated_at"] = time.Now()

	if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal memperbarui profil User",
			"error":   err.Error(),
		})
		return
	}

	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat ulang profil User", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profil User berhasil diperbarui",
		"data":    user, // PasswordHash sudah tersembunyi via json:"-"
	})
}

// Admin: Ganti password
type UserChangePasswordInput struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

func UserChangePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User tidak ditemukan"})
		return
	}

	var in UserChangePasswordInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Payload tidak valid",
			"error":   err.Error(),
		})
		return
	}

	// Verifikasi password lama
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.CurrentPassword)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Password lama salah",
		})
		return
	}

	// Hash password baru
	hashed, _ := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	if err := config.DB.Model(&user).Update("password_hash", string(hashed)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal mengganti password",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password User berhasil diganti",
	})
}