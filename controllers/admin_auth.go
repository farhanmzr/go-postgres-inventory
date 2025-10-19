package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"go-postgres-inventory/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminRegisterInput struct {
	Username string `json:"username" binding:"required"`
	FullName string `json:"full_name" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

func AdminRegister(c *gin.Context) {
	var in AdminRegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var exists models.Admin
	if err := config.DB.Where("username = ?", in.Username).First(&exists).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username sudah dipakai"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	admin := models.Admin{
		Username:     in.Username,
		FullName:     in.FullName,
		PasswordHash: string(hash),
		IsActive:     true,
	}

	if admin.AvatarURL == "" {
		admin.AvatarURL = utils.DefaultAvatar(in.FullName)
	}

	if err := config.DB.Create(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat admin"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Admin berhasil dibuat", "username": admin.Username})
}

// Admin: Login
type AdminLoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func AdminLogin(c *gin.Context) {
	var in AdminLoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var admin models.Admin
	if err := config.DB.Where("username = ? AND is_active = true", in.Username).First(&admin).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin tidak ditemukan"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(in.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password salah"})
		return
	}

	token, _ := utils.GenerateAdminToken(admin.ID, admin.Username, 24*time.Hour)
	c.JSON(http.StatusOK, gin.H{
		"message": "Login admin sukses",
		"token":   token,
	})
}

// Admin: Get Data Admin
func GetDataAdminProfile(c *gin.Context) {
	aid, _ := c.Get("admin_id")

	var admin models.Admin

	if err := config.DB.First(&admin, aid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Admin tidak ditemukan",
			"error":   err.Error(),
		})
		return
	}

	if admin.AvatarURL == "" {
		admin.AvatarURL = utils.DefaultAvatar(admin.FullName)
	} else {
		admin.AvatarURL = utils.CloudinaryThumb256(admin.AvatarURL)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil data admin",
		"data":    admin, // PasswordHash tersembunyi karena json:"-"
	})
}

// Admin: Update data profile
type AdminUpdateProfileInput struct {
	FullName  *string `json:"full_name,omitempty"`
	AdminCode *string `json:"admin_code,omitempty"`
	Position  *string `json:"position,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Address   *string `json:"address,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

func AdminUpdateProfile(c *gin.Context) {
	// --- normalisasi admin_id dari context ---
	rawID, _ := c.Get("admin_id")
	var adminID uint
	switch v := rawID.(type) {
	case uint:
		adminID = v
	case int:
		adminID = uint(v)
	case int64:
		adminID = uint(v)
	case float64:
		adminID = uint(v)
	case string:
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			adminID = uint(n)
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"message": "admin_id tidak valid"})
		return
	}

	var admin models.Admin
	if err := config.DB.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Admin tidak ditemukan",
			"error":   err.Error(),
		})
		return
	}

	var in AdminUpdateProfileInput
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
	if in.AdminCode != nil {
		updates["admin_code"] = *in.AdminCode
	}
	if in.Position != nil {
		updates["position"] = *in.Position
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

	if err := config.DB.Model(&admin).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal memperbarui profil admin",
			"error":   err.Error(),
		})
		return
	}

	if err := config.DB.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat ulang profil admin", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profil admin berhasil diperbarui",
		"data":    admin, // PasswordHash sudah tersembunyi via json:"-"
	})
}

// Admin: Ganti password
type AdminChangePasswordInput struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

func AdminChangePassword(c *gin.Context) {
	adminID, _ := c.Get("admin_id")

	var admin models.Admin
	if err := config.DB.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Admin tidak ditemukan"})
		return
	}

	var in AdminChangePasswordInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Payload tidak valid",
			"error":   err.Error(),
		})
		return
	}

	// Verifikasi password lama
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(in.CurrentPassword)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Password lama salah",
		})
		return
	}

	// Hash password baru
	hashed, _ := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	if err := config.DB.Model(&admin).Update("password_hash", string(hashed)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal mengganti password",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password admin berhasil diganti",
	})
}

