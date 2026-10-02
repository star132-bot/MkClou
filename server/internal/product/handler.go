package product

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mkclou/server/internal/pkg/auth"
	"mkclou/server/internal/pkg/errcode"
	"mkclou/server/internal/pkg/response"
)

type Handler struct {
	svc       *Service
	market    *Market
	favorites *Favorites
}

func NewHandler(svc *Service, market *Market, favorites *Favorites) *Handler {
	return &Handler{svc: svc, market: market, favorites: favorites}
}

// Register 注册路由（接口清单 #34 ～ #39、#42、#73、#74）。
func (h *Handler) Register(api *gin.RouterGroup, requireMerchant gin.HandlerFunc) {
	m := api.Group("/products", requireMerchant)
	m.GET("", h.list)
	m.POST("", h.create)
	m.GET("/:publicId", h.get)
	m.PUT("/:publicId", h.update)
	m.POST("/:publicId/publish", h.publish)
	m.POST("/:publicId/unpublish", h.unpublish)
	m.DELETE("/:publicId", h.delete)

	api.GET("/public/products/:publicId", h.publicDetail)
	api.GET("/public/shops/:slug/products", h.shopProducts)
	api.GET("/public/categories", h.categories)
	api.GET("/public/market/home", h.home)
	api.GET("/public/market/search", h.search)

	fav := api.Group("/me/favorites", requireMerchant)
	fav.GET("", h.listFavorites)
	fav.GET("/status", h.favoriteStatus)
	fav.PUT("/:publicId", h.addFavorite)
	fav.DELETE("/:publicId", h.removeFavorite)
}

func (h *Handler) listFavorites(c *gin.Context) {
	r, err := h.favorites.List(c.Request.Context(), merchantID(c), intQuery(c, "page"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, r)
}

func (h *Handler) favoriteStatus(c *gin.Context) {
	var ids []string
	for _, id := range strings.Split(c.Query("ids"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	r, err := h.favorites.Status(c.Request.Context(), merchantID(c), ids)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"favorited": r})
}

func (h *Handler) addFavorite(c *gin.Context) {
	n, err := h.favorites.Add(c.Request.Context(), merchantID(c), c.Param("publicId"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"favorited": true, "favoriteCount": n})
}

func (h *Handler) removeFavorite(c *gin.Context) {
	if err := h.favorites.Remove(c.Request.Context(), merchantID(c), c.Param("publicId")); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"favorited": false})
}

func (h *Handler) home(c *gin.Context) {
	r, err := h.market.Home(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, r)
}

func (h *Handler) search(c *gin.Context) {
	r, err := h.market.Search(c.Request.Context(), SearchInput{
		Query: c.Query("q"), Category: c.Query("category"), Price: c.Query("price"),
		Sort: c.Query("sort"), Page: intQuery(c, "page"),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, r)
}

func merchantID(c *gin.Context) uint64 {
	id, _ := auth.IdentityFrom(c)
	return id.MerchantID
}

func bind(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		response.Error(c, errcode.InvalidParams.WithMessage("请求格式不正确"))
		return false
	}
	return true
}

func intQuery(c *gin.Context, key string) int {
	n, _ := strconv.Atoi(c.Query(key))
	return n
}

func (h *Handler) list(c *gin.Context) {
	page, err := h.svc.List(c.Request.Context(), merchantID(c), ListFilter{
		Status: c.Query("status"), Query: c.Query("q"), Page: intQuery(c, "page"), PageSize: intQuery(c, "pageSize"),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, page)
}

func (h *Handler) create(c *gin.Context) {
	var req struct {
		Name         string `json:"name"`
		DeliveryType string `json:"deliveryType"`
		Price        int    `json:"price"`
	}
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.Create(c.Request.Context(), merchantID(c), CreateInput{Name: req.Name, DeliveryType: req.DeliveryType, Price: req.Price})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, v)
}

func (h *Handler) get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), merchantID(c), c.Param("publicId"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, v)
}

func (h *Handler) update(c *gin.Context) {
	var req struct {
		Version        int            `json:"version"`
		Name           string         `json:"name"`
		Tagline        string         `json:"tagline"`
		Category       *string        `json:"category"`
		Price          int            `json:"price"`
		OriginalPrice  *int           `json:"originalPrice"`
		DeliveryType   string         `json:"deliveryType"`
		DescriptionMD  string         `json:"descriptionMd"`
		Detail         Detail         `json:"detail"`
		DeliveryConfig DeliveryConfig `json:"deliveryConfig"`
		MaxPerOrder    int            `json:"maxPerOrder"`
		Images         []ImageInput   `json:"images"`
	}
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.Update(c.Request.Context(), merchantID(c), c.Param("publicId"), UpdateInput{
		Version: req.Version, Name: req.Name, Tagline: req.Tagline, Category: req.Category,
		Price: req.Price, OriginalPrice: req.OriginalPrice, DeliveryType: req.DeliveryType,
		DescriptionMD: req.DescriptionMD, Detail: req.Detail, DeliveryConfig: req.DeliveryConfig,
		MaxPerOrder: req.MaxPerOrder, Images: req.Images,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, v)
}

func (h *Handler) publish(c *gin.Context) {
	v, err := h.svc.Publish(c.Request.Context(), merchantID(c), c.Param("publicId"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, v)
}

func (h *Handler) unpublish(c *gin.Context) {
	v, err := h.svc.Unpublish(c.Request.Context(), merchantID(c), c.Param("publicId"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, v)
}

func (h *Handler) delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), merchantID(c), c.Param("publicId")); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *Handler) publicDetail(c *gin.Context) {
	p, err := h.svc.PublicDetail(c.Request.Context(), c.Param("publicId"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, p)
}

func (h *Handler) shopProducts(c *gin.Context) {
	items, next, err := h.svc.ShopProducts(c.Request.Context(), c.Param("slug"), c.Query("cursor"), intQuery(c, "limit"))
	if err != nil {
		response.Error(c, err)
		return
	}
	var nextCursor *string
	if next != "" {
		nextCursor = &next
	}
	response.OK(c, gin.H{"items": items, "nextCursor": nextCursor})
}

func (h *Handler) categories(c *gin.Context) {
	response.OK(c, gin.H{"items": Categories})
}
