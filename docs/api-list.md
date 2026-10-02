# MkClou 接口清单

- 文档版本：v0.1
- 日期：2026-10-01
- 依据：[PRD 需求文档](./prd/README.md) · [数据库设计](./database-design.md)

> 本清单定义所有接口的路径、方法、鉴权方式和对应需求，作为前后端分工和开发排期的依据。**请求参数和响应字段的详细定义由代码注释通过 swag 自动生成 OpenAPI 文档**，开发时以 Swagger 为准，前端据此生成 TypeScript 类型。

---

## 1. 通用约定

### 1.1 基础
| 项目 | 约定 |
|---|---|
| 基础路径 | `/api/v1` |
| 协议 | HTTPS；请求与响应均为 JSON（`Content-Type: application/json`），文件上传和支付宝回调除外 |
| 命名 | 路径使用小写 + 连字符（`after-sales`）；JSON 字段使用小驼峰（`buyerEmail`） |
| 资源 ID | 只使用对外 ID：`publicId`（商品）、`orderNo`（订单）、`ticketNo`（售后单）；不暴露自增主键 |
| 金额 | 整数，单位**分**（`2990` 表示 ¥29.90），格式化由前端完成 |
| 时间 | ISO 8601 UTC 字符串，如 `2026-10-01T10:04:05.123Z` |
| 空值 | 字段无值时返回 `null`，不省略字段 |

### 1.2 统一响应
```json
{ "code": 0, "message": "ok", "data": { }, "traceId": "b3f1c2…" }
```
- HTTP 状态码表达错误大类（400 / 401 / 403 / 404 / 409 / 429 / 500），`code` 表达具体业务错误。
- 字段校验失败：`code = 10001`，`data.fields = { "email": "邮箱格式不正确" }`。

### 1.3 分页
- 请求：`?page=1&pageSize=20`（`pageSize` 最大 100）
- 响应：`{ "items": [], "total": 135, "page": 1, "pageSize": 20 }`
- 买家端店铺商品列表使用游标分页：`?cursor=…&limit=24`，响应 `{ "items": [], "nextCursor": "…" }`（为空表示没有更多）。

### 1.4 鉴权方式
| 标记 | 方式 | 说明 |
|---|---|---|
| 🌐 公开 | 无 | 受 IP 限流保护 |
| 🔑 商家 | `Authorization: Bearer <AccessToken>` | 从令牌中解析 `merchantId`，服务端查出 `shopId`；**不接受前端传入的店铺 ID** |
| 🍪 刷新 | `mk_rt` Cookie（HttpOnly） | 仅用于 `/auth/refresh`、`/auth/logout` |
| 🎫 订单 | `X-Order-Token: <token>` **或** 买家会话 Cookie `mk_buyer`（邮箱与订单一致） | 买家访问自己的订单；令牌放在请求头中，不放在 API 的 URL 里，避免被日志记录 |
| 🛡 管理员 | `Authorization: Bearer <AdminToken>` | 独立签发，与商家令牌互不通用 |
| ✍ 签名 | 支付宝 RSA2 签名 | 仅支付回调 |

### 1.5 幂等
- 创建订单必须携带请求头 `Idempotency-Key: <UUID>`，10 分钟内重复提交返回同一订单。
- 退款、补发等写操作由服务端状态机保证幂等（重复调用返回当前状态，不重复执行）。

