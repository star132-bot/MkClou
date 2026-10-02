# 商品模块端到端测试：需先启动 docker compose 依赖、API 与 worker（验证邮件由 worker 发送）。
# 运行：python scripts/e2e/product_e2e.py（在 server 目录下）
# 注意：每次运行注册 1 个商家，注册接口按 IP 每小时限 5 次。
import json, re, struct, sys, time, urllib.error, urllib.parse, urllib.request, uuid, zlib

API = "http://127.0.0.1:18080/api/v1"
MAIL = "http://127.0.0.1:8025/api/v1"
run = str(int(time.time()))[-7:]
email = f"product-e2e{run}@example.com"
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
    """生成纯色 PNG，用于上传测试。"""
    row = b"\x00" + bytes(rgb) * w
    raw = zlib.compress(row * h)
    def chunk(t, d): return struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d))
    return b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0)) + chunk(b"IDAT", raw) + chunk(b"IEND", b"")

def upload(token, data, filename="cover.png", ctype="image/png"):
    boundary = uuid.uuid4().hex
    body = (f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"{filename}\"\r\n"
            f"Content-Type: {ctype}\r\n\r\n").encode() + data + f"\r\n--{boundary}--\r\n".encode()
    return req("POST", "/uploads/images", token=token, raw=body, ctype=f"multipart/form-data; boundary={boundary}")

def verify_email(addr):
    for _ in range(40):
        with urllib.request.urlopen(f"{MAIL}/search?query=to:{addr}") as r:
            msgs = json.loads(r.read())["messages"]
        if msgs:
            with urllib.request.urlopen(f"{MAIL}/message/{msgs[0]['ID']}") as r:
                text = json.loads(r.read())["Text"]
            token = re.search(r"token=([A-Za-z0-9_\-]+)", text).group(1)
            return req("POST", "/auth/email/verify", {"token": token})[0] == 200
        time.sleep(0.5)
    return False

s, b = req("POST", "/auth/register", {"email": email, "password": "Passw0rd!", "agreeTerms": True})
if s != 200: sys.exit(f"注册失败（可能触发了注册限流）：{b}")
tok = b["data"]["accessToken"]

# 1. 开店前
s, b = upload(tok, png(200, 150, (91, 91, 214)))
check("未开店不能上传图片", s == 404 and b["code"] == 30003, b)
s, b = req("POST", "/products", {"name": "测试", "deliveryType": "LINK"}, token=tok)
check("未开店不能创建商品", s == 404 and b["code"] == 30003, b)
slug = f"pe2e-{run}"
s, b = req("POST", "/shop", {"name": "商品测试店", "slug": slug}, token=tok)
check("开店", s == 200, b)

# 2. 图片上传
s, b = upload(tok, png(400, 300, (91, 91, 214)))
check("上传 PNG 封面", s == 200 and b["data"]["width"] == 400 and b["data"]["height"] == 300 and b["data"]["key"].endswith(".jpg"), b)
cover = b["data"]
try:
    with urllib.request.urlopen(cover["url"]) as r:
        ok = r.status == 200 and r.headers.get("Content-Type") == "image/jpeg" and r.read(3) == b"\xff\xd8\xff"
except Exception as e:  # noqa: BLE001
    ok, cover_err = False, e
check("公有桶匿名可读，图片已重新编码为 JPEG", ok)
s, b = upload(tok, b"not an image at all", "x.png")
check("非图片文件被拒绝", s == 400, b)
s, b = upload(tok, png(20, 20, (0, 0, 0)))
check("过小的图片被拒绝", s == 400, b)
s, b = upload(tok, png(300, 300, (4, 120, 87)))
cover2 = b["data"]

# 3. 创建与保存
s, b = req("POST", "/products", {"name": "文件商品", "deliveryType": "FILE"}, token=tok)
check("文件交付暂不支持", s == 400, b)
s, b = req("POST", "/products", {"name": "Figma 设计模板包", "deliveryType": "LINK", "price": 0}, token=tok)
check("创建链接商品草稿", s == 200 and b["data"]["status"] == "DRAFT" and re.match(r"^p_[0-9A-Za-z]{12}$", b["data"]["publicId"]), b)
pid, version = b["data"]["publicId"], b["data"]["version"]

s, b = req("POST", f"/products/{pid}/publish", token=tok)
issues = {i["field"] for i in b.get("data", {}).get("issues", [])}
check("内容不完整时上架返回 40001 与问题列表", s == 422 and b["code"] == 40001 and {"account", "category", "images", "descriptionMd", "deliveryConfig.links"} <= issues, b)

