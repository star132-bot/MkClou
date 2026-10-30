package shop

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mkclou/server/internal/pkg/errcode"
	"mkclou/server/internal/pkg/secure"
)

// fakePayments 同时更新 fakeRepo.payments，使新手清单等读取环境与状态的逻辑看到同一份数据。
type fakePayments struct {
	repo  *fakeRepo
	byID  map[uint64]*PaymentConfig
	saves int
}

func (f *fakePayments) FindPayment(_ context.Context, shopID uint64) (*PaymentConfig, error) {
	p, ok := f.byID[shopID]
	if !ok {
		return nil, errNotFound
	}
	return p, nil
}

func (f *fakePayments) UpsertPayment(_ context.Context, p *PaymentConfig) error {
	f.saves++
	f.byID[p.ShopID] = p
	f.repo.mu.Lock()
	f.repo.payments[p.ShopID] = payment{env: p.Env, status: p.Status}
	f.repo.mu.Unlock()
	return nil
}

type fakeTester struct {
	err   error
	calls int
}

func (f *fakeTester) Test(context.Context, PaymentCredentials) error {
	f.calls++
	return f.err
}

type paymentEnv struct {
	*env
	payments *fakePayments
	tester   *fakeTester
	shopID   uint64
}

func setupPayment(t *testing.T) *paymentEnv {
	t.Helper()
	e := setup(t)
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	box, err := secure.NewBox(base64.StdEncoding.EncodeToString(k), 1)
	require.NoError(t, err)
	pe := &paymentEnv{env: e, payments: &fakePayments{repo: e.repo, byID: map[uint64]*PaymentConfig{}}, tester: &fakeTester{}}
	e.svc.payments, e.svc.tester, e.svc.box = pe.payments, pe.tester, box

	_, err = e.svc.Create(ctx(), 1, CreateInput{Name: "阿杰的工具铺", Slug: "ajie-tools"})
	require.NoError(t, err)
	sh, err := e.repo.FindByMerchant(ctx(), 1)
	require.NoError(t, err)
	pe.shopID = sh.ID
	return pe
}

// rsaKeys 生成一对 RSA 密钥，返回私钥对象与 Base64 编码的 PKCS#8 私钥、PKIX 公钥（与支付宝开放平台格式一致）。
func rsaKeys(t *testing.T) (*rsa.PrivateKey, string, string) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	priv, _ := x509.MarshalPKCS8PrivateKey(k)
	pub, _ := x509.MarshalPKIXPublicKey(&k.PublicKey)
	return k, base64.StdEncoding.EncodeToString(priv), base64.StdEncoding.EncodeToString(pub)
}

func TestPayment_NotConfigured(t *testing.T) {
	e := setupPayment(t)
	v, err := e.svc.Payment(ctx(), 1)
	require.NoError(t, err)
	assert.False(t, v.Configured)

	_, err = e.svc.Payment(ctx(), 2)
	requireCode(t, err, errcode.ShopNotCreated)
}

func TestSavePayment_EncryptsAndHidesPrivateKey(t *testing.T) {
	e := setupPayment(t)
	_, priv, pub := rsaKeys(t)
	pemPriv := "-----BEGIN PRIVATE KEY-----\n" + priv[:64] + "\n" + priv[64:] + "\n-----END PRIVATE KEY-----\n"

	v, err := e.svc.SavePayment(ctx(), 1, PaymentCredentials{Env: "sandbox", AppID: " 9021000123456789 ", PrivateKey: pemPriv, PublicKey: pub})
	require.NoError(t, err)
	assert.Equal(t, 1, e.tester.calls, "保存前服务端重新测试")
	assert.Equal(t, PaymentStatusActive, v.Status)
	assert.Equal(t, PaymentEnvSandbox, v.Env)
	assert.Equal(t, "9021000123456789", v.AppID)
	assert.Equal(t, priv[len(priv)-6:], v.PrivateKeyTail)

	stored := e.payments.byID[e.shopID]
	assert.NotContains(t, string(stored.PrivateKeyEnc), priv[:24], "数据库中不能出现明文私钥")
	plain, err := e.svc.DecryptPrivateKey(stored)
	require.NoError(t, err)
	assert.Equal(t, priv, plain, "PEM 头尾与换行已被规范化")

	moved := *stored
	moved.ShopID = e.shopID + 1
	_, err = e.svc.DecryptPrivateKey(&moved)
	assert.ErrorIs(t, err, secure.ErrDecrypt, "密文挪到其他店铺的记录上无法解密")

	o, err := e.svc.Onboarding(ctx(), 1)
	require.NoError(t, err)
	assert.True(t, o.PaymentReady, "新手清单显示收款已配置")
}

func TestSavePayment_InvalidatesPublicCache(t *testing.T) {
	e := setupPayment(t)
	_, err := e.svc.Public(ctx(), "ajie-tools")
	require.NoError(t, err)
	require.True(t, e.redis.Exists(publicCacheKey("ajie-tools")), "买家端店铺信息已缓存")

	_, priv, pub := rsaKeys(t)
	_, err = e.svc.SavePayment(ctx(), 1, PaymentCredentials{Env: PaymentEnvSandbox, AppID: "9021000123456789", PrivateKey: priv, PublicKey: pub})
	require.NoError(t, err)
	assert.False(t, e.redis.Exists(publicCacheKey("ajie-tools")), "保存后删除缓存，测试模式标记立即更新")
}