// Admin: lihat semua user (contoh endpoint admin)
func AdminGetAllUsers(c *gin.Context) {
	var users []models.User
	if err := config.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal ambil data pengguna"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil mengambil semua data Users", "total": len(users), "data": users})
}

// Input gabungan (create user + permissions)
type CreateUserWithPermsInput struct {
	Username        string   `json:"username"       binding:"required"`
	FullName        string   `json:"full_name"      binding:"required"`
	Password        string   `json:"password"       binding:"required,min=6"`
	UserCode        string   `json:"user_code"`
	Position        string   `json:"position"`
	WorkLocation    string   `json:"work_location"`
	Phone           string   `json:"phone"`
	Address         string   `json:"address"`
	AvatarURL       string   `json:"avatar_url"`
	PermissionCodes []string `json:"permission_codes"` // contoh: ["PURCHASE","SALES"]
}

// Admin: buat user + set permissions (ATOMIK)
func AdminCreateUser(c *gin.Context) {
	var in CreateUserWithPermsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	in.Username = strings.TrimSpace(in.Username)
	in.FullName = strings.TrimSpace(in.FullName)
	if in.Username == "" || in.FullName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username/full_name wajib"})
		return
	}

	// Validasi unik username (case-insensitive)
	var exists int64
	config.DB.Model(&models.User{}).
		Where("LOWER(username) = LOWER(?)", in.Username).
		Count(&exists)
	if exists > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username sudah dipakai"})
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
		return
	}

	var newUser models.User
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Create user
		newUser = models.User{
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
		if err := tx.Create(&newUser).Error; err != nil {
			return err
		}

		// 2) Ambil permissions dari codes (boleh kosong)
		var perms []models.Permission
		if len(in.PermissionCodes) > 0 {
			if err := tx.Where("code IN ?", in.PermissionCodes).Find(&perms).Error; err != nil {
				return err
			}
			// Validasi: jika ada code yang tidak ada di DB
			if len(perms) != len(in.PermissionCodes) {
				return gorm.ErrRecordNotFound
			}
		}

		// 3) Insert user_permissions (bulk)
		if len(perms) > 0 {
			now := time.Now()
			records := make([]models.UserPermission, 0, len(perms))
			for _, p := range perms {
				records = append(records, models.UserPermission{
					UserID:       newUser.ID,
					PermissionID: p.ID,
					GrantedAt:    now,
				})
			}
			if err := tx.Create(&records).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kode permission tidak valid"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                   "User + permissions berhasil dibuat",
		"id":                        newUser.ID,
		"username":                  newUser.Username,
		"applied_permission_codes":  in.PermissionCodes,
	})
}

// Admin: set (replace) permissions user
type SetUserPermissionsInput struct {
	PermissionCodes []string `json:"permission_codes"`
}

func AdminSetUserPermissions(c *gin.Context) {
	userID := c.Param("userID")
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	var in SetUserPermissionsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// fetch permissions by code
		var perms []models.Permission
		if len(in.PermissionCodes) > 0 {
			if err := tx.Where("code IN ?", in.PermissionCodes).Find(&perms).Error; err != nil {
				return err
			}
			if len(perms) != len(in.PermissionCodes) {
				return gorm.ErrRecordNotFound
			}
		}
		// clear existing
		if err := tx.Where("user_id = ?", user.ID).Delete(&models.UserPermission{}).Error; err != nil {
			return err
		}
		// re-insert
		if len(perms) > 0 {
			now := time.Now()
			bulk := make([]models.UserPermission, 0, len(perms))
			for _, p := range perms {
				bulk = append(bulk, models.UserPermission{
					UserID:       user.ID,
					PermissionID: p.ID,
					GrantedAt:    now,
				})
			}
			if err := tx.Create(&bulk).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kode permission tidak valid"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal reset permission"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permissions disimpan", "applied": len(in.PermissionCodes)})
}

// Admin: list permissions (untuk UI centang)
func AdminListPermissions(c *gin.Context) {
	var perms []models.Permission
	if err := config.DB.Order("code asc").Find(&perms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal ambil permission"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": perms})
}
