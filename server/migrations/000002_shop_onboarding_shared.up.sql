-- 新手清单第 4 步“分享你的店铺”（PRD SHOP-05）：记录商家首次复制链接或生成海报的时间
ALTER TABLE shops
  ADD COLUMN shared_at DATETIME(3) NULL COMMENT '首次分享店铺的时间，用于新手清单' AFTER slug_changed_at;
