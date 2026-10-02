# 商城（首页、搜索）与收藏端到端测试：需先启动 docker compose 依赖、API 与 worker。
# 运行：python scripts/e2e/market_e2e.py（在 server 目录下）
# 注意：每次运行注册 2 个账号（卖家、买家），注册接口按 IP 每小时限 5 次。
import json, re, struct, sys, time, urllib.error, urllib.parse, urllib.request, uuid, zlib

API = "http://127.0.0.1:18080/api/v1"
MAIL = "http://127.0.0.1:8025/api/v1"
run = str(int(time.time()))[-7:]
results = []

def check(name, cond, detail=""):
    results.append(bool(cond))
    print(("PASS " if cond else "FAIL ") + name + (f"  [{detail}]" if detail and not cond else ""))

def req(method, path, body=None, token=None, raw=None, ctype=None):
    h = {"User-Agent": "e2e-script"}
    data = None
    if raw is not None:
        data, h["Content-Type"] = raw, ctype
    elif body is not None:
        data, h["Content-Type"] = json.dumps(body).encode(), "application/json"
    if token: h["Authorization"] = "Bearer " + token
    r = urllib.request.Request(API + path, data=data, headers=h, method=method)
    try:
        with urllib.request.urlopen(r) as resp: return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read() or b"{}")

def png(w, h, rgb):
    row = b"\x00" + bytes(rgb) * w
    def chunk(t, d): return struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d))
    return b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0)) + chunk(b"IDAT", zlib.compress(row * h)) + chunk(b"IEND", b"")

def upload(token, data):
    boundary = uuid.uuid4().hex
    body = (f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"c.png\"\r\nContent-Type: image/png\r\n\r\n").encode() + data + f"\r\n--{boundary}--\r\n".encode()
    return req("POST", "/uploads/images", token=token, raw=body, ctype=f"multipart/form-data; boundary={boundary}")

def register(prefix):
    email = f"{prefix}{run}@example.com"
    s, b = req("POST", "/auth/register", {"email": email, "password": "Passw0rd!", "agreeTerms": True})
    if s != 200: sys.exit(f"注册失败（可能触发了注册限流）：{b}")
    return email, b["data"]["accessToken"]

def verify(addr):
    for _ in range(40):
        with urllib.request.urlopen(f"{MAIL}/search?query=to:{addr}") as r:
            msgs = json.loads(r.read())["messages"]
        if msgs:
            with urllib.request.urlopen(f"{MAIL}/message/{msgs[0]['ID']}") as r:
                token = re.search(r"token=([A-Za-z0-9_\-]+)", json.loads(r.read())["Text"]).group(1)
            return req("POST", "/auth/email/verify", {"token": token})[0] == 200
        time.sleep(0.5)
    return False

def search(**params):
    return req("GET", "/public/market/search?" + urllib.parse.urlencode(params))

# 准备：卖家开店并上架一个免费商品
seller_email, seller = register("mk-seller")
check("卖家完成邮箱验证", verify(seller_email))
slug = f"mk-{run}"
req("POST", "/shop", {"name": f"市场测试店{run}", "slug": slug}, token=seller)
img = upload(seller, png(400, 300, (14, 116, 144)))[1]["data"]
s, b = req("POST", "/products", {"name": f"UI 图标库 {run}", "deliveryType": "TEXT", "price": 0}, token=seller)
pid = b["data"]["publicId"]
body = {"version": b["data"]["version"], "name": f"UI 图标库 {run}", "tagline": "800 个线性图标，支持 Figma", "category": "design",
        "price": 0, "originalPrice": None, "deliveryType": "TEXT", "descriptionMd": "图标合集", "maxPerOrder": 1,
        "detail": {"includes": [], "audience": "", "faqs": [], "notice": ""},
        "deliveryConfig": {"links": [], "text": "下载地址见此处", "cardInstructions": "", "note": ""},
        "images": [{"key": img["key"], "width": 400, "height": 300}]}
s, b = req("PUT", f"/products/{pid}", body, token=seller)
s, b = req("POST", f"/products/{pid}/publish", token=seller)
check("卖家上架免费商品", s == 200 and b["data"]["status"] == "ON_SALE", b)

# 1. 首页
s, b = req("GET", "/public/market/home")
latest = [c["publicId"] for c in b["data"]["latest"]]
check("首页“最新上架”包含新商品且排在第一", s == 200 and latest and latest[0] == pid, latest[:3])
card = b["data"]["latest"][0]
check("商品卡片包含店铺与封面", card["shop"] == {"slug": slug, "name": f"市场测试店{run}"} and card["cover"]["url"], card)
check("首页热门区不为空（热度不足时用最新补齐）", len(b["data"]["hot"]) > 0, b["data"]["hot"])

# 2. 搜索
s, b = search(q=run)
check("按商品名中的数字搜索", s == 200 and pid in [c["publicId"] for c in b["data"]["items"]], b)
s, b = search(q="线性图标")
check("中文关键词匹配卖点", pid in [c["publicId"] for c in b["data"]["items"]], b)
s, b = search(q="ui 图标")
check("英文不区分大小写，多个词同时匹配", pid in [c["publicId"] for c in b["data"]["items"]], b)
s, b = search(q=f"市场测试店{run}")
check("按店铺名搜索", [c["publicId"] for c in b["data"]["items"]] == [pid], b)
s, b = search(q=run, category="software")
check("分类筛选排除不符合的商品", b["data"]["items"] == [], b)
s, b = search(q=run, price="0-50")
check("价格区间筛选（免费商品不在 0～50 区间）", b["data"]["items"] == [], b)
s, b = search(q=run, price="free", category="design", sort="price_desc")
check("免费 + 分类 + 排序组合", [c["publicId"] for c in b["data"]["items"]] == [pid], b)
s, b = search(q='"+-()*~')
check("只有运算符的关键词不报错", s == 200, b)

# 3. 收藏
buyer_email, buyer = register("mk-buyer")
s, b = req("PUT", f"/me/favorites/{pid}")
check("未登录不能收藏", s == 401, b)
s, b = req("PUT", f"/me/favorites/{pid}", token=buyer)
check("收藏商品，收藏人数 +1", s == 200 and b["data"]["favoriteCount"] == 1, b)
s, b = req("PUT", f"/me/favorites/{pid}", token=buyer)
check("重复收藏不重复计数", b["data"]["favoriteCount"] == 1, b)
s, b = req("GET", f"/me/favorites/status?ids={pid},p_notexist0000", token=buyer)
check("收藏状态查询", b["data"]["favorited"] == [pid], b)
s, b = req("GET", "/me/favorites", token=buyer)
item = b["data"]["items"][0] if b["data"]["items"] else {}
check("我的收藏列表", b["data"]["total"] == 1 and item.get("publicId") == pid and item.get("available") is True, b)
s, b = search(q=run)
check("搜索结果显示收藏人数", b["data"]["items"][0]["favoriteCount"] == 1, b)
s, b = req("PUT", "/me/favorites/p_notexist0000", token=buyer)
check("收藏不存在的商品返回 404", s == 404, b)

# 4. 下架后的表现
req("POST", f"/products/{pid}/unpublish", token=seller)
s, b = search(q=run)
check("下架商品不出现在搜索结果", b["data"]["items"] == [], b)
s, b = req("GET", "/public/market/home")
check("下架商品不出现在首页", pid not in [c["publicId"] for c in b["data"]["latest"] + b["data"]["hot"]], b)
s, b = req("GET", "/me/favorites", token=buyer)
item = b["data"]["items"][0]
check("收藏列表保留下架商品并标注已失效", item["available"] is False and item["reason"] == "已失效", item)
s, b = req("PUT", f"/me/favorites/{pid}", token=seller)
check("不能收藏已下架商品", s == 409, b)
s, b = req("DELETE", f"/me/favorites/{pid}", token=buyer)
check("移除失效收藏", s == 200, b)
s, b = req("GET", "/me/favorites", token=buyer)
check("移除后收藏列表为空", b["data"]["total"] == 0, b)
s, b = req("POST", f"/products/{pid}/publish", token=seller)
s, b = search(q=run)
check("收藏人数随取消收藏减少", b["data"]["items"][0]["favoriteCount"] == 0, b)

print(f"\n{sum(results)}/{len(results)} passed")
sys.exit(0 if all(results) else 1)
