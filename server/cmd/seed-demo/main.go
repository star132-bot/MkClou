// 本地演示数据：创建若干演示卖家、店铺与已上架商品，让商城首页、搜索、热门有内容可看。
//
// 只用于开发环境（app.env = production 时拒绝执行）。可重复执行：已存在的演示账号、店铺和商品会跳过。
// 演示店铺写入“沙箱”状态的收款配置（不含真实密钥），因此买家页面会显示“测试模式”横幅，与真实店铺区分。
//
// 运行：go run ./cmd/seed-demo（在 server 目录下）
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"mkclou/server/internal/pkg/config"
	"mkclou/server/internal/pkg/database"
	"mkclou/server/internal/pkg/storage"
	"mkclou/server/internal/product"
	"mkclou/server/internal/shop"
	"mkclou/server/internal/user"
)

type demoProduct struct {
	id       string // 固定的对外 ID，便于重复执行时识别
	name     string
	tagline  string
	category string
	price    int
	original int
	includes []string
	desc     string
	palette  [2]color.RGBA
}

type demoShop struct {
	email, slug, name, desc string
	color                   string
	products                []demoProduct
}

func rgb(hex uint32) color.RGBA {
	return color.RGBA{uint8(hex >> 16), uint8(hex >> 8), uint8(hex), 255}
}

