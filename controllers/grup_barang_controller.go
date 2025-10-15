package controllers

import (
    "net/http"
    "strconv"

    "go-postgres-inventory/config"
    "go-postgres-inventory/models"

    "github.com/gin-gonic/gin"
)

func CreateGrupBarang(c *gin.Context) {
    var input struct {
        Nama string `json:"nama"`
        Kode string `json:"kode"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
        return
    }

    kodeGrup := models.GrupBarang{
        Nama: input.Nama,
        Kode: input.Kode,
    }

    if err := config.DB.Create(&kodeGrup).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Kode grup berhasil ditambahkan", "data": kodeGrup})
}

func GetAllGrupBarang(c *gin.Context) {
    var grups []models.GrupBarang
    if err := config.DB.Find(&grups).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": grups})
}

func GetGrupBarangByID(c *gin.Context) {
    idParam := c.Param("id")
    id, err := strconv.Atoi(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
        return
    }

    var grup models.GrupBarang
    if err := config.DB.First(&grup, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Kode grup tidak ditemukan"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"data": grup})
}

func UpdateGrupBarang(c *gin.Context) {
    idParam := c.Param("id")
    id, err := strconv.Atoi(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
        return
    }

    var grup models.GrupBarang
    if err := config.DB.First(&grup, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Kode grup tidak ditemukan"})
        return
    }

    var input struct {
        Nama string `json:"nama"`
        Kode string `json:"kode"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
        return
    }

    updateData := models.GrupBarang{
        Nama: input.Nama,
        Kode: input.Kode,
    }

    if err := config.DB.Model(&grup).Updates(updateData).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Kode grup berhasil diupdate", "data": grup})
}

func DeleteGrupBarang(c *gin.Context) {
    idParam := c.Param("id")
    id, err := strconv.Atoi(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
        return
    }

    var grup models.GrupBarang
    if err := config.DB.First(&grup, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Kode grup tidak ditemukan"})
        return
    }

    if err := config.DB.Delete(&grup).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hapus kode grup"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Kode grup berhasil dihapus"})
}
