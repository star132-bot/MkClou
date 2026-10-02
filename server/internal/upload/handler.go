package upload

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"mkclou/server/internal/pkg/auth"
	"mkclou/server/internal/pkg/errcode"
	"mkclou/server/internal/pkg/ratelimit"
	"mkclou/server/internal/pkg/response"
)

// ShopResolver 根据商家查出其店铺 ID，由店铺模块实现。未开店时返回 errcode.ShopNotCreated。
type ShopResolver interface {
	ShopID(ctx context.Context, merchantID uint64) (uint64, error)
}

type Handler struct {
	images  *ImageService
	shops   ShopResolver
	limiter *ratelimit.Limiter
}

func NewHandler(images *ImageService, shops ShopResolver, limiter *ratelimit.Limiter) *Handler {
	return &Handler{images: images, shops: shops, limiter: limiter}
}

// Register 注册路由（接口清单 #28）。
func (h *Handler) Register(api *gin.RouterGroup, requireMerchant gin.HandlerFunc) {
	g := api.Group("/uploads", requireMerchant)
	g.POST("/images", h.limiter.ByIP(ratelimit.PerMinute("upload-image", 60)), h.uploadImage)
}

func (h *Handler) uploadImage(c *gin.Context) {
	id, _ := auth.IdentityFrom(c)
	shopID, err := h.shops.ShopID(c.Request.Context(), id.MerchantID)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 请求体上限 = 图片上限 + multipart 头部余量
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxImageBytes+64<<10)
	fh, err := c.FormFile("file")
	if err != nil {
		response.Error(c, errcode.InvalidParams.WithMessage("请选择不超过 5MB 的图片"))
		return
	}
	if fh.Size > MaxImageBytes {
		response.Error(c, errcode.InvalidParams.WithMessage("图片不能超过 5MB，请压缩后重新上传"))
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.Error(c, err)
		return
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, MaxImageBytes+1))
	if err != nil {
		response.Error(c, err)
		return
	}

	r, err := h.images.Upload(c.Request.Context(), shopID, data)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, r)
}
