package shop

// 收款配置（PRD SHOP-04，接口清单 #25 ～ #27）。
//
// 资金直连：商家填写自己在支付宝开放平台创建的应用参数，买家付款直接进入商家账户，平台不经手资金。
// 应用私钥使用 AES-256-GCM 加密存储（PRD SEC-06），永不回显；保存前服务端用参数实际调用支付宝验证。

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mkclou/server/internal/pkg/errcode"
)

const (
	PaymentEnvProduction  = "PRODUCTION"
	PaymentStatusInvalid  = "INVALID"
	PaymentProviderAlipay = "ALIPAY"
)

// PaymentConfig 对应 shop_payment_configs 表。私钥只以密文形式存在。
type PaymentConfig struct {
	ID             uint64 `gorm:"primaryKey"`
	ShopID         uint64
	Provider       string
	Env            string
	AppID          string
	PrivateKeyEnc  []byte
	PrivateKeyTail string
	PlatformPubKey string
	KeyVersion     int16
	Status         string
	VerifiedAt     *time.Time
	LastError      *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (PaymentConfig) TableName() string { return "shop_payment_configs" }

// PaymentView 是返回给商家的收款配置状态，不含私钥，只返回末尾 6 位用于识别。
type PaymentView struct {
	Configured     bool       `json:"configured"`
	Provider       string     `json:"provider"`
	Env            string     `json:"env"`
	AppID          string     `json:"appId"`
	PrivateKeyTail string     `json:"privateKeyTail"`
	Status         string     `json:"status"`
	VerifiedAt     *time.Time `json:"verifiedAt"`
	LastError      *string    `json:"lastError"`
}

func (p *PaymentConfig) view() *PaymentView {
	return &PaymentView{Configured: true, Provider: p.Provider, Env: p.Env, AppID: p.AppID,
		PrivateKeyTail: p.PrivateKeyTail, Status: p.Status, VerifiedAt: p.VerifiedAt, LastError: p.LastError}
}

// ---------- 存储 ----------

// PaymentRepository 读写收款配置。与 Repository.PaymentConfig 操作同一张表：后者只读环境与状态，
// 供新手清单与买家端“测试模式”判断使用。
type PaymentRepository interface {
	FindPayment(ctx context.Context, shopID uint64) (*PaymentConfig, error)
	UpsertPayment(ctx context.Context, p *PaymentConfig) error
}

type gormPaymentRepository struct{ db *gorm.DB }

func NewPaymentRepository(db *gorm.DB) PaymentRepository { return &gormPaymentRepository{db: db} }

func (r *gormPaymentRepository) FindPayment(ctx context.Context, shopID uint64) (*PaymentConfig, error) {
	var p PaymentConfig
	err := r.db.WithContext(ctx).Where("shop_id = ? AND provider = ?", shopID, PaymentProviderAlipay).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errNotFound
	}
	return &p, err
}

// UpsertPayment 依赖唯一索引 uk_shop_provider：每个店铺每个支付渠道只有一条配置，重复保存即覆盖。
func (r *gormPaymentRepository) UpsertPayment(ctx context.Context, p *PaymentConfig) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "shop_id"}, {Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"env", "app_id", "private_key_enc", "private_key_tail", "platform_pub_key",
			"key_version", "status", "verified_at", "last_error", "updated_at",
		}),
	}).Create(p).Error
}

// ---------- 业务 ----------

// PaymentCredentials 是商家填写的支付宝开放平台应用参数。
type PaymentCredentials struct {
	Env        string
	AppID      string
	PrivateKey string
	PublicKey  string // 支付宝公钥（不是应用公钥）
}

