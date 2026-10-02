-- 收藏（PRD MKT-05、MKT-06）。merchants 表即统一账号表（MKT-08），收藏者用 merchant_id 表示
CREATE TABLE favorites (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL COMMENT '收藏者账号',
  product_id  BIGINT UNSIGNED NOT NULL,
  created_at  DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_merchant_product (merchant_id, product_id),
  KEY idx_merchant_created (merchant_id, created_at),
  KEY idx_product_created (product_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品收藏';

-- 收藏人数冗余计数：与收藏记录在同一事务中增减，商品卡片直接读取
ALTER TABLE products
  ADD COLUMN favorite_count INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '收藏人数（冗余计数）' AFTER sales_count;