### 1.6 业务错误码
| 范围 | 模块 | 常用错误码 |
|---|---|---|
| `0` | 成功 | — |
| `10000-19999` | 通用 | `10001` 参数校验失败、`10002` 请求过于频繁、`10003` 资源不存在、`10004` 状态冲突、`10005` 版本冲突（乐观锁）、`10500` 服务器内部错误 |
| `20000-29999` | 认证 | `20001` 邮箱或密码错误、`20002` 需要图形验证码、`20003` 账号已锁定、`20004` 邮箱未验证、`20005` 令牌无效或已过期、`20006` 账号已停用、`20007` 邮箱已注册、`20008` 验证码错误 |
| `30000-39999` | 店铺 | `30001` 店铺链接不可用、`30002` 店铺链接 30 天内只能修改一次、`30003` 店铺未创建、`30004` 收款配置测试失败、`30005` 店铺暂停营业、`30006` 收款未配置或已失效 |
| `40000-49999` | 商品 | `40001` 上架检查未通过（`data.issues` 列出未通过项）、`40002` 商品不可购买、`40003` 已有订单不能删除、`40004` 交付类型不能修改、`40005` 内容包含违规词、`40006` 文件上传未完成、`40007` 卡密导入预检已过期 |
| `50000-59999` | 订单 | `50001` 库存不足（`data.available`）、`50002` 价格已变更（`data.price`）、`50003` 订单已关闭、`50004` 不可退款（`data.reason`）、`50005` 待支付订单过多、`50006` 支付服务不可用、`50007` 已有处理中的售后单 |
| `60000-69999` | 交付 | `60001` 下载次数已用完、`60002` 交付内容已撤销（已退款）、`60003` 订单令牌无效 |
| `90000-99999` | 平台管理 | `90001` 需要两步验证、`90002` 两步验证码错误 |

---

## 2. 接口清单

### 2.1 认证与账号（商家）

| # | 方法 | 路径 | 鉴权 | 说明 | 需求 |
|---|---|---|---|---|---|
| 1 | POST | `/auth/register` | 🌐 | 注册，成功后返回登录令牌 | AUTH-01 |
| 2 | POST | `/auth/email/verify` | 🌐 | 提交邮件中的验证令牌 | AUTH-02 |
| 3 | POST | `/auth/email/resend` | 🔑 | 重新发送验证邮件 | AUTH-02 |
| 4 | GET | `/auth/captcha` | 🌐 | 获取图形验证码（图片 + captchaId） | AUTH-08 |
| 5 | POST | `/auth/login` | 🌐 | 登录；返回 Access Token，Refresh Token 写入 Cookie | AUTH-03 |
| 6 | POST | `/auth/refresh` | 🍪 | 轮换 Refresh Token，返回新 Access Token | AUTH-06 |
| 7 | POST | `/auth/logout` | 🍪 | 登出当前会话 | AUTH-04 |
| 8 | POST | `/auth/password/forgot` | 🌐 | 发送重置密码邮件（对不存在的邮箱响应相同） | AUTH-05 |
| 9 | POST | `/auth/password/reset` | 🌐 | 使用令牌设置新密码 | AUTH-05 |
| 10 | GET | `/me` | 🔑 | 当前账号信息（含邮箱验证状态、店铺概要） | AUTH-07 |
| 11 | PATCH | `/me` | 🔑 | 修改昵称 | AUTH-07 |
| 12 | PUT | `/me/password` | 🔑 | 修改密码，其他设备下线 | AUTH-07 |
| 13 | GET | `/me/sessions` | 🔑 | 登录设备列表 | AUTH-07 |
| 14 | DELETE | `/me/sessions/{sessionId}` | 🔑 | 退出指定设备 | AUTH-07 |
| 15 | DELETE | `/me/sessions` | 🔑 | 退出除当前设备外的所有设备 | AUTH-07 |
| 16 | GET | `/me/notification-settings` | 🔑 | 获取邮件通知开关 | DSB-07 |
| 17 | PUT | `/me/notification-settings` | 🔑 | 修改邮件通知开关 | DSB-07 |

### 2.2 店铺与收款（商家）

| # | 方法 | 路径 | 鉴权 | 说明 | 需求 |
|---|---|---|---|---|---|
| 18 | GET | `/shop/slug-availability?slug=` | 🔑 | 检查店铺链接是否可用，不可用时返回 3 个推荐 | SHOP-01 |
| 19 | POST | `/shop` | 🔑 | 创建店铺（一个商家只能创建一个） | SHOP-01 |
| 20 | GET | `/shop` | 🔑 | 获取店铺信息与装修配置 | SHOP-02 |
| 21 | PATCH | `/shop` | 🔑 | 修改基本信息与装修配置 | SHOP-02、SHOP-03 |
| 22 | PUT | `/shop/status` | 🔑 | 暂停营业 / 恢复营业 | SHOP-06 |
| 23 | GET | `/shop/onboarding` | 🔑 | 新手清单完成情况 | SHOP-05 |
| 24 | POST | `/shop/onboarding/shared` | 🔑 | 标记“已分享店铺” | SHOP-05 |
| 25 | GET | `/shop/payment-config` | 🔑 | 收款配置状态（不返回私钥明文） | SHOP-04 |
| 26 | POST | `/shop/payment-config/test` | 🔑 | 用提交的参数测试连接，不保存 | SHOP-04 |
| 27 | PUT | `/shop/payment-config` | 🔑 | 保存收款配置（服务端再次测试通过才保存） | SHOP-04 |

### 2.3 上传（商家）

| # | 方法 | 路径 | 鉴权 | 说明 | 需求 |
|---|---|---|---|---|---|
| 28 | POST | `/uploads/images` | 🔑 | 上传图片（multipart，≤ 5MB）；服务端校验、去除 EXIF、重新编码后存入公有桶，返回 key 和宽高 | PRD-02、SHOP-02 |
| 29 | POST | `/uploads/files` | 🔑 | 初始化分片上传：文件名、大小、类型 → `fileId`、分片大小 | PRD-03 |
| 30 | POST | `/uploads/files/{fileId}/parts` | 🔑 | 批量获取分片的预签名上传 URL（前端直传对象存储） | PRD-03 |
| 31 | GET | `/uploads/files/{fileId}` | 🔑 | 查询上传状态与已完成分片（断点续传） | PRD-03 |
| 32 | POST | `/uploads/files/{fileId}/complete` | 🔑 | 合并分片，校验文件头与大小，状态改为 READY | PRD-03 |
| 33 | DELETE | `/uploads/files/{fileId}` | 🔑 | 取消上传，清理已上传分片 | PRD-03 |

### 2.4 商品与卡密（商家）

| # | 方法 | 路径 | 鉴权 | 说明 | 需求 |
|---|---|---|---|---|---|
| 34 | GET | `/products` | 🔑 | 商品列表：`status`、`q`、分页 | PRD-01 |
| 35 | POST | `/products` | 🔑 | 创建商品草稿（需指定交付类型） | PRD-02 |
| 36 | GET | `/products/{publicId}` | 🔑 | 商品详情（编辑用，含交付配置、文件、图片、库存概况） | PRD-02 |
| 37 | PUT | `/products/{publicId}` | 🔑 | 保存商品；携带 `version` 做乐观锁 | PRD-02 ～ PRD-06 |
| 38 | POST | `/products/{publicId}/publish` | 🔑 | 上架；检查未通过时返回 `40001` 与问题列表，命中敏感词进入待审核 | PRD-08 |
| 39 | POST | `/products/{publicId}/unpublish` | 🔑 | 下架 | PRD-08 |
| 40 | POST | `/products/batch-status` | 🔑 | 批量上架 / 下架，返回每个商品的结果 | PRD-01 |
| 41 | POST | `/products/{publicId}/duplicate` | 🔑 | 复制为新草稿（不含卡密） | PRD-13 |
| 42 | DELETE | `/products/{publicId}` | 🔑 | 删除（有订单时返回 `40003`） | PRD-15 |
| 43 | PUT | `/products/sort-order` | 🔑 | 提交新的商品排序 | PRD-12 |
| 44 | POST | `/products/{publicId}/preview-token` | 🔑 | 生成 1 小时有效的预览令牌 | PRD-09 |
| 45 | GET | `/products/{publicId}/cards` | 🔑 | 卡密列表（脱敏）：`status`、分页 | PRD-07 |
| 46 | GET | `/products/{publicId}/cards/stats` | 🔑 | 库存统计：可售 / 预占 / 已售 / 总计 | PRD-07 |
| 47 | POST | `/products/{publicId}/cards/precheck` | 🔑 | 导入预检（文本或文件）：返回有效、重复、空行、超长统计与 `importToken`（10 分钟有效） | PRD-07 |
| 48 | POST | `/products/{publicId}/cards/import` | 🔑 | 凭 `importToken` 执行导入（事务性） | PRD-07 |
| 49 | POST | `/products/{publicId}/cards/batch-delete` | 🔑 | 批量删除可售卡密 | PRD-07 |
| 50 | POST | `/products/{publicId}/cards/{cardId}/reveal` | 🔑 | 查看完整卡密（记录审计日志） | PRD-07 |
| 51 | POST | `/products/{publicId}/cards/export` | 🔑 | 导出 CSV：`status`（记录审计日志） | PRD-07 |

### 2.5 订单、售后与交付管理（商家）

| # | 方法 | 路径 | 鉴权 | 说明 | 需求 |
|---|---|---|---|---|---|
| 52 | GET | `/orders` | 🔑 | 订单列表：`status`、时间范围、`productId`、`q`（订单号 / 邮箱）、分页；返回各状态数量 | ORD-10 |
| 53 | POST | `/orders/export` | 🔑 | 按筛选条件导出 CSV，最多 10,000 条（记录审计日志） | ORD-10 |
| 54 | GET | `/orders/{orderNo}` | 🔑 | 订单详情：订单信息、交付信息、下载记录、邮件状态、时间线 | ORD-11 |
| 55 | POST | `/orders/{orderNo}/refund` | 🔑 | 全额退款：原因、卡密处理方式、订单号后 4 位确认 | ORD-12 |
| 56 | POST | `/orders/{orderNo}/redeliver` | 🔑 | 补发（交付异常时重新交付；卡密商品可更换卡密） | DLV-05、ORD-15 |
| 57 | POST | `/orders/{orderNo}/resend-email` | 🔑 | 重发交付邮件 | DLV-03 |
| 58 | POST | `/orders/{orderNo}/reset-downloads` | 🔑 | 重置下载次数 | DLV-04 |
| 59 | POST | `/orders/{orderNo}/reset-access` | 🔑 | 重置交付页访问令牌，并向买家发送新链接 | DLV-02 |
| 60 | PATCH | `/orders/{orderNo}/buyer-email` | 🔑 | 修改买家邮箱并重发交付邮件（记录审计日志） | DLV-03 |
| 61 | POST | `/orders/{orderNo}/cards/reveal` | 🔑 | 查看该订单已交付的完整卡密（记录审计日志） | ORD-11 |
| 62 | GET | `/after-sales` | 🔑 | 售后列表：`status`、分页 | ORD-15 |
| 63 | GET | `/after-sales/{ticketNo}` | 🔑 | 售后详情（含订单摘要、买家描述、截图） | ORD-15 |
| 64 | POST | `/after-sales/{ticketNo}/resolve` | 🔑 | 处理售后：`action = RESEND / REFUND / REJECT`，附说明 | ORD-15 |

### 2.6 概览与数据（商家）

| # | 方法 | 路径 | 鉴权 | 说明 | 需求 |
|---|---|---|---|---|---|
| 65 | GET | `/dashboard/overview?range=` | 🔑 | 核心指标与环比 | DSB-01 |
| 66 | GET | `/dashboard/todos` | 🔑 | 待办：交付异常、待处理售后、低库存 | DSB-01 |
| 67 | GET | `/analytics/trend` | 🔑 | 趋势：`metric`（销售额 / 订单 / 访客）、时间范围 | DSB-03 |
| 68 | GET | `/analytics/products` | 🔑 | 商品排行 | DSB-03 |
| 69 | GET | `/analytics/funnel` | 🔑 | 转化漏斗 | DSB-04 |
| 70 | GET | `/analytics/sources` | 🔑 | 来源分布 | DSB-05 |
| 71 | GET | `/search?q=` | 🔑 | 全局搜索：订单号、买家邮箱、商品名 | DSB-06 |

### 2.7 买家端：浏览（公开）

| # | 方法 | 路径 | 鉴权 | 说明 | 需求 |
|---|---|---|---|---|---|
| 72 | GET | `/public/shops/{slug}` | 🌐 | 店铺信息与装修配置；旧链接返回 `redirectTo` | SF-01 |
| 73 | GET | `/public/shops/{slug}/products` | 🌐 | 店铺商品列表（游标分页） | SF-01 |
| 74 | GET | `/public/products/{publicId}` | 🌐 | 商品详情（含可售状态、库存提示）；携带 `?previewToken=` 时可查看未上架商品 | SF-02、PRD-09 |
| 75 | POST | `/public/events` | 🌐 | 批量上报埋点事件（最多 20 条） | SF-06 |
| 76 | POST | `/public/reports` | 🌐 | 举报商品 | ADM-05 |
| 76a | GET | `/public/categories` | 🌐 | 平台一级分类列表 | MKT-04 |

### 2.7a 商城与收藏

| # | 方法 | 路径 | 鉴权 | 说明 | 需求 |
|---|---|---|---|---|---|
| 76b | GET | `/public/market/home` | 🌐 | 首页：热门 8 个（不足用最新补齐）+ 最新上架 12 个 | MKT-01、MKT-03 |
| 76c | GET | `/public/market/search` | 🌐 | 搜索：`q`、`category`、`price`（free / 0-50 / 50-200 / 200+）、`sort`（sales / latest / price_asc / price_desc）、`page`，每页 24 个 | MKT-02 |
| 76d | GET | `/me/favorites` | 🔑 | 我的收藏（分页，失效商品标注 `available=false`） | MKT-06 |
| 76e | GET | `/me/favorites/status?ids=` | 🔑 | 批量查询商品是否已收藏（最多 100 个） | MKT-05 |
| 76f | PUT | `/me/favorites/{publicId}` | 🔑 | 收藏（幂等），返回最新收藏人数 | MKT-05 |
| 76g | DELETE | `/me/favorites/{publicId}` | 🔑 | 取消收藏（幂等，失效商品也可取消） | MKT-05 |

### 2.8 买家端：下单与订单

| # | 方法 | 路径 | 鉴权 | 说明 | 需求 |
|---|---|---|---|---|---|
| 77 | POST | `/checkout/orders` | 🌐 + `Idempotency-Key` | 创建订单：商品、数量、邮箱、预期价格、来源；返回 `orderNo`、订单令牌、支付参数。免费商品直接完成交付 | ORD-01、ORD-02、ORD-03、ORD-07 |
| 78 | POST | `/buyer/orders/{orderNo}/pay` | 🎫 | 继续支付：重新生成支付参数（电脑网站 / 手机网站，由 `channel` 指定） | ORD-08 |
| 79 | POST | `/buyer/orders/{orderNo}/cancel` | 🎫 | 取消未支付订单 | ORD-09 |
| 80 | GET | `/buyer/orders/{orderNo}/status` | 🎫 | 轮询订单状态；仍为待支付时触发一次支付宝主动查询（5 秒内最多一次） | SF-04、ORD-05 |
| 81 | GET | `/buyer/orders/{orderNo}` | 🎫 | 订单交付页数据：订单摘要、交付内容（卡密、链接、文本、文件列表与剩余次数）、店铺联系方式 | DLV-02 |
| 82 | POST | `/buyer/orders/{orderNo}/files/{fileId}/download` | 🎫 | 扣减下载次数，返回 5 分钟有效的下载地址 | DLV-04 |
| 83 | POST | `/buyer/orders/{orderNo}/after-sales` | 🎫 | 提交售后申请（截图先通过 #85 上传） | ORD-14 |
| 84 | GET | `/buyer/orders/{orderNo}/after-sales` | 🎫 | 查看该订单的售后进度 | ORD-14 |
| 85 | POST | `/buyer/uploads/images` | 🎫 | 上传售后截图（≤ 3 张，每张 ≤ 5MB） | ORD-14 |

### 2.9 买家端：已购查询

| # | 方法 | 路径 | 鉴权 | 说明 | 需求 |
|---|---|---|---|---|---|
| 86 | POST | `/buyer/lookup/code` | 🌐 | 发送查单验证码（对无订单的邮箱响应相同） | DLV-06、AUTH-09 |
| 87 | POST | `/buyer/lookup/verify` | 🌐 | 校验验证码，写入 2 小时有效的买家会话 Cookie | DLV-06 |
| 88 | GET | `/buyer/lookup/orders` | 🍪 买家会话 | 该邮箱在全平台的订单（按店铺分组） | DLV-06 |
| 89 | POST | `/buyer/lookup/logout` | 🍪 买家会话 | 退出买家会话 | DLV-06 |

> 买家通过已购查询进入订单后，#78 ～ #85 接口可使用买家会话 Cookie 鉴权（服务端校验会话邮箱与订单邮箱一致），无需订单令牌。

### 2.10 支付回调

| # | 方法 | 路径 | 鉴权 | 说明 | 需求 |
|---|---|---|---|---|---|
| 90 | POST | `/webhooks/alipay` | ✍ | 支付宝异步通知（`application/x-www-form-urlencoded`）；处理成功返回纯文本 `success` | ORD-04 |

**处理顺序**：根据报文中的 `out_trade_no` 查出订单与店铺 → 读取该店铺的支付宝公钥验签 → 核对 `app_id`、金额 → 条件更新订单状态。报文中的订单号在验签前**不可信**，但伪造报文无法通过对应店铺公钥的验签，因此路径中无需携带店铺 ID，避免暴露内部 ID。

### 2.11 平台管理

| # | 方法 | 路径 | 鉴权 | 说明 | 需求 |
|---|---|---|---|---|---|
| 91 | POST | `/admin/auth/login` | 🌐 | 邮箱 + 密码 + TOTP 动态码登录；未绑定 TOTP 时返回 `90001` 与绑定信息 | ADM-01 |
| 92 | POST | `/admin/auth/totp/bind` | 🌐（临时令牌） | 首次登录绑定 TOTP | ADM-01 |
| 93 | POST | `/admin/auth/logout` | 🛡 | 登出 | ADM-01 |
| 94 | GET | `/admin/overview` | 🛡 | 平台指标与待处理事项 | ADM-02 |
| 95 | GET | `/admin/merchants` | 🛡 | 商家列表：状态、搜索、分页 | ADM-03 |
| 96 | GET | `/admin/merchants/{merchantId}` | 🛡 | 商家详情：店铺、商品、订单、举报、操作日志 | ADM-03 |
| 97 | POST | `/admin/merchants/{merchantId}/ban` | 🛡 | 封禁（原因必填） | ADM-03 |
| 98 | POST | `/admin/merchants/{merchantId}/unban` | 🛡 | 解封 | ADM-03 |
| 99 | GET | `/admin/products` | 🛡 | 商品列表：待审核 / 全部 / 平台下架 | ADM-04 |
| 100 | POST | `/admin/products/{publicId}/review` | 🛡 | 审核：`action = APPROVE / REJECT`，拒绝需填原因 | ADM-04 |
| 101 | POST | `/admin/products/{publicId}/takedown` | 🛡 | 平台下架（原因必填） | ADM-04 |
| 102 | POST | `/admin/products/{publicId}/restore` | 🛡 | 恢复被平台下架的商品 | ADM-04 |
| 103 | GET | `/admin/reports` | 🛡 | 举报列表 | ADM-05 |
| 104 | GET | `/admin/reports/{reportId}` | 🛡 | 举报详情 | ADM-05 |
| 105 | POST | `/admin/reports/{reportId}/resolve` | 🛡 | 处理：`CONFIRM`（下架）/ `DISMISS`（驳回） | ADM-05 |
| 106 | GET | `/admin/sensitive-words` | 🛡 | 敏感词列表 | ADM-06 |
| 107 | POST | `/admin/sensitive-words` | 🛡 | 批量新增敏感词 | ADM-06 |
| 108 | DELETE | `/admin/sensitive-words/{wordId}` | 🛡 | 删除敏感词 | ADM-06 |
| 109 | GET | `/admin/configs/{key}` | 🛡 | 读取系统配置 | ADM-08 |
| 110 | PUT | `/admin/configs/{key}` | 🛡 | 修改系统配置 | ADM-08 |
| 111 | GET | `/admin/audit-logs` | 🛡 | 审计日志查询 | SEC-08 |

> 平台管理接口中的 `merchantId`、`reportId` 使用内部 ID：仅管理员可访问，不对公众暴露。

### 2.12 运维（不经过公网网关）

| # | 方法 | 路径 | 说明 |
|---|---|---|---|
| 112 | GET | `/healthz` | 存活检查（进程正常即返回 200） |
| 113 | GET | `/readyz` | 就绪检查（MySQL、Redis 可连接） |
| 114 | GET | `/metrics` | Prometheus 指标（V2） |

---

## 3. 异步任务清单（Asynq）

不是 HTTP 接口，但属于服务端对外行为的一部分，一并列出。

| 任务 | 触发 | 说明 | 需求 |
|---|---|---|---|
| `order:close` | 下单后延时 15 分钟 | 查询支付宝 → 关闭交易 → 关单并释放卡密 | ORD-06 |
| `order:deliver` | 支付成功 | 生成交付快照，失败重试 3 次 | DLV-01 |
| `email:send` | 各业务事件 | 渲染模板并发送，失败重试 3 次 | DLV-08 |
| `refund:query` | 退款结果未知 | 每 5 分钟查询，最长 24 小时 | ORD-12 |
| `stock:alert` | 卡密库存变化 | 低于阈值或售罄时通知商家（同一商品 24 小时内最多提醒一次） | PRD-11 |
| `aftersale:remind` | 售后创建后 48 小时 | 未处理则提醒商家 | ORD-15 |
| `cron:payment-compensate` | 每 2 分钟 | 查询 3～20 分钟前创建、仍待支付的订单 | ORD-05 |
| `cron:order-close-sweep` | 每 5 分钟 | 兜底关闭超过 20 分钟的待支付订单 | ORD-06 |
| `cron:analytics-flush` | 每 5 秒 | 批量写入埋点事件 | SF-06 |
| `cron:analytics-rollup` | 每小时 / 每日凌晨 | 汇总每日统计 | DSB-03 |
| `cron:stock-reconcile` | 每日凌晨 | 校准商品的可售库存、销量冗余计数 | PRD-07 |
| `cron:market-hot` | 每小时（worker 启动时也执行一次） | 重算热门商品榜单 `mk:market:hot` | MKT-03 |
| `cron:cleanup` | 每日凌晨 | 清理过期数据（埋点 90 天、回调报文 180 天、邮件记录 90 天）、未完成的上传、无引用的文件对象 | SEC-10 |

---

## 4. 统计

| 分组 | 接口数 |
|---|---|
| 认证与账号 | 17 |
| 店铺与收款 | 10 |
| 上传 | 6 |
| 商品与卡密 | 18 |
| 订单、售后与交付管理 | 13 |
| 概览与数据 | 7 |
| 买家端：浏览 | 5 |
| 买家端：下单与订单 | 9 |
| 买家端：已购查询 | 4 |
| 支付回调 | 1 |
| 平台管理 | 21 |
| 运维 | 3 |
| **合计** | **114** |
| 异步任务 | 12 |

---

## 变更记录

| 版本 | 日期 | 修改内容 |
|---|---|---|
| v0.1 | 2026-10-01 | 初始版本：114 个接口、12 个异步任务；支付回调路径去掉店铺 ID |
| v0.2 | 2026-10-02 | 商品接口按商城化调整：创建草稿需提供 `price`；商品增加 `category` 字段；新增 #76a 分类列表；本迭代交付类型支持链接与文本（文件、卡密接口 #29～#33、#45～#51 待实现） |
| v0.3 | 2026-10-02 | 新增商城与收藏接口 #76b～#76g、定时任务 `cron:market-hot`；“🔑 商家”鉴权在统一账号后适用于所有登录用户 |
