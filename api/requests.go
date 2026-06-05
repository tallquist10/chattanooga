package api

import (
	"github.com/gin-gonic/gin"
)

func FormatJsonInput[T any](c *gin.Context) (*T, error) {
	var output T
	// Decode from the request body stream directly into the struct
	err := c.BindJSON(&output)
	if err != nil {
		return nil, err
	}
	return &output, nil
}

func WriteResponse[T any](c *gin.Context, code int, val *T) {
	c.JSON(code, val)
}
