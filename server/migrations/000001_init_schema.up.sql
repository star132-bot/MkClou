-- 初始表结构，与 docs/database-design.md v0.1 一致

CREATE TABLE merchants (
  id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  email             VARCHAR(254)    NOT NULL COMMENT '登录邮箱，小写',
  password_hash     VARCHAR(100)    NOT NULL COMMENT 'bcrypt 哈希',
  nickname          VARCHAR(40)     NOT NULL DEFAULT '',
  email_verified_at DATETIME(3)     NULL     COMMENT 'NULL 表示未验证',
  status            VARCHAR(20)     NOT NULL DEFAULT 'ACTIVE' COMMENT 'ACTIVE / BANNED',
  ban_reason        VARCHAR(500)    NULL,
  settings          JSON            NULL     COMMENT '通知开关等个人设置',
  last_login_at     DATETIME(3)     NULL,
  created_at        DATETIME(3)     NOT NULL,
  updated_at        DATETIME(3)     NOT NULL,
  deleted_at        DATETIME(3)     NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商家';

CREATE TABLE admins (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  email           VARCHAR(254)    NOT NULL,
  password_hash   VARCHAR(100)    NOT NULL,
  totp_secret_enc VARBINARY(256)  NULL     COMMENT 'TOTP 密钥，AES-GCM 加密；NULL 表示未绑定',
  status          VARCHAR(20)     NOT NULL DEFAULT 'ACTIVE',
  last_login_at   DATETIME(3)     NULL,
  created_at      DATETIME(3)     NOT NULL,
  updated_at      DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='平台管理员';

CREATE TABLE shops (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id     BIGINT UNSIGNED NOT NULL,
  slug            VARCHAR(20)     NOT NULL COMMENT '店铺链接标识',
  name            VARCHAR(40)     NOT NULL,
  description     VARCHAR(400)    NOT NULL DEFAULT '',
  avatar_key      VARCHAR(255)    NULL     COMMENT '对象存储 key',
  cover_key       VARCHAR(255)    NULL,
  contact_email   VARCHAR(254)    NOT NULL,
  social_links    JSON            NULL     COMMENT '[{"type":"github","url":"…"}]',
  theme           JSON            NOT NULL COMMENT '主题色、卡片比例、布局、明暗模式',
  status          VARCHAR(20)     NOT NULL DEFAULT 'OPEN' COMMENT 'OPEN / PAUSED / BANNED',
  pause_note      VARCHAR(200)    NULL,
  slug_changed_at DATETIME(3)     NULL     COMMENT '用于限制 30 天改一次',
  created_at      DATETIME(3)     NOT NULL,
  updated_at      DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_merchant (merchant_id) COMMENT 'V1.0 一个商家一个店铺',
  UNIQUE KEY uk_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='店铺';

CREATE TABLE shop_slug_redirects (
  old_slug   VARCHAR(20)     NOT NULL,
  shop_id    BIGINT UNSIGNED NOT NULL,
  expires_at DATETIME(3)     NOT NULL COMMENT '修改后 90 天',
  created_at DATETIME(3)     NOT NULL,
  PRIMARY KEY (old_slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='店铺旧链接 301 跳转';

CREATE TABLE shop_payment_configs (
  id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  shop_id           BIGINT UNSIGNED NOT NULL,
  provider          VARCHAR(20)     NOT NULL DEFAULT 'ALIPAY',
  env               VARCHAR(20)     NOT NULL COMMENT 'SANDBOX / PRODUCTION',
  app_id            VARCHAR(32)     NOT NULL,
  private_key_enc   VARBINARY(4096) NOT NULL COMMENT '应用私钥，AES-256-GCM 加密',
  private_key_tail  CHAR(6)         NOT NULL COMMENT '私钥末尾 6 位，用于页面识别',
  platform_pub_key  TEXT            NOT NULL COMMENT '支付宝公钥',
  key_version       SMALLINT        NOT NULL DEFAULT 1 COMMENT '加密主密钥版本，支持密钥轮换',
  status            VARCHAR(20)     NOT NULL COMMENT 'ACTIVE / INVALID',
  verified_at       DATETIME(3)     NULL,
  last_error        VARCHAR(500)    NULL,
  created_at        DATETIME(3)     NOT NULL,
  updated_at        DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_shop_provider (shop_id, provider)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='店铺收款配置';

CREATE TABLE products (
  id                  BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  public_id           CHAR(14)         NOT NULL COMMENT '对外 ID，如 p_7Hk2Qa9xLm3c',
  shop_id             BIGINT UNSIGNED  NOT NULL,
  name                VARCHAR(120)     NOT NULL,
  tagline             VARCHAR(160)     NOT NULL DEFAULT '' COMMENT '一句话卖点',
  price               INT UNSIGNED     NOT NULL COMMENT '分；0 表示免费',
  original_price      INT UNSIGNED     NULL     COMMENT '划线价，分',
  delivery_type       VARCHAR(20)      NOT NULL COMMENT 'FILE / LINK / TEXT / CARD',
  status              VARCHAR(20)      NOT NULL DEFAULT 'DRAFT'
                      COMMENT 'DRAFT / PENDING_REVIEW / ON_SALE / OFF_SALE / BANNED',
  description_md      TEXT             NULL     COMMENT '商品介绍 Markdown',
  detail              JSON             NULL     COMMENT '包含内容、适合谁、常见问题、购买须知',
  delivery_config     JSON             NULL     COMMENT '链接列表、文本内容、卡密说明、交付附言',
  max_per_order       TINYINT UNSIGNED NOT NULL DEFAULT 1  COMMENT '单次限购（卡密）',
  download_limit      TINYINT UNSIGNED NULL     DEFAULT 5  COMMENT '每文件下载次数；NULL 不限',
  low_stock_threshold SMALLINT UNSIGNED NOT NULL DEFAULT 5,
  stock_available     INT UNSIGNED     NOT NULL DEFAULT 0  COMMENT '卡密可售数量（冗余计数）',
  sales_count         INT UNSIGNED     NOT NULL DEFAULT 0  COMMENT '累计销量（冗余计数）',
  sort_order          INT              NOT NULL DEFAULT 0,
  ban_reason          VARCHAR(500)     NULL,
  version             INT UNSIGNED     NOT NULL DEFAULT 1  COMMENT '乐观锁',
  published_at        DATETIME(3)      NULL,
  created_at          DATETIME(3)      NOT NULL,
  updated_at          DATETIME(3)      NOT NULL,
  deleted_at          DATETIME(3)      NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_public_id (public_id),
  KEY idx_shop_status_sort (shop_id, status, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品';

CREATE TABLE product_images (
  id         BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  product_id BIGINT UNSIGNED  NOT NULL,
  object_key VARCHAR(255)     NOT NULL COMMENT '公有桶 key',
  width      SMALLINT UNSIGNED NOT NULL,
  height     SMALLINT UNSIGNED NOT NULL,
  sort_order TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '0 为主图',
  created_at DATETIME(3)      NOT NULL,
  PRIMARY KEY (id),
  KEY idx_product_sort (product_id, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品封面图';

CREATE TABLE file_objects (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  shop_id       BIGINT UNSIGNED NOT NULL COMMENT '上传者店铺',
  bucket        VARCHAR(63)     NOT NULL COMMENT '私有桶',
  object_key    VARCHAR(255)    NOT NULL COMMENT '随机路径，不含原文件名',
  original_name VARCHAR(255)    NOT NULL,
  size          BIGINT UNSIGNED NOT NULL COMMENT '字节',
  mime_type     VARCHAR(127)    NOT NULL,
  sha256        CHAR(64)        NULL     COMMENT '上传完成后计算',
  status        VARCHAR(20)     NOT NULL COMMENT 'UPLOADING / READY / DELETED',
  upload_id     VARCHAR(255)    NULL     COMMENT '分片上传 ID，用于断点续传',
  created_at    DATETIME(3)     NOT NULL,
  updated_at    DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_bucket_key (bucket, object_key),
  KEY idx_shop_status (shop_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='私有文件对象';

CREATE TABLE product_files (
  id             BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  product_id     BIGINT UNSIGNED  NOT NULL,
  file_object_id BIGINT UNSIGNED  NOT NULL,
  display_name   VARCHAR(255)     NOT NULL COMMENT '展示给买家的文件名',
  sort_order     TINYINT UNSIGNED NOT NULL DEFAULT 0,
  created_at     DATETIME(3)      NOT NULL,
  PRIMARY KEY (id),
  KEY idx_product_sort (product_id, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品交付文件';

CREATE TABLE cards (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  shop_id      BIGINT UNSIGNED NOT NULL,
  product_id   BIGINT UNSIGNED NOT NULL,
  content_enc  VARBINARY(1024) NOT NULL COMMENT '卡密内容，AES-256-GCM 加密',
  content_hmac CHAR(64)        NOT NULL COMMENT 'HMAC-SHA256(卡密)，用于去重',
  masked       VARCHAR(32)     NOT NULL COMMENT '脱敏展示，如 ABCD****WXYZ',
  status       VARCHAR(20)     NOT NULL DEFAULT 'AVAILABLE' COMMENT 'AVAILABLE / RESERVED / SOLD',
  order_id     BIGINT UNSIGNED NULL,
  batch_no     VARCHAR(32)     NOT NULL COMMENT '导入批次号',
  reserved_at  DATETIME(3)     NULL,
  sold_at      DATETIME(3)     NULL,
  created_at   DATETIME(3)     NOT NULL,
  updated_at   DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_product_hmac (product_id, content_hmac),
  KEY idx_product_status (product_id, status, id),
  KEY idx_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='卡密';

CREATE TABLE orders (
  id                BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  order_no          CHAR(20)         NOT NULL COMMENT 'MK + yyyyMMdd + 10 位',
  shop_id           BIGINT UNSIGNED  NOT NULL,
  product_id        BIGINT UNSIGNED  NOT NULL,
  buyer_email       VARCHAR(254)     NOT NULL,
  quantity          TINYINT UNSIGNED NOT NULL DEFAULT 1,
  unit_price        INT UNSIGNED     NOT NULL COMMENT '下单时单价，分',
  total_amount      INT UNSIGNED     NOT NULL COMMENT '应付总额，分',
  status            VARCHAR(20)      NOT NULL
                    COMMENT 'PENDING / PAID / DELIVERED / DELIVERY_FAILED / CLOSED / REFUNDING / REFUNDED',
  is_test           TINYINT(1)       NOT NULL DEFAULT 0 COMMENT '沙箱订单',
  product_snapshot  JSON             NOT NULL COMMENT '下单时商品名称、封面、价格、交付类型',
  access_token_hash CHAR(64)         NOT NULL COMMENT '交付页令牌的 SHA-256',
  idempotency_key   CHAR(36)         NOT NULL COMMENT '前端生成的 UUID',
  pay_channel       VARCHAR(20)      NULL     COMMENT 'ALIPAY_PAGE / ALIPAY_WAP / FREE',
  trade_no          VARCHAR(64)      NULL     COMMENT '支付宝交易号',
  paid_amount       INT UNSIGNED     NULL     COMMENT '实际到账金额（回调）',
  expires_at        DATETIME(3)      NOT NULL COMMENT '支付截止时间',
  paid_at           DATETIME(3)      NULL,
  delivered_at      DATETIME(3)      NULL,
  closed_at         DATETIME(3)      NULL,
  refunded_at       DATETIME(3)      NULL,
  close_reason      VARCHAR(20)      NULL     COMMENT 'TIMEOUT / BUYER_CANCEL',
  buyer_ip          VARBINARY(16)    NULL     COMMENT 'INET6_ATON，兼容 IPv4/IPv6',
  source            JSON             NULL     COMMENT '来源渠道、UTM 参数',
  version           INT UNSIGNED     NOT NULL DEFAULT 1,
  created_at        DATETIME(3)      NOT NULL,
  updated_at        DATETIME(3)      NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_order_no (order_no),
  UNIQUE KEY uk_idempotency (idempotency_key),
  UNIQUE KEY uk_trade_no (trade_no),
  KEY idx_shop_created (shop_id, created_at),
  KEY idx_shop_status_created (shop_id, status, created_at),
  KEY idx_buyer_email_created (buyer_email, created_at),
  KEY idx_status_created (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='订单';

CREATE TABLE order_events (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_id    BIGINT UNSIGNED NOT NULL,
  event       VARCHAR(40)     NOT NULL COMMENT 'CREATED / PAID / DELIVERED / EMAIL_SENT / CLOSED / REFUND_REQUESTED …',
  from_status VARCHAR(20)     NULL,
  to_status   VARCHAR(20)     NULL,
  actor_type  VARCHAR(20)     NOT NULL COMMENT 'SYSTEM / BUYER / MERCHANT / ADMIN / ALIPAY',
  actor_id    BIGINT UNSIGNED NULL,
  detail      JSON            NULL,
  created_at  DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  KEY idx_order_created (order_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='订单事件流水';

CREATE TABLE payment_notifications (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  shop_id        BIGINT UNSIGNED NOT NULL,
  provider       VARCHAR(20)     NOT NULL,
  kind           VARCHAR(20)     NOT NULL COMMENT 'NOTIFY / QUERY / REFUND',
  order_no       CHAR(20)        NULL,
  raw_body       MEDIUMTEXT      NOT NULL COMMENT '原始报文',
  verify_result  VARCHAR(20)     NOT NULL COMMENT 'OK / BAD_SIGN / MISMATCH',
  process_result VARCHAR(40)     NOT NULL COMMENT 'PROCESSED / DUPLICATE / IGNORED / ERROR',
  remote_ip      VARBINARY(16)   NULL,
  created_at     DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  KEY idx_order_no (order_no),
  KEY idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='支付回调与查询原始记录，保留 180 天';

CREATE TABLE refunds (
  id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_id          BIGINT UNSIGNED NOT NULL,
  shop_id           BIGINT UNSIGNED NOT NULL,
  out_request_no    VARCHAR(64)     NOT NULL COMMENT '退款请求号，支付宝侧幂等',
  amount            INT UNSIGNED    NOT NULL,
  reason            VARCHAR(200)    NOT NULL,
  card_action       VARCHAR(20)     NULL     COMMENT 'KEEP_SOLD / RETURN_STOCK',
  status            VARCHAR(20)     NOT NULL COMMENT 'PROCESSING / SUCCESS / FAILED',
  fail_reason       VARCHAR(500)    NULL,
  operator_type     VARCHAR(20)     NOT NULL COMMENT 'MERCHANT / SYSTEM',
  operator_id       BIGINT UNSIGNED NULL,
  after_sale_id     BIGINT UNSIGNED NULL COMMENT '由售后单发起时关联',
  created_at        DATETIME(3)     NOT NULL,
  finished_at       DATETIME(3)     NULL,
  updated_at        DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_out_request_no (out_request_no),
  KEY idx_order (order_id),
  KEY idx_status_created (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='退款';

CREATE TABLE deliveries (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_id      BIGINT UNSIGNED NOT NULL,
  delivery_type VARCHAR(20)     NOT NULL,
  snapshot      JSON            NOT NULL COMMENT '链接、文本、卡密说明、交付附言（交付时的内容）',
  revoked_at    DATETIME(3)     NULL     COMMENT '退款后撤销访问',
  created_at    DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='交付快照';

CREATE TABLE delivery_files (
  id             BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  order_id       BIGINT UNSIGNED  NOT NULL,
  file_object_id BIGINT UNSIGNED  NOT NULL,
  display_name   VARCHAR(255)     NOT NULL,
  size           BIGINT UNSIGNED  NOT NULL,
  download_count SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  download_limit SMALLINT UNSIGNED NULL     COMMENT 'NULL 不限',
  sort_order     TINYINT UNSIGNED NOT NULL DEFAULT 0,
  created_at     DATETIME(3)      NOT NULL,
  updated_at     DATETIME(3)      NOT NULL,
  PRIMARY KEY (id),
  KEY idx_order (order_id),
  KEY idx_file_object (file_object_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='订单可下载文件';

CREATE TABLE download_records (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  delivery_file_id BIGINT UNSIGNED NOT NULL,
  order_id         BIGINT UNSIGNED NOT NULL,
  ip               VARBINARY(16)   NULL,
  user_agent       VARCHAR(255)    NULL,
  counted          TINYINT(1)      NOT NULL COMMENT '是否计入次数',
  created_at       DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  KEY idx_order_created (order_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='下载记录';

CREATE TABLE after_sales (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  ticket_no      CHAR(20)        NOT NULL COMMENT 'AS + yyyyMMdd + 10 位',
  order_id       BIGINT UNSIGNED NOT NULL,
  shop_id        BIGINT UNSIGNED NOT NULL,
  type           VARCHAR(30)     NOT NULL COMMENT 'CANNOT_DOWNLOAD / INVALID_CARD / NOT_AS_DESCRIBED / DUPLICATE / OTHER',
  description    VARCHAR(1000)   NOT NULL,
  image_keys     JSON            NULL     COMMENT '截图，最多 3 张',
  buyer_email    VARCHAR(254)    NOT NULL,
  status         VARCHAR(20)     NOT NULL COMMENT 'OPEN / RESENT / REFUNDED / REJECTED / CLOSED',
  merchant_reply VARCHAR(1000)   NULL,
  reminded_at    DATETIME(3)     NULL     COMMENT '48 小时未处理提醒时间',
  handled_at     DATETIME(3)     NULL,
  created_at     DATETIME(3)     NOT NULL,
  updated_at     DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_ticket_no (ticket_no),
  KEY idx_shop_status_created (shop_id, status, created_at),
  KEY idx_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='售后单';

CREATE TABLE reports (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  product_id     BIGINT UNSIGNED NOT NULL,
  shop_id        BIGINT UNSIGNED NOT NULL,
  type           VARCHAR(30)     NOT NULL COMMENT 'ILLEGAL / PIRACY / FRAUD / NOT_AS_DESCRIBED / OTHER',
  description    VARCHAR(1000)   NOT NULL DEFAULT '',
  reporter_email VARCHAR(254)    NULL,
  reporter_ip    VARBINARY(16)   NOT NULL,
  status         VARCHAR(20)     NOT NULL DEFAULT 'PENDING' COMMENT 'PENDING / CONFIRMED / DISMISSED',
  handled_by     BIGINT UNSIGNED NULL     COMMENT 'admins.id',
  result_note    VARCHAR(500)    NULL,
  handled_at     DATETIME(3)     NULL,
  created_at     DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  KEY idx_status_created (status, created_at),
  KEY idx_product_created (product_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品举报';

CREATE TABLE sensitive_words (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  word       VARCHAR(100)    NOT NULL,
  level      VARCHAR(20)     NOT NULL COMMENT 'BLOCK / REVIEW',
  created_by BIGINT UNSIGNED NOT NULL,
  created_at DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_word (word)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='敏感词';

CREATE TABLE system_configs (
  config_key VARCHAR(64)     NOT NULL,
  value      JSON            NOT NULL,
  updated_by BIGINT UNSIGNED NULL,
  updated_at DATETIME(3)     NOT NULL,
  PRIMARY KEY (config_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='系统配置';

CREATE TABLE email_logs (
  id          BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  template    VARCHAR(40)      NOT NULL COMMENT 'DELIVERY / NEW_ORDER / VERIFY_EMAIL …',
  to_email    VARCHAR(254)     NOT NULL,
  subject     VARCHAR(200)     NOT NULL,
  shop_id     BIGINT UNSIGNED  NULL,
  order_id    BIGINT UNSIGNED  NULL,
  status      VARCHAR(20)      NOT NULL COMMENT 'PENDING / SENT / FAILED',
  attempts    TINYINT UNSIGNED NOT NULL DEFAULT 0,
  last_error  VARCHAR(500)     NULL,
  sent_at     DATETIME(3)      NULL,
  created_at  DATETIME(3)      NOT NULL,
  updated_at  DATETIME(3)      NOT NULL,
  PRIMARY KEY (id),
  KEY idx_order (order_id),
  KEY idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='邮件发送记录，保留 90 天';

CREATE TABLE analytics_events (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  event       VARCHAR(30)     NOT NULL COMMENT 'shop_view / product_view / checkout_start / order_create / order_paid',
  shop_id     BIGINT UNSIGNED NOT NULL,
  product_id  BIGINT UNSIGNED NULL,
  visitor_id  CHAR(22)        NOT NULL COMMENT '匿名访客 ID',
  source      VARCHAR(20)     NOT NULL COMMENT 'WECHAT / XHS / BILIBILI / SEARCH / DIRECT / OTHER',
  utm         JSON            NULL,
  created_at  DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  KEY idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='埋点原始事件，保留 90 天';

CREATE TABLE analytics_daily (
  stat_date      DATE            NOT NULL COMMENT '北京时间自然日',
  shop_id        BIGINT UNSIGNED NOT NULL,
  product_id     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '0 表示店铺汇总',
  source         VARCHAR(20)     NOT NULL DEFAULT 'ALL',
  pv             INT UNSIGNED    NOT NULL DEFAULT 0,
  uv             INT UNSIGNED    NOT NULL DEFAULT 0,
  checkout_uv    INT UNSIGNED    NOT NULL DEFAULT 0,
  orders_created INT UNSIGNED    NOT NULL DEFAULT 0,
  orders_paid    INT UNSIGNED    NOT NULL DEFAULT 0,
  revenue        INT UNSIGNED    NOT NULL DEFAULT 0 COMMENT '分',
  refund_amount  INT UNSIGNED    NOT NULL DEFAULT 0 COMMENT '分',
  updated_at     DATETIME(3)     NOT NULL,
  PRIMARY KEY (shop_id, stat_date, product_id, source)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='每日统计汇总';

CREATE TABLE audit_logs (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  actor_type  VARCHAR(20)     NOT NULL COMMENT 'MERCHANT / ADMIN / SYSTEM',
  actor_id    BIGINT UNSIGNED NULL,
  shop_id     BIGINT UNSIGNED NULL,
  action      VARCHAR(50)     NOT NULL COMMENT 'LOGIN / REFUND / EXPORT_CARDS / VIEW_CARD / BAN_SHOP …',
  target_type VARCHAR(30)     NULL,
  target_id   VARCHAR(64)     NULL,
  ip          VARBINARY(16)   NULL,
  detail      JSON            NULL     COMMENT '变更前后摘要，不含敏感明文',
  created_at  DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  KEY idx_shop_created (shop_id, created_at),
  KEY idx_actor_created (actor_type, actor_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='审计日志，保留 1 年';

