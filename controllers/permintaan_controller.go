package controllers

import (
	"errors"
	"net/http"
	"time"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
)


func currentUserID(c *gin.Context) (uint, error) {
	// Sesuaikan key dengan middleware-mu
	v, ok := c.Get("user_id")
	if !ok {
		return 0, errors.New("user_id tidak ada di context")
	}
	id, ok := v.(uint)
	if !ok || id == 0 {
		return 0, errors.New("user_id tidak valid")
	}
	return id, nil
}

func CreatePermintaan(c *gin.Context) {

    var input struct {
        Keterangan string `json:"keterangan"`
        NamaPeminta string `json:"nama_peminta"`
        KodePeminta string `json:"kode_peminta"`
        TanggalPermintaan time.Time `json:"tanggal_permintaan"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
        return
    }

	uid, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized", "error": err.Error()})
		return
	}

    permintaan := models.Permintaan{
        Keterangan: input.Keterangan,
        NamaPeminta: input.NamaPeminta,
        KodePeminta: input.KodePeminta,
        TanggalPermintaan: input.TanggalPermintaan,
		CreatedByID: uid,
    }

    if err := config.DB.Create(&permintaan).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Berhasil membuat Permintaan", "data": permintaan})
}

func GetMyPermintaan(c *gin.Context) {

	uid, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized", "error": err.Error()})
		return
	}

    var grups []models.Permintaan
    if err := config.DB.
		Where("created_by_id = ?", uid).
		Order("id DESC").
		Find(&grups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengambil data", "error": err.Error()})
		return
	}
    c.JSON(http.StatusOK, gin.H{"message": "Berhasil mengambil data Permintaan", "data": grups})
}

func AdminGetAllPermintaan(c *gin.Context) {
	
	var grups []models.Permintaan
	if err := config.DB.
		Order("id DESC").
		Find(&grups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengambil data", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": grups})
}