package controllers

import (
    "net/http"
    "strconv"

    "go-postgres-inventory/config"
    "go-postgres-inventory/models"

    "github.com/gin-gonic/gin"
)

func CreateSupplier(c *gin.Context) {
    var input struct {
        Nama string `json:"nama"`
        Kode string `json:"kode"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
        return
    }

    supplier := models.Supplier{
        Nama: input.Nama,
        Kode: input.Kode,
    }

    if err := config.DB.Create(&supplier).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Supplier berhasil ditambahkan", "data": supplier})
}

func GetAllSupplier(c *gin.Context) {
    var grups []models.Supplier
    if err := config.DB.Find(&grups).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": grups})
}

func GetSupplierByID(c *gin.Context) {
    idParam := c.Param("id")
    id, err := strconv.Atoi(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
        return
    }

    var grup models.Supplier
    if err := config.DB.First(&grup, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Supplier tidak ditemukan"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"data": grup})
}

func UpdateSupplier(c *gin.Context) {
    idParam := c.Param("id")
    id, err := strconv.Atoi(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
        return
    }

    var grup models.Supplier
    if err := config.DB.First(&grup, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Supplier tidak ditemukan"})
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

    updateData := models.Supplier{
        Nama: input.Nama,
        Kode: input.Kode,
    }

    if err := config.DB.Model(&grup).Updates(updateData).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Supplier berhasil diupdate", "data": grup})
}

func DeleteSupplier(c *gin.Context) {
    idParam := c.Param("id")
    id, err := strconv.Atoi(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
        return
    }

    var grup models.Supplier
    if err := config.DB.First(&grup, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Supplier tidak ditemukan"})
        return
    }

    if err := config.DB.Delete(&grup).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hapus supplier"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Supplier berhasil dihapus"})
}