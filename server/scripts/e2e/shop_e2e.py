# 店铺模块端到端测试：需先启动 docker compose 依赖与 API。
# 运行：python scripts/e2e/shop_e2e.py（在 server 目录下）
# 注意：每次运行注册 2 个商家，注册接口按 IP 每小时限 5 次，连续运行前需清理 Redis 中 rate:mk:rl:register:* 键。
import json, time, urllib.request, urllib.error, sys

API = "http://127.0.0.1:18080/api/v1"
run = str(int(time.time()))[-7:]
pw = "Passw0rd!"
results = []

def check(name, cond, detail=""):
    results.append(bool(cond))
    print(("PASS " if cond else "FAIL ") + name + (f"  [{detail}]" if detail and not cond else ""))

def req(method, path, body=None, token=None):
    h = {"Content-Type": "application/json", "User-Agent": "e2e-script"}
    if token: h["Authorization"] = "Bearer " + token
    r = urllib.request.Request(API + path, data=json.dumps(body).encode() if body is not None else None, headers=h, method=method)
    try:
        with urllib.request.urlopen(r) as resp: return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read() or b"{}")

def register(prefix):
    s, b = req("POST", "/auth/register", {"email": f"{prefix}{run}@example.com", "password": pw, "agreeTerms": True})
    if s != 200:
        sys.exit(f"注册失败（可能触发了注册限流）：{b}")
    return b["data"]["accessToken"]

a, b_tok = register("shop-a"), register("shop-b")
slug_a, slug_b = f"e2e-a{run}", f"e2e-b{run}"

# 1. 创建前
s, b = req("GET", "/shop", token=a)
check("未创建店铺时 GET /shop 返回 404 / 30003", s == 404 and b["code"] == 30003, b)
s, b = req("GET", "/me", token=a)
check("未创建店铺时 /me 的 shop 为 null", s == 200 and b["data"]["shop"] is None, b)
s, b = req("GET", f"/shop/slug-availability?slug={slug_a}", token=a)
check("新链接可用", s == 200 and b["data"]["available"] is True, b)
s, b = req("GET", "/shop/slug-availability?slug=dashboard", token=a)
check("保留词不可用且返回 3 个推荐", b["data"]["available"] is False and len(b["data"]["suggestions"]) == 3, b)
s, b = req("GET", "/shop/slug-availability?slug=x", token=None)
check("未登录检查链接返回 401", s == 401, b)

# 2. 创建
s, b = req("POST", "/shop", {"name": "E2E 测试店", "slug": slug_a.upper()}, token=a)
check("创建店铺成功，链接转为小写", s == 200 and b["data"]["slug"] == slug_a, b)
check("默认主题与联系邮箱", b["data"]["theme"] == {"color": "#5B5BD6", "cardRatio": "4:3", "layout": "grid", "mode": "light"}
      and b["data"]["contactEmail"] == f"shop-a{run}@example.com", b)
s, b = req("POST", "/shop", {"name": "第二家", "slug": slug_a + "x"}, token=a)
check("一个商家只能创建一个店铺", s == 409 and b["code"] == 10004, b)
s, b = req("POST", "/shop", {"name": "B 的店", "slug": slug_a}, token=b_tok)
check("链接被占用返回 30001 + 推荐", s == 409 and b["code"] == 30001 and len(b["data"]["suggestions"]) == 3, b)
s, b = req("POST", "/shop", {"name": "B", "slug": "-bad"}, token=b_tok)
check("名称和链接格式错误返回字段错误", s == 400 and b["code"] == 10001, b)
s, b = req("POST", "/shop", {"name": "B 的店", "slug": slug_b}, token=b_tok)
check("第二个商家创建店铺成功", s == 200, b)
s, b = req("GET", "/me", token=a)
check("/me 返回店铺概要", b["data"]["shop"] == {"slug": slug_a, "name": "E2E 测试店", "status": "OPEN"}, b)

# 3. 修改基本信息与装修（JSON 列写入 MySQL 后读回）
patch = {
    "description": "端到端测试店铺",
    "contactEmail": "Help@Example.com",
    "socialLinks": [{"type": "github", "url": "https://github.com/star132-bot"}],
    "theme": {"color": "#047857", "cardRatio": "16:9", "layout": "list", "mode": "dark"},
}
s, b = req("PATCH", "/shop", patch, token=a)
check("修改基本信息与装修", s == 200 and b["data"]["contactEmail"] == "help@example.com", b)
s, b = req("GET", "/shop", token=a)
check("装修配置与社交链接持久化", b["data"]["theme"] == patch["theme"] and b["data"]["socialLinks"] == patch["socialLinks"], b)
s, b = req("PATCH", "/shop", {"theme": {"color": "#FFE066", "cardRatio": "4:3", "layout": "grid", "mode": "light"}}, token=a)
check("对比度不足的主题色被拒绝", s == 400 and "theme.color" in b["data"]["fields"], b)

# 4. 买家端与缓存
s, b = req("GET", f"/public/shops/{slug_a}")
check("买家端可访问店铺", s == 200 and b["data"]["shop"]["name"] == "E2E 测试店" and b["data"]["redirectTo"] is None, b)
check("买家端返回装修配置", b["data"]["shop"]["theme"]["mode"] == "dark" and b["data"]["shop"]["isTestMode"] is False, b)
check("买家端不暴露内部字段", "merchantId" not in json.dumps(b) and "id" not in b["data"]["shop"], b)
req("PATCH", "/shop", {"name": "改名后的店"}, token=a)
s, b = req("GET", f"/public/shops/{slug_a}")
check("修改后缓存失效，买家看到新名称", b["data"]["shop"]["name"] == "改名后的店", b)
s, b = req("GET", "/public/shops/no-such-shop-e2e")
check("不存在的店铺返回 404", s == 404, b)

# 5. 修改链接：旧链接跳转、30 天限制、旧链接保留
new_slug = slug_a + "n"
s, b = req("PATCH", "/shop", {"slug": new_slug}, token=a)
check("修改链接成功", s == 200 and b["data"]["slug"] == new_slug and b["data"]["slugChangeAllowed"] is False and b["data"]["nextSlugChangeAt"], b)
s, b = req("GET", f"/public/shops/{slug_a}")
check("旧链接返回 redirectTo", s == 200 and b["data"]["redirectTo"] == new_slug and b["data"]["shop"] is None, b)
s, b = req("PATCH", "/shop", {"slug": slug_a + "z"}, token=a)
check("30 天内再次修改返回 30002", s == 409 and b["code"] == 30002, b)
s, b = req("GET", f"/shop/slug-availability?slug={slug_a}", token=b_tok)
check("旧链接在保留期内不能被他人使用", b["data"]["available"] is False, b)
s, b = req("PATCH", "/shop", {"slug": slug_a}, token=b_tok)
check("他人修改为该旧链接被拒绝", s == 409 and b["code"] == 30001, b)

# 6. 暂停营业
s, b = req("PUT", "/shop/status", {"status": "PAUSED", "pauseNote": "国庆休息"}, token=a)
check("暂停营业", s == 200 and b["data"]["status"] == "PAUSED" and b["data"]["pauseNote"] == "国庆休息", b)
s, b = req("GET", f"/public/shops/{new_slug}")
check("买家看到暂停状态与说明", b["data"]["shop"]["status"] == "PAUSED" and b["data"]["shop"]["pauseNote"] == "国庆休息", b)
s, b = req("PUT", "/shop/status", {"status": "OPEN"}, token=a)
check("恢复营业并清空说明", b["data"]["status"] == "OPEN" and b["data"]["pauseNote"] is None, b)
s, b = req("PUT", "/shop/status", {"status": "BANNED"}, token=a)
check("商家不能自行设置封禁", s == 400, b)

# 7. 新手清单
s, b = req("GET", "/shop/onboarding", token=a)
check("新手清单初始状态", b["data"] == {"emailVerified": False, "paymentReady": False, "hasProduct": False, "shared": False}, b)
s, _ = req("POST", "/shop/onboarding/shared", token=a)
s, b = req("GET", "/shop/onboarding", token=a)
check("标记已分享", b["data"]["shared"] is True, b)

print(f"\n{sum(results)}/{len(results)} passed")
sys.exit(0 if all(results) else 1)