body = {
    "version": version, "name": "Figma 设计模板包", "tagline": "3 套模板，拿来即用", "category": "design",
    "price": 0, "originalPrice": None, "deliveryType": "LINK",
    "descriptionMd": "## 包含什么\n\n- 登录页\n- 仪表盘", "maxPerOrder": 1,
    "detail": {"includes": ["3 套 Figma 模板", " "], "audience": "独立开发者", "faqs": [{"q": "能商用吗", "a": "可以"}], "notice": ""},
    "deliveryConfig": {"links": [{"name": "百度网盘", "url": "https://pan.baidu.com/s/1abc", "code": "ab12"}], "text": "会被清掉", "note": "感谢购买"},
    "images": [{"key": cover["key"], "width": cover["width"], "height": cover["height"]}, {"key": cover2["key"], "width": 300, "height": 300}],
}
s, b = req("PUT", f"/products/{pid}", body, token=tok)
check("保存商品", s == 200 and b["data"]["version"] == version + 1 and len(b["data"]["images"]) == 2, b)
check("空项被清理、无关交付内容被丢弃", b["data"]["detail"]["includes"] == ["3 套 Figma 模板"] and b["data"]["deliveryConfig"]["text"] == "", b)
s, b2 = req("PUT", f"/products/{pid}", body, token=tok)
check("旧版本保存返回 10005 乐观锁冲突", s == 409 and b2["code"] == 10005, b2)
version = b["data"]["version"]

s, b = req("GET", f"/products/{pid}", token=tok)
check("读取编辑详情（JSON 列持久化）", b["data"]["detail"]["faqs"] == [{"q": "能商用吗", "a": "可以"}] and b["data"]["deliveryConfig"]["links"][0]["code"] == "ab12", b)

# 4. 上架：邮箱未验证 → 验证后上架
s, b = req("POST", f"/products/{pid}/publish", token=tok)
check("邮箱未验证不能上架", s == 422 and [i["field"] for i in b["data"]["issues"]] == ["account"], b)
check("从 Mailpit 取验证邮件并完成验证", verify_email(email))
s, b = req("POST", f"/products/{pid}/publish", token=tok)
check("免费商品上架成功", s == 200 and b["data"]["status"] == "ON_SALE" and b["data"]["publishedAt"] and b["data"]["deliveryTypeLocked"], b)

s, b = req("POST", "/products", {"name": "付费商品", "deliveryType": "TEXT", "price": 2990}, token=tok)
paid = b["data"]["publicId"]
s, b = req("POST", f"/products/{paid}/publish", token=tok)
check("付费商品未配置收款不能上架", s == 422 and "payment" in {i["field"] for i in b["data"]["issues"]}, b)

# 5. 买家端
s, b = req("GET", f"/public/products/{pid}")
p = b.get("data", {}).get("product", {})
check("买家查看商品详情", s == 200 and p.get("status") == "ON_SALE" and p.get("purchasable") is True and len(p.get("images", [])) == 2, b)
check("买家端不返回交付内容", "deliveryConfig" not in json.dumps(b) and "pan.baidu.com" not in json.dumps(b), b)
check("详情附带店铺信息", b["data"]["shop"]["slug"] == slug, b)
s, b = req("GET", f"/public/products/{paid}")
check("草稿商品对买家显示为已下架", s == 200 and b["data"]["product"]["status"] == "OFF_SALE" and b["data"]["product"]["descriptionMd"] == "", b)
s, b = req("GET", f"/public/shops/{slug}/products")
check("店铺页只列出已上架商品", s == 200 and [i["publicId"] for i in b["data"]["items"]] == [pid] and b["data"]["nextCursor"] is None, b)
s, b = req("GET", "/public/categories")
check("分类列表", s == 200 and len(b["data"]["items"]) == 6, b)

# 6. 已上架商品的修改约束
body["version"] = version
body["deliveryConfig"]["links"] = []
s, b = req("PUT", f"/products/{pid}", body, token=tok)
check("已上架商品不能删光交付内容", s == 422 and b["code"] == 40001, b)
body["deliveryType"] = "TEXT"
s, b = req("PUT", f"/products/{pid}", body, token=tok)
check("上架过的商品不能改交付类型", s == 409 and b["code"] == 40004, b)

# 7. 列表、下架、删除
s, b = req("GET", "/products?status=ON_SALE", token=tok)
check("按状态筛选商品列表", s == 200 and b["data"]["total"] == 1 and b["data"]["items"][0]["cover"]["url"], b)
s, b = req("GET", "/products?q=" + urllib.parse.quote("付费"), token=tok)
check("按名称搜索商品列表", b["data"]["total"] == 1 and b["data"]["items"][0]["publicId"] == paid, b)
s, b = req("DELETE", f"/products/{pid}", token=tok)
check("上架中的商品不能直接删除", s == 409, b)
s, b = req("POST", f"/products/{pid}/unpublish", token=tok)
check("下架", s == 200 and b["data"]["status"] == "OFF_SALE", b)
s, b = req("GET", f"/public/shops/{slug}/products")
check("下架后店铺页不再展示", b["data"]["items"] == [], b)
s, b = req("DELETE", f"/products/{pid}", token=tok)
check("删除已下架商品", s == 200, b)
s, b = req("GET", f"/public/products/{pid}")
check("删除后买家访问返回 404", s == 404, b)

print(f"\n{sum(results)}/{len(results)} passed")
sys.exit(0 if all(results) else 1)
