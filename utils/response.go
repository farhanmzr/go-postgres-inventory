package utils

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

func Success(c *gin.Context, message string, data interface{}) {
    c.JSON(http.StatusOK, gin.H{
        "message": message,
        "data":    data,
    })
}

func Error(c *gin.Context, status int, message string, err error) {
    resp := gin.H{"message": message}
    if err != nil {
        resp["error"] = err.Error()
    }
    c.JSON(status, resp)
}
