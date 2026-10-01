# 账号与认证模块端到端测试：需先启动 docker compose 依赖、API 与 worker。
# 运行：python scripts/e2e/auth_e2e.py（在 server 目录下）
# 注意：注册接口按 IP 每小时限 5 次，连续运行前需清理 Redis 中 rate:mk:rl:register:* 键。
import json, re, time, urllib.request, urllib.error, http.cookiejar, sys

API = "http://127.0.0.1:8080/api/v1"
MAIL = "http://127.0.0.1:8025/api/v1"
email = f"e2e{int(time.time())}@example.com"
pw, pw2 = "Passw0rd!", "NewPassw0rd"
results = []

def check(name, cond, detail=""):
    results.append(bool(cond))
    print(("PASS " if cond else "FAIL ") + name + (f"  [{detail}]" if detail and not cond else ""))

class Client:
    def __init__(self):
        self.jar = http.cookiejar.CookieJar()
        self.op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar))
        self.token = None
    def req(self, method, path, body=None, token=None):
        h = {"Content-Type": "application/json", "User-Agent": "e2e-script"}
        t = token if token is not None else self.token
        if t: h["Authorization"] = "Bearer " + t
        r = urllib.request.Request(API + path, data=json.dumps(body).encode() if body is not None else None, headers=h, method=method)
        try:
            with self.op.open(r) as resp: return resp.status, json.loads(resp.read()), resp.headers
        except urllib.error.HTTPError as e:
            return e.code, json.loads(e.read() or b"{}"), e.headers
    def rt(self):
        return next((c for c in self.jar if c.name == "mk_rt"), None)

def wait_mail(to, subject_kw, after=0):
    for _ in range(40):
        with urllib.request.urlopen(f"{MAIL}/search?query=to:{to}") as r:
            msgs = [m for m in json.loads(r.read())["messages"] if subject_kw in m["Subject"]]
        if len(msgs) > after:
            with urllib.request.urlopen(f"{MAIL}/message/{msgs[0]['ID']}") as r:
                return json.loads(r.read())
        time.sleep(0.5)
    return None

def token_in(msg):
    return re.search(r"token=([A-Za-z0-9_\-]+)", msg["Text"]).group(1)

a = Client()
# 1. 注册
s, b, h = a.req("POST", "/auth/register", {"email": email.upper(), "password": pw, "agreeTerms": True})
a.token = b.get("data", {}).get("accessToken")
check("注册成功并返回 accessToken", s == 200 and a.token, b)
check("邮箱转为小写", b["data"]["merchant"]["email"] == email)
check("响应体中不含 refreshToken", "refreshToken" not in json.dumps(b))
cookie = h.get("Set-Cookie", "")
check("Refresh Token 写入 HttpOnly Cookie，路径限定 /api/v1/auth", "HttpOnly" in cookie and "Path=/api/v1/auth" in cookie and "SameSite=Lax" in cookie, cookie)

s, b, _ = a.req("POST", "/auth/register", {"email": email, "password": pw, "agreeTerms": True})
check("重复注册返回 409 / 20007", s == 409 and b["code"] == 20007, b)
s, b, _ = a.req("POST", "/auth/register", {"email": "bad", "password": "short", "agreeTerms": True})
check("参数错误返回字段级错误", s == 400 and "email" in b["data"]["fields"], b)

# 2. 账号信息与邮箱验证
s, b, _ = a.req("GET", "/me")
check("GET /me 返回未验证状态", s == 200 and b["data"]["merchant"]["emailVerified"] is False, b)
s, b, _ = a.req("GET", "/me", token="")
check("未带 Token 访问 /me 返回 401", s == 401, b)

msg = wait_mail(email, "验证")
check("Mailpit 收到验证邮件（worker 异步发送）", msg is not None)
s, b, _ = a.req("POST", "/auth/email/verify", {"token": token_in(msg)})
check("验证邮箱成功", s == 200 and b["data"]["alreadyVerified"] is False, b)
s, b, _ = a.req("GET", "/me")
check("验证后 emailVerified = true", b["data"]["merchant"]["emailVerified"] is True)
s, b, _ = a.req("POST", "/auth/email/verify", {"token": token_in(msg)})
check("验证链接只能使用一次", s == 401 and b["code"] == 20005, b)

