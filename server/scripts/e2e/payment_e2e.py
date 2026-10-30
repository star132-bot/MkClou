# 收款配置端到端测试（接口 #25 ～ #27）：需先启动 docker compose 依赖与 API。
# 运行：python scripts/e2e/payment_e2e.py（在 server 目录下，依赖 cryptography 包）
# 最后一项会访问支付宝沙箱网关（用不存在的 AppID），需要能连外网。
import base64, json, sys, time, urllib.error, urllib.request, http.cookiejar
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa

API = "http://127.0.0.1:18080/api/v1"
results = []


def check(name, cond, detail=""):
    results.append(bool(cond))
    print(("PASS " if cond else "FAIL ") + name + (f"  [{detail}]" if detail and not cond else ""))


class Client:
    def __init__(self):
        self.op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
        self.token = None

    def req(self, method, path, body=None):
        h = {"Content-Type": "application/json"}
        if self.token:
            h["Authorization"] = "Bearer " + self.token
        r = urllib.request.Request(API + path, data=json.dumps(body).encode() if body is not None else None, headers=h, method=method)
        try:
            with self.op.open(r, timeout=30) as resp:
                return resp.status, json.loads(resp.read())
        except urllib.error.HTTPError as e:
            return e.code, json.loads(e.read() or b"{}")


stamp = int(time.time() * 1000)
c = Client()
s, b = c.req("POST", "/auth/register", {"email": f"pay-{stamp}@example.com", "password": "Passw0rd!", "agreeTerms": True})
assert s == 200, b
c.token = b["data"]["accessToken"]

s, b = c.req("GET", "/shop/payment-config")
check("未创建店铺时返回 30003", s == 404 and b["code"] == 30003, b)

slug = f"pay-{stamp % 1000000}"
s, b = c.req("POST", "/shop", {"name": "收款测试店", "slug": slug})
assert s == 200, b

s, b = c.req("GET", "/shop/payment-config")
check("未配置收款", s == 200 and b["data"]["configured"] is False, b)

key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
priv = base64.b64encode(key.private_bytes(serialization.Encoding.DER, serialization.PrivateFormat.PKCS8, serialization.NoEncryption())).decode()
pub = base64.b64encode(key.public_key().public_bytes(serialization.Encoding.DER, serialization.PublicFormat.SubjectPublicKeyInfo)).decode()
pem_pub = "-----BEGIN PUBLIC KEY-----\n" + "\n".join(pub[i:i + 64] for i in range(0, len(pub), 64)) + "\n-----END PUBLIC KEY-----"

s, b = c.req("POST", "/shop/payment-config/test", {"env": "SANDBOX", "appId": "123", "privateKey": priv, "publicKey": pub})
check("AppID 格式错误定位到字段", s == 400 and "appId" in b["data"]["fields"], b)
s, b = c.req("POST", "/shop/payment-config/test", {"env": "SANDBOX", "appId": "9021000000000001", "privateKey": priv, "publicKey": priv})
check("公钥栏误填私钥被识别", s == 400 and "publicKey" in b["data"]["fields"], b)
s, b = c.req("POST", "/shop/payment-config/test", {"env": "BETA", "appId": "9021000000000001", "privateKey": priv, "publicKey": pub})
check("环境取值校验", s == 400 and "env" in b["data"]["fields"], b)

s, b = c.req("PUT", "/shop/payment-config", {"env": "SANDBOX", "appId": "9021000000000001", "privateKey": priv, "publicKey": pem_pub})
detail = (b.get("data") or {}).get("detail")
print(f"     支付宝沙箱返回：{b.get('message')}（{detail}）")
check("真实调用支付宝沙箱：不存在的 AppID 测试失败（带 PEM 头尾的公钥被正确识别）", s == 400 and b["code"] == 30004 and detail == "40002 isv.invalid-app-id", b)
s, b = c.req("GET", "/shop/payment-config")
check("测试失败不保存", b["data"]["configured"] is False, b)

s, b = Client().req("PUT", "/shop/payment-config", {})
check("未登录返回 401", s == 401, b)

print(f"\n{sum(results)}/{len(results)} passed")
sys.exit(0 if all(results) else 1)