func TestSavePayment_RejectsWhenTestFails(t *testing.T) {
	e := setupPayment(t)
	_, priv, pub := rsaKeys(t)
	e.tester.err = &TestError{Field: "appId", Message: "AppID 不存在", Detail: "40002 isv.invalid-app-id"}

	_, err := e.svc.SavePayment(ctx(), 1, PaymentCredentials{Env: PaymentEnvSandbox, AppID: "9021000123456789", PrivateKey: priv, PublicKey: pub})
	ce := requireCode(t, err, errcode.PaymentTestFailed)
	assert.Contains(t, ce.Message, "AppID")
	assert.Contains(t, ce.Data.(map[string]any)["fields"], "appId", "错误定位到表单字段")
	assert.Zero(t, e.payments.saves, "测试失败不保存")

	e.tester.err = errors.New("network down")
	_, err = e.svc.SavePayment(ctx(), 1, PaymentCredentials{Env: PaymentEnvSandbox, AppID: "9021000123456789", PrivateKey: priv, PublicKey: pub})
	assert.EqualError(t, err, "network down", "非预期错误原样返回，由响应层记录日志并返回 500")
}

func TestSavePayment_BannedShop(t *testing.T) {
	e := setupPayment(t)
	e.repo.shops[e.shopID].Status = StatusBanned
	_, priv, pub := rsaKeys(t)
	_, err := e.svc.SavePayment(ctx(), 1, PaymentCredentials{Env: PaymentEnvSandbox, AppID: "9021000123456789", PrivateKey: priv, PublicKey: pub})
	requireCode(t, err, errcode.StateConflict)
	assert.Zero(t, e.tester.calls)
}

func TestValidateCredentials(t *testing.T) {
	_, priv, pub := rsaKeys(t)
	ok := PaymentCredentials{Env: PaymentEnvSandbox, AppID: "9021000123456789", PrivateKey: priv, PublicKey: pub}
	require.NoError(t, validateCredentials(ok))

	cases := map[string]func(*PaymentCredentials){
		"env":        func(c *PaymentCredentials) { c.Env = "TEST" },
		"appId":      func(c *PaymentCredentials) { c.AppID = "123" },
		"privateKey": func(c *PaymentCredentials) { c.PrivateKey = "short" },
		"publicKey":  func(c *PaymentCredentials) { c.PublicKey = priv }, // 把私钥误填到公钥栏
	}
	for field, mutate := range cases {
		c := ok
		mutate(&c)
		ce := requireCode(t, validateCredentials(c), errcode.InvalidParams)
		assert.Contains(t, ce.Data.(map[string]any)["fields"], field)
	}
}

// alipayResponse 模拟支付宝响应：用“支付宝私钥”对 response 节点的原始 JSON 签名。
func alipayResponse(t *testing.T, k *rsa.PrivateKey, node string) []byte {
	t.Helper()
	h := sha256.Sum256([]byte(node))
	sig, err := rsa.SignPKCS1v15(rand.Reader, k, crypto.SHA256, h[:])
	require.NoError(t, err)
	return []byte(fmt.Sprintf(`{"alipay_trade_query_response":%s,"sign":"%s"}`, node, base64.StdEncoding.EncodeToString(sig)))
}

func TestInterpretQueryResponse(t *testing.T) {
	alipayKey, _, alipayPub := rsaKeys(t)
	_, _, otherPub := rsaKeys(t)
	notExist := `{"code":"40004","msg":"Business Failed","sub_code":"ACQ.TRADE_NOT_EXIST","sub_msg":"交易不存在"}`

	assert.NoError(t, interpretQueryResponse(alipayResponse(t, alipayKey, notExist), alipayPub),
		"交易不存在 + 签名验证通过 = 三项参数都正确")

	fieldOf := func(err error) string {
		var te *TestError
		require.ErrorAs(t, err, &te)
		return te.Field
	}
	assert.Equal(t, "publicKey", fieldOf(interpretQueryResponse(alipayResponse(t, alipayKey, notExist), otherPub)),
		"支付宝公钥填错能被识别")
	assert.Equal(t, "publicKey", fieldOf(interpretQueryResponse([]byte(`{"alipay_trade_query_response":`+notExist+`}`), alipayPub)),
		"缺少签名时无法确认公钥")

	cases := map[string]string{
		`{"code":"40002","sub_code":"isv.invalid-app-id","sub_msg":"无效的AppID参数"}`:       "appId",
		`{"code":"40002","sub_code":"isv.invalid-signature","sub_msg":"验签出错"}`:           "privateKey",
		`{"code":"40006","sub_code":"isv.insufficient-isv-permissions","sub_msg":"权限不足"}`: "appId",
		`{"code":"20000","sub_code":"isp.unknow-error","sub_msg":"系统繁忙"}`:                "",
	}
	for node, field := range cases {
		assert.Equal(t, field, fieldOf(interpretQueryResponse([]byte(`{"alipay_trade_query_response":`+node+`}`), alipayPub)), node)
	}

	assert.Equal(t, "", fieldOf(interpretQueryResponse([]byte(`<html>502 Bad Gateway</html>`), alipayPub)), "非 JSON 响应")
}