var shops = []demoShop{
	{
		email: "demo-pixel@mkclou.local", slug: "pixel-lab", name: "像素实验室", color: "#2563EB",
		desc: "独立设计师，做好用的 UI 素材与图标。",
		products: []demoProduct{
			{"p_demoPixel001", "极简线性图标 800 枚", "SVG + Figma 组件，统一 1.5px 线宽", "design", 4900, 6900,
				[]string{"800 个 SVG 图标", "Figma 组件库", "商用授权"}, "## 包含什么\n\n- 24 个分类，覆盖常见后台与移动端场景\n- 每个图标都有 **线性** 与 **填充** 两种样式", [2]color.RGBA{rgb(0x1E3A8A), rgb(0x60A5FA)}},
			{"p_demoPixel002", "SaaS 后台 UI Kit", "60+ 页面模板，深浅色双主题", "design", 19900, 29900,
				[]string{"60+ 页面", "200+ 组件", "设计规范文档"}, "适合独立开发者快速搭建后台界面。", [2]color.RGBA{rgb(0x312E81), rgb(0xA78BFA)}},
			{"p_demoPixel003", "渐变背景素材包", "120 张 4K 渐变背景，免费下载", "design", 0, 0,
				[]string{"120 张 4K 背景", "PNG + JPG"}, "可用于海报、PPT、网站背景。", [2]color.RGBA{rgb(0xBE185D), rgb(0xFDBA74)}},
		},
	},
	{
		email: "demo-code@mkclou.local", slug: "code-forge", name: "代码锻造坊", color: "#047857",
		desc: "后端工程师，分享能直接用在生产环境的项目模板。",
		products: []demoProduct{
			{"p_demoCode0001", "Go 后端项目模板", "Gin + GORM + Asynq，含鉴权与支付回调", "software", 9900, 0,
				[]string{"完整项目源码", "部署文档", "一年更新"}, "## 你会得到\n\n- 分层清晰的项目结构\n- 登录、限流、异步任务开箱即用", [2]color.RGBA{rgb(0x064E3B), rgb(0x34D399)}},
			{"p_demoCode0002", "Next.js 落地页模板", "Tailwind 编写，Lighthouse 满分", "template", 2900, 4900,
				[]string{"5 套落地页", "响应式布局"}, "复制即用的产品落地页。", [2]color.RGBA{rgb(0x0F172A), rgb(0x22D3EE)}},
			{"p_demoCode0003", "Git 速查手册 PDF", "120 个常用命令，按场景整理", "course", 0, 0,
				[]string{"PDF 电子书", "命令速查表"}, "适合新手入门与日常查阅。", [2]color.RGBA{rgb(0x7C2D12), rgb(0xFB923C)}},
		},
	},
	{
		email: "demo-notion@mkclou.local", slug: "notion-life", name: "效率生活家", color: "#B45309",
		desc: "用 Notion 管理生活与工作，分享我的模板。",
		products: []demoProduct{
			{"p_demoNote0001", "Notion 年度计划模板", "目标、习惯、复盘一体化", "template", 1990, 3990,
				[]string{"年度目标看板", "习惯打卡", "月度复盘"}, "一套模板管理一整年。", [2]color.RGBA{rgb(0x78350F), rgb(0xFCD34D)}},
			{"p_demoNote0002", "读书笔记模板", "支持标签检索与金句收集", "template", 0, 0,
				[]string{"Notion 模板", "使用说明"}, "让读过的书真正留下来。", [2]color.RGBA{rgb(0x365314), rgb(0xBEF264)}},
			{"p_demoNote0003", "AI 提示词手册", "300 条写作与办公提示词", "template", 3900, 0,
				[]string{"300 条提示词", "使用示例"}, "覆盖写作、翻译、总结、编程等场景。", [2]color.RGBA{rgb(0x4C1D95), rgb(0xF0ABFC)}},
		},
	},
	{
		email: "demo-learn@mkclou.local", slug: "deep-learn", name: "深度学习笔记", color: "#7E22CE",
		desc: "把复杂的技术讲清楚。",
		products: []demoProduct{
			{"p_demoLearn001", "从零实现 Transformer", "配套代码与 40 页图解讲义", "course", 12900, 19900,
				[]string{"图解讲义", "PyTorch 源码", "练习题"}, "## 适合谁\n\n有 Python 基础、想真正理解大模型原理的同学。", [2]color.RGBA{rgb(0x1E1B4B), rgb(0xF472B6)}},
			{"p_demoLearn002", "Python 数据分析入门", "10 个真实数据集练习", "course", 4900, 0,
				[]string{"10 个 Notebook", "数据集"}, "边做边学 pandas 与可视化。", [2]color.RGBA{rgb(0x134E4A), rgb(0x5EEAD4)}},
			{"p_demoLearn003", "考研英语高频词表", "2000 词，按词频排序", "course", 0, 0,
				[]string{"Excel 词表", "PDF 打印版"}, "免费领取，祝考试顺利。", [2]color.RGBA{rgb(0x881337), rgb(0xFDA4AF)}},
		},
	},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed-demo:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("configs")
	if err != nil {
		return err
	}
	if cfg.IsProduction() {
		return errors.New("refusing to seed demo data in production")
	}
	db, err := database.NewMySQL(cfg.MySQL, false)
	if err != nil {
		return err
	}
	// “查无记录”是本脚本判断是否需要创建的正常情况，不打印
	db = db.Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	store := storage.New(cfg.S3)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := store.EnsureBuckets(ctx); err != nil {
		return err
	}

	now := time.Now().UTC()
	var created, skipped int
	var allProducts []uint64
	var sellers []uint64
	for si, s := range shops {
		mid, err := ensureMerchant(ctx, db, s.email, now)
		if err != nil {
			return err
		}
		sellers = append(sellers, mid)
		shopID, err := ensureShop(ctx, db, mid, s, now)
		if err != nil {
			return err
		}
		for pi, p := range s.products {
			id, isNew, err := ensureProduct(ctx, db, store, shopID, p, now.Add(-time.Duration(si*3+pi)*time.Hour))
			if err != nil {
				return err
			}
			allProducts = append(allProducts, id)
			if isNew {
				created++
			} else {
				skipped++
			}
		}
	}
	// 演示卖家互相收藏一部分商品，让热门榜单和收藏人数有数据
	for i, pid := range allProducts {
		for j, mid := range sellers {
			if (i+j)%3 == 0 {
				res := db.WithContext(ctx).Exec("INSERT IGNORE INTO favorites (merchant_id, product_id, created_at) VALUES (?, ?, ?)", mid, pid, now)
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected > 0 {
					db.WithContext(ctx).Exec("UPDATE products SET favorite_count = favorite_count + 1 WHERE id = ?", pid)
				}
			}
		}
	}
	fmt.Printf("demo data ready: %d products created, %d already existed. Restart the worker (or wait for the hourly job) to refresh the hot list.\n", created, skipped)
	return nil
}

