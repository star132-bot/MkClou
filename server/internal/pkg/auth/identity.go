package auth

import "github.com/gin-gonic/gin"

const identityKey = "mk.identity"

// Identity 是已登录商家的身份，由鉴权中间件写入请求上下文。
type Identity struct {
	MerchantID uint64
	SessionID  string
}

func SetIdentity(c *gin.Context, id Identity) { c.Set(identityKey, id) }

// IdentityFrom 读取当前商家身份。只应在经过鉴权中间件的路由中调用。
func IdentityFrom(c *gin.Context) (Identity, bool) {
	v, ok := c.Get(identityKey)
	if !ok {
		return Identity{}, false
	}
	id, ok := v.(Identity)
	return id, ok
}
