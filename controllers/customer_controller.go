package controllers

import (
    "net/http"
    "strconv"

    "go-postgres-inventory/config"
    "go-postgres-inventory/models"

    "github.com/gin-gonic/gin"
)

func CreateCustomer(c *gin.Context) {
    var input struct {
        Nama string `json:"nama"`
        Kode string `json:"kode"`
        Seri string `json:"seri"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
        return
    }

	// Cek apakah kode customer sudah ada
	var exist models.Customer
	if err := config.DB.Where("kode = ?", input.Kode).First(&exist).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kode Customer sudah digunakan"})
		return
	}

    customer := models.Customer{
        Nama: input.Nama,
        Kode: input.Kode,
        Seri: input.Seri,
    }

    if err := config.DB.Create(&customer).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Customer berhasil ditambahkan", "data": customer})
}

func GetAllCustomer(c *gin.Context) {
    var grups []models.Customer
    if err := config.DB.Find(&grups).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "Berhasil mengambil data Customer", "data": grups})
}

func GetCustomerByID(c *gin.Context) {
    idParam := c.Param("id")
    id, err := strconv.Atoi(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
        return
    }

    var grup models.Customer
    if err := config.DB.First(&grup, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Customer tidak ditemukan"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Berhasil mengambil detail data Customer", "data": grup})
}

func UpdateCustomer(c *gin.Context) {
    idParam := c.Param("id")
    id, err := strconv.Atoi(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
        return
    }

    var grup models.Customer
    if err := config.DB.First(&grup, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Customer tidak ditemukan"})
        return
    }

    var input struct {
        Nama string `json:"nama"`
        Kode string `json:"kode"`
        Seri string `json:"seri"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
        return
    }

	// Cek apakah kode customer sudah ada
	var exist models.Customer
	if err := config.DB.Where("kode = ?", input.Kode).First(&exist).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kode Customer sudah digunakan"})
		return
	}

    updateData := models.Customer{
        Nama: input.Nama,
        Kode: input.Kode,
        Seri: input.Seri,
    }

    if err := config.DB.Model(&grup).Updates(updateData).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Customer berhasil diupdate", "data": grup})
}

func DeleteCustomer(c *gin.Context) {
    idParam := c.Param("id")
    id, err := strconv.Atoi(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
        return
    }

    var grup models.Customer
    if err := config.DB.First(&grup, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Customer tidak ditemukan"})
        return
    }

    if err := config.DB.Delete(&grup).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hapus Customer"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Customer berhasil dihapus"})
}