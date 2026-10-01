// Package response 封装统一响应格式：{ code, message, data, traceId }。
package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mkclou/server/internal/pkg/errcode"
)

// TraceIDKey 是 traceId 在 gin.Context 中的键，由 RequestID 中间件写入。
const TraceIDKey = "traceId"

type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	TraceID string `json:"traceId"`
}

// Page 是分页列表的统一结构。
type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: 0, Message: "ok", Data: data, TraceID: c.GetString(TraceIDKey)})
}

// Error 输出错误响应。业务错误按其定义返回；其他错误一律记录日志并返回 500，不向客户端泄露细节。
func Error(c *gin.Context, err error) {
	var e *errcode.Error
	if !errors.As(err, &e) {
		if l, ok := c.Get("logger"); ok {
			l.(*zap.Logger).Error("unhandled error", zap.Error(err))
		}
		e = errcode.Internal
	}
	c.AbortWithStatusJSON(e.HTTP, Body{Code: e.Code, Message: e.Message, Data: e.Data, TraceID: c.GetString(TraceIDKey)})
}
