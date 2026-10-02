-- 商城化调整（PRD 10 · MKT）：商品平台分类、商城列表索引、中文全文检索
ALTER TABLE products
  ADD COLUMN category VARCHAR(20) NULL COMMENT '平台一级分类（MKT-04），上架时必填' AFTER delivery_type;

-- 商城首页“最新上架”与分类浏览：只查已上架商品，按上架时间倒序
ALTER TABLE products
  ADD KEY idx_status_published (status, published_at),
  ADD KEY idx_status_category_published (status, category, published_at);

-- 商品搜索（MKT-02）：ngram 分词支持中文，默认 2 字切分
ALTER TABLE products
  ADD FULLTEXT KEY ft_name_tagline (name, tagline) WITH PARSER ngram;
