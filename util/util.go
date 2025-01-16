package util

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nchc-ai/rfstack/model"
)

func RespondWithError(c *gin.Context, code int, format string, args ...interface{}) {
	resp := genericResponse(true, format, args...)
	c.JSON(code, resp)
	c.Abort()
}

func genericResponse(isError bool, format string, args ...interface{}) model.GenericResponse {
	resp := model.GenericResponse{
		Error:   isError,
		Message: fmt.Sprintf(format, args...),
	}
	return resp
}

func RespondWithOk(c *gin.Context, format string, args ...interface{}) {
	resp := genericResponse(false, format, args...)
	c.JSON(http.StatusOK, resp)
	c.Abort()
}
