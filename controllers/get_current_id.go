package controllers

import (
	"errors"

	"github.com/gin-gonic/gin"
)

func currentAdminID(c *gin.Context) (uint, error) {
    v, ok := c.Get("admin_id")
    if !ok {
        return 0, errors.New("admin_id tidak ada di context")
    }
    id, ok := v.(uint)
    if !ok || id == 0 {
        return 0, errors.New("admin_id tidak valid")
    }
    return id, nil
}

func currentUserID(c *gin.Context) (uint, error) {
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