// Payment 返回收款配置状态（接口 #25）。
func (s *Service) Payment(ctx context.Context, merchantID uint64) (*PaymentView, error) {
	sh, err := s.mine(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	p, err := s.payments.FindPayment(ctx, sh.ID)
	if errors.Is(err, errNotFound) {
		return &PaymentView{Provider: PaymentProviderAlipay}, nil
	}
	if err != nil {
		return nil, err
	}
	return p.view(), nil
}

// TestPayment 只测试不保存（接口 #26）。
func (s *Service) TestPayment(ctx context.Context, merchantID uint64, c PaymentCredentials) error {
	if _, err := s.mine(ctx, merchantID); err != nil {
		return err
	}
	return s.runPaymentTest(ctx, normalizeCredentials(c))
}

// SavePayment 服务端重新测试，通过后加密保存（接口 #27）。
// 不信任前端“已测试通过”的状态，防止绕过测试直接保存错误参数。
func (s *Service) SavePayment(ctx context.Context, merchantID uint64, c PaymentCredentials) (*PaymentView, error) {
	sh, err := s.mine(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if sh.Status == StatusBanned {
		return nil, errcode.StateConflict.WithMessage("店铺已被封禁，如有疑问请联系平台客服")
	}
	c = normalizeCredentials(c)
	if err := s.runPaymentTest(ctx, c); err != nil {
		return nil, err
	}

	enc, err := s.box.Encrypt([]byte(c.PrivateKey), paymentAAD(sh.ID))
	if err != nil {
		return nil, err
	}
	now := s.now()
	p := &PaymentConfig{
		ShopID: sh.ID, Provider: PaymentProviderAlipay, Env: c.Env, AppID: c.AppID,
		PrivateKeyEnc: enc, PrivateKeyTail: c.PrivateKey[len(c.PrivateKey)-6:], PlatformPubKey: c.PublicKey,
		KeyVersion: s.box.Version(), Status: PaymentStatusActive, VerifiedAt: &now, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.payments.UpsertPayment(ctx, p); err != nil {
		return nil, err
	}
	// 买家端店铺信息包含“测试模式”标记，环境变化后需要刷新缓存
	s.invalidate(ctx, sh.Slug)
	return p.view(), nil
}

// DecryptPrivateKey 供订单模块调起支付时使用；明文只在内存中短暂存在。
func (s *Service) DecryptPrivateKey(p *PaymentConfig) (string, error) {
	b, err := s.box.Decrypt(p.PrivateKeyEnc, paymentAAD(p.ShopID))
	return string(b), err
}

// paymentAAD 把私钥密文绑定到所属店铺：密文被复制到其他店铺的记录上时无法解密。
func paymentAAD(shopID uint64) []byte { return []byte(fmt.Sprintf("shop-payment:%d", shopID)) }

func (s *Service) runPaymentTest(ctx context.Context, c PaymentCredentials) error {
	if err := validateCredentials(c); err != nil {
		return err
	}
	err := s.tester.Test(ctx, c)
	if err == nil {
		return nil
	}
	var te *TestError
	if !errors.As(err, &te) {
		return err
	}
	s.log.Info("payment test failed", zap.String("appId", c.AppID), zap.String("env", c.Env), zap.String("detail", te.Detail))
	data := map[string]any{"detail": te.Detail}
	if te.Field != "" {
		data["fields"] = map[string]string{te.Field: te.Message}
	}
	return errcode.PaymentTestFailed.WithMessage(te.Message).WithData(data)
}

// ---------- 参数校验 ----------

var (
	appIDPattern = regexp.MustCompile(`^\d{16}$`)
	pemStrip     = regexp.MustCompile(`-----[A-Z ]+-----|\s`)
)

// normalizeCredentials 去掉 PEM 头尾与所有空白：商家常把带头尾或换行的内容直接粘贴进来。
func normalizeCredentials(c PaymentCredentials) PaymentCredentials {
	c.Env = strings.ToUpper(strings.TrimSpace(c.Env))
	c.AppID = strings.TrimSpace(c.AppID)
	c.PrivateKey = pemStrip.ReplaceAllString(c.PrivateKey, "")
	c.PublicKey = pemStrip.ReplaceAllString(c.PublicKey, "")
	return c
}

func validateCredentials(c PaymentCredentials) error {
	if c.Env != PaymentEnvSandbox && c.Env != PaymentEnvProduction {
		return fieldError("env", "请选择环境：沙箱或正式")
	}
	if !appIDPattern.MatchString(c.AppID) {
		return fieldError("appId", "AppID 为 16 位数字")
	}
	if len(c.PrivateKey) < 100 {
		return fieldError("privateKey", "请粘贴完整的应用私钥")
	}
	der, err := base64.StdEncoding.DecodeString(c.PublicKey)
	if err != nil {
		return fieldError("publicKey", "支付宝公钥格式不正确")
	}
	if _, err := x509.ParsePKIXPublicKey(der); err != nil {
		return fieldError("publicKey", "支付宝公钥格式不正确，请确认复制的是“支付宝公钥”而不是“应用公钥”或私钥")
	}
	return nil
}

// ---------- 支付宝连接测试 ----------

// PaymentTester 测试收款参数是否可用。接口化便于单元测试替换。
type PaymentTester interface {
	Test(ctx context.Context, c PaymentCredentials) error
}

// TestError 是可展示给商家的测试失败原因。
type TestError struct {
	Field   string // 关联的表单字段，便于前端定位
	Message string
	Detail  string // 支付宝返回的错误码，便于排查
}

func (e *TestError) Error() string { return e.Message }

// AlipayTester 用商家参数调用“交易查询”接口查询一个不存在的订单号（PRD 02 3.3）：
// 返回“交易不存在”且响应签名能用商家填写的支付宝公钥验证通过，说明 AppID、应用私钥、支付宝公钥三项都正确。
type AlipayTester struct {
	Timeout time.Duration
}

func (t AlipayTester) Test(ctx context.Context, c PaymentCredentials) error {
	client, err := alipay.NewClient(c.AppID, c.PrivateKey, c.Env == PaymentEnvProduction)
	if err != nil {
		return &TestError{Field: "privateKey", Message: "应用私钥格式不正确，请粘贴“应用私钥”的完整内容"}
	}
	timeout := t.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	bm := make(gopay.BodyMap).Set("out_trade_no", fmt.Sprintf("MKTEST%d", time.Now().UnixNano()))
	body, err := client.DoAliPay(ctx, bm, "alipay.trade.query")
	if err != nil {
		return &TestError{Message: "支付宝服务暂时不可用，请稍后重试", Detail: err.Error()}
	}
	return interpretQueryResponse(body, c.PublicKey)
}

// interpretQueryResponse 解析交易查询的原始响应并验签。
// 支付宝对 alipay_trade_query_response 节点的原始 JSON 文本签名，json.RawMessage 会原样保留这段字节。
func interpretQueryResponse(body []byte, publicKey string) error {
	var env struct {
		Response json.RawMessage `json:"alipay_trade_query_response"`
		Sign     string          `json:"sign"`
	}
	unknown := &TestError{Message: "支付宝返回了无法识别的响应，请稍后重试"}
	if err := json.Unmarshal(body, &env); err != nil || len(env.Response) == 0 {
		return unknown
	}
	var r struct {
		Code    string `json:"code"`
		SubCode string `json:"sub_code"`
		SubMsg  string `json:"sub_msg"`
	}
	if err := json.Unmarshal(env.Response, &r); err != nil {
		return unknown
	}
	detail := strings.TrimSpace(r.Code + " " + r.SubCode)

	switch {
	case r.Code == "10000", r.Code == "40004" && r.SubCode == "ACQ.TRADE_NOT_EXIST":
		// 请求已被支付宝受理：AppID 与应用私钥正确，接下来验证支付宝公钥
	case strings.Contains(r.SubCode, "invalid-app-id"), strings.Contains(r.SubCode, "app-not-exist"):
		return &TestError{Field: "appId", Message: "AppID 不存在，请检查 AppID 或环境（沙箱 / 正式）是否选对", Detail: detail}
	case strings.Contains(r.SubCode, "invalid-signature"):
		return &TestError{Field: "privateKey", Message: "应用私钥与开放平台上配置的应用公钥不匹配", Detail: detail}
	case strings.Contains(r.SubCode, "insufficient-isv-permissions"):
		return &TestError{Field: "appId", Message: "该应用尚未开通“电脑网站支付”等交易能力，请在开放平台签约后重试", Detail: detail}
	default:
		msg := "支付宝返回错误"
		if r.SubMsg != "" {
			msg += "：" + r.SubMsg
		}
		return &TestError{Message: msg, Detail: detail}
	}

	if env.Sign == "" {
		return &TestError{Field: "publicKey", Message: "无法验证支付宝公钥，请稍后重试", Detail: detail}
	}
	if _, err := alipay.VerifySyncSign(publicKey, string(env.Response), env.Sign); err != nil {
		return &TestError{Field: "publicKey", Message: "支付宝公钥不正确，请确认复制的是“支付宝公钥”而不是“应用公钥”", Detail: detail}
	}
	return nil
}

var _ PaymentTester = AlipayTester{}