# 3. 续期与轮换
old_rt = a.rt().value
s, b, _ = a.req("POST", "/auth/refresh")
check("续期成功", s == 200 and b["data"]["accessToken"], b)
check("Refresh Token 已轮换", a.rt().value != old_rt)
a.token = b["data"]["accessToken"]
evil = Client(); evil.jar.set_cookie(http.cookiejar.Cookie(0, "mk_rt", old_rt, None, False, "127.0.0.1", False, False, "/api/v1/auth", True, False, None, False, None, None, {}))
s, b, _ = evil.req("POST", "/auth/refresh")
check("旧 Refresh Token 不能再用（宽限期内拒绝）", s == 401, b)
s, b, _ = a.req("GET", "/me")
check("宽限期内重放不影响正常用户", s == 200)

# 4. 登录保护
c = Client()
for i in range(3):
    s, b, _ = c.req("POST", "/auth/login", {"email": email, "password": "Wrong-pass1"})
check("连续失败 3 次后提示需要验证码", b.get("data", {}).get("captchaRequired") is True, b)
s, b, _ = c.req("POST", "/auth/login", {"email": email, "password": pw})
check("此后即使密码正确也要求验证码", s == 401 and b["code"] == 20002, b)
s, b, _ = c.req("GET", "/auth/captcha")
check("获取图形验证码图片", s == 200 and b["data"]["image"].startswith("data:image/png;base64,"), str(b)[:120])

# 5. 找回密码
s, b, _ = c.req("POST", "/auth/password/forgot", {"email": "nobody-" + email})
check("不存在的邮箱也返回成功（防枚举）", s == 200, b)
s, b, _ = c.req("POST", "/auth/password/forgot", {"email": email})
check("找回密码请求成功", s == 200, b)
s2, b2, h2 = c.req("POST", "/auth/password/forgot", {"email": email})
check("60 秒内重复请求被限流并带 Retry-After", s2 == 429 and h2.get("Retry-After"), b2)
msg = wait_mail(email, "重置")
check("收到重置密码邮件", msg is not None)
s, b, _ = c.req("POST", "/auth/password/reset", {"token": token_in(msg), "password": pw2})
check("重置密码成功", s == 200, b)
s, b, _ = a.req("GET", "/me")
check("重置后原设备 Access Token 立即失效（无需等 15 分钟）", s == 401, b)

# 6. 设备管理与登出
d1, d2 = Client(), Client()
s, b, _ = d1.req("POST", "/auth/login", {"email": email, "password": pw2}); d1.token = b["data"]["accessToken"]
check("新密码登录成功，并解除了验证码要求", s == 200, b)
s, b, _ = d2.req("POST", "/auth/login", {"email": email, "password": pw2, "remember": True}); d2.token = b["data"]["accessToken"]
s, b, _ = d1.req("GET", "/me/sessions")
items = b["data"]["items"]
check("设备列表有 2 个会话且标记了当前设备", len(items) == 2 and sum(i["current"] for i in items) == 1, items)
s, b, _ = d1.req("DELETE", "/me/sessions")
s, b, _ = d2.req("GET", "/me")
check("退出其他设备后，设备 2 立即失效", s == 401)
s, b, _ = d1.req("POST", "/auth/logout")
check("登出成功并清除 Cookie", s == 200 and d1.rt() is None)
s, b, _ = d1.req("GET", "/me")
check("登出后 Access Token 立即失效", s == 401)

# 7. IP 限流：注册每小时 5 次（本脚本已用 3 次）
codes = [Client().req("POST", "/auth/register", {"email": f"rl{i}{email}", "password": pw, "agreeTerms": True})[0] for i in range(4)]
check("同一 IP 注册超过每小时 5 次返回 429", codes[-1] == 429, codes)

print(f"\n{sum(results)}/{len(results)} passed  (email={email})")
sys.exit(0 if all(results) else 1)
