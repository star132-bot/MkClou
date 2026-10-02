// Package errcode 定义业务错误码，与 docs/api-list.md 第 1.6 节保持一致。
package errcode

import (
	"fmt"
	"net/http"
)

// Error 是可直接返回给客户端的业务错误。
type Error struct {
	Code    int    // 业务错误码
	HTTP    int    // HTTP 状态码
	Message string // 面向用户的默认文案
	Data    any    // 附加数据，如字段错误、最新价格
}

func (e *Error) Error() string { return fmt.Sprintf("errcode %d: %s", e.Code, e.Message) }

// WithMessage 返回替换了文案的副本，不修改预定义错误。
func (e *Error) WithMessage(msg string) *Error {
	cp := *e
	cp.Message = msg
	return &cp
}

// WithData 返回附带数据的副本，不修改预定义错误。
func (e *Error) WithData(data any) *Error {
	cp := *e
	cp.Data = data
	return &cp
}

func newErr(code, httpStatus int, msg string) *Error {
	return &Error{Code: code, HTTP: httpStatus, Message: msg}
}

// 通用 10000-19999
var (
	InvalidParams   = newErr(10001, http.StatusBadRequest, "参数校验失败")
	TooManyRequests = newErr(10002, http.StatusTooManyRequests, "操作太频繁，请稍后再试")
	NotFound        = newErr(10003, http.StatusNotFound, "资源不存在")
	StateConflict   = newErr(10004, http.StatusConflict, "当前状态不允许该操作")
	VersionConflict = newErr(10005, http.StatusConflict, "数据已在其他地方被修改，请刷新后重试")
	Internal        = newErr(10500, http.StatusInternalServerError, "服务异常，请稍后重试")
)

// 认证 20000-29999
var (
	Unauthorized     = newErr(20000, http.StatusUnauthorized, "请先登录")
	BadCredentials   = newErr(20001, http.StatusUnauthorized, "邮箱或密码错误")
	CaptchaRequired  = newErr(20002, http.StatusUnauthorized, "请输入图形验证码")
	AccountLocked    = newErr(20003, http.StatusForbidden, "登录尝试过多，请稍后再试")
	EmailNotVerified = newErr(20004, http.StatusForbidden, "请先验证邮箱")
	TokenInvalid     = newErr(20005, http.StatusUnauthorized, "链接已失效")
	AccountDisabled  = newErr(20006, http.StatusForbidden, "账号已被停用")
	EmailRegistered  = newErr(20007, http.StatusConflict, "该邮箱已注册")
	CodeInvalid      = newErr(20008, http.StatusBadRequest, "验证码错误")
	Forbidden        = newErr(20403, http.StatusForbidden, "无权访问")
)

// 店铺 30000-39999
var (
	SlugUnavailable    = newErr(30001, http.StatusConflict, "该店铺链接不可用")
	SlugChangeTooSoon  = newErr(30002, http.StatusConflict, "店铺链接 30 天内只能修改一次")
	ShopNotCreated     = newErr(30003, http.StatusNotFound, "请先创建店铺")
	PaymentTestFailed  = newErr(30004, http.StatusBadRequest, "收款配置测试失败")
	ShopPaused         = newErr(30005, http.StatusConflict, "店铺暂停营业中")
	PaymentUnavailable = newErr(30006, http.StatusConflict, "店铺收款未配置或已失效")
)

// 商品 40000-49999
var (
	PublishCheckFailed   = newErr(40001, http.StatusUnprocessableEntity, "还有内容需要完善才能上架")
	ProductUnavailable   = newErr(40002, http.StatusConflict, "商品暂时无法购买")
	ProductHasOrders     = newErr(40003, http.StatusConflict, "该商品已有订单，无法删除，可以选择下架")
	DeliveryTypeLocked   = newErr(40004, http.StatusConflict, "商品上架后不能修改交付类型，如需修改请复制商品")
	ContentBlocked       = newErr(40005, http.StatusBadRequest, "内容包含违规词，请修改后再提交")
	FileUploadIncomplete = newErr(40006, http.StatusConflict, "文件尚未上传完成")
	CardPrecheckExpired  = newErr(40007, http.StatusConflict, "卡密导入预检已过期，请重新导入")
)