func ensureMerchant(ctx context.Context, db *gorm.DB, email string, now time.Time) (uint64, error) {
	var m user.Merchant
	err := db.WithContext(ctx).Where("email = ?", email).First(&m).Error
	if err == nil {
		return m.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	// 演示账号使用随机密码，无法登录
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	hash, err := bcrypt.GenerateFromPassword(secret, bcrypt.MinCost)
	if err != nil {
		return 0, err
	}
	m = user.Merchant{Email: email, PasswordHash: string(hash), Status: user.StatusActive, EmailVerifiedAt: &now}
	if err := db.WithContext(ctx).Create(&m).Error; err != nil {
		return 0, err
	}
	return m.ID, nil
}

func ensureShop(ctx context.Context, db *gorm.DB, merchantID uint64, s demoShop, now time.Time) (uint64, error) {
	var sh shop.Shop
	err := db.WithContext(ctx).Where("merchant_id = ?", merchantID).First(&sh).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		theme := shop.DefaultTheme()
		theme.Color = s.color
		sh = shop.Shop{MerchantID: merchantID, Slug: s.slug, Name: s.name, Description: s.desc, ContactEmail: s.email,
			SocialLinks: []shop.SocialLink{}, Theme: theme, Status: shop.StatusOpen}
		if err := db.WithContext(ctx).Create(&sh).Error; err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}
	// 沙箱收款配置（演示用，不含真实密钥）：让付费演示商品可以展示，并在买家页显示“测试模式”
	return sh.ID, db.WithContext(ctx).Exec(`INSERT IGNORE INTO shop_payment_configs
(shop_id, provider, env, app_id, private_key_enc, private_key_tail, platform_pub_key, status, verified_at, created_at, updated_at)
VALUES (?, 'ALIPAY', 'SANDBOX', 'DEMO-NOT-REAL', 'demo', 'DEMO00', 'demo', 'ACTIVE', ?, ?, ?)`, sh.ID, now, now, now).Error
}

func ensureProduct(ctx context.Context, db *gorm.DB, store *storage.Client, shopID uint64, d demoProduct, publishedAt time.Time) (uint64, bool, error) {
	var existing product.Product
	err := db.WithContext(ctx).Unscoped().Where("public_id = ?", d.id).First(&existing).Error
	if err == nil {
		return existing.ID, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, err
	}

	key := fmt.Sprintf("img/demo/%s.jpg", d.id)
	data := cover(d.palette[0], d.palette[1])
	if err := store.PutPublic(ctx, key, bytes.NewReader(data), int64(len(data)), "image/jpeg"); err != nil {
		return 0, false, fmt.Errorf("upload cover: %w", err)
	}

	desc := d.desc
	cat := d.category
	limit := 5
	p := product.Product{
		PublicID: d.id, ShopID: shopID, Name: d.name, Tagline: d.tagline, Price: d.price,
		DeliveryType: product.DeliveryText, Category: &cat, Status: product.StatusOnSale,
		DescriptionMD: &desc, Detail: product.Detail{Includes: d.includes, FAQs: []product.FAQ{}},
		DeliveryConfig: product.DeliveryConfig{Links: []product.Link{}, Text: "这是演示商品，没有实际交付内容。"},
		MaxPerOrder:    1, DownloadLimit: &limit, LowStockThreshold: 5, Version: 1, PublishedAt: &publishedAt,
	}
	if d.original > 0 {
		p.OriginalPrice = &d.original
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		return tx.Create(&product.Image{ProductID: p.ID, ObjectKey: key, Width: 1200, Height: 900, CreatedAt: publishedAt}).Error
	})
	return p.ID, true, err
}

// cover 生成 1200×900 的抽象封面：对角渐变 + 两个半透明圆。
func cover(from, to color.RGBA) []byte {
	const w, h = 1200, 900
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	lerp := func(a, b uint8, t float64) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t) }
	circles := [][3]float64{{w * 0.78, h * 0.28, 260}, {w * 0.22, h * 0.82, 340}}
	for y := range h {
		for x := range w {
			t := (float64(x)/w + float64(y)/h) / 2
			c := color.RGBA{lerp(from.R, to.R, t), lerp(from.G, to.G, t), lerp(from.B, to.B, t), 255}
			for _, ci := range circles {
				if math.Hypot(float64(x)-ci[0], float64(y)-ci[1]) < ci[2] {
					c = color.RGBA{lerp(c.R, 255, 0.12), lerp(c.G, 255, 0.12), lerp(c.B, 255, 0.12), 255}
				}
			}
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	return buf.Bytes()
}
