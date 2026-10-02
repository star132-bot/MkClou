// Package upload 实现文件上传（接口清单 #28 ～ #33）。本迭代实现图片上传 #28；
// 虚拟商品文件的分片上传（#29 ～ #33）在文件交付类型中实现。
package upload

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif" // 注册 GIF 解码器（只取第一帧）
	"image/jpeg"
	"image/png"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	_ "golang.org/x/image/webp" // 注册 WebP 解码器

	"mkclou/server/internal/pkg/errcode"
)

const (
	MaxImageBytes = 5 << 20 // 5MB（PRD-02、SHOP-02）
	// maxImagePixels 限制解码后的像素数，防止“解压炸弹”耗尽内存
	maxImagePixels = 40_000_000
	minImageSide   = 64
	jpegQuality    = 88
	// ownerTTL 是上传后等待被商品或店铺引用的时间，超时未引用的图片由清理任务删除
	ownerTTL = 24 * time.Hour
)

// ImageResult 是上传结果：key 用于保存到商品或店铺，url 用于立即预览。
type ImageResult struct {
	Key    string `json:"key"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// PublicStore 是公有桶的写入接口，由 storage.Client 实现。
type PublicStore interface {
	PutPublic(ctx context.Context, key string, body io.ReadSeeker, size int64, contentType string) error
	PublicURL(key string) string
}

type ImageService struct {
	store PublicStore
	rdb   *redis.Client
	now   func() time.Time
}

func NewImageService(store PublicStore, rdb *redis.Client) *ImageService {
	return &ImageService{store: store, rdb: rdb, now: time.Now}
}

func ownerKey(objectKey string) string { return "mk:upload:img:" + objectKey }

var errBadImage = errcode.InvalidParams.WithMessage("只支持 JPG、PNG、WebP、GIF 格式的图片")

// Upload 校验并重新编码图片后存入公有桶。重新编码会丢弃 EXIF 等元数据（含拍摄位置），见 PRD SEC。
// 有透明通道的图片保存为 PNG，其余保存为 JPEG。
func (s *ImageService) Upload(ctx context.Context, shopID uint64, data []byte) (*ImageResult, error) {
	if len(data) == 0 {
		return nil, errBadImage
	}
	if len(data) > MaxImageBytes {
		return nil, errcode.InvalidParams.WithMessage("图片不能超过 5MB，请压缩后重新上传")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errBadImage
	}
	if cfg.Width*cfg.Height > maxImagePixels {
		return nil, errcode.InvalidParams.WithMessage("图片尺寸过大，请缩小到 4000 万像素以内")
	}
	if cfg.Width < minImageSide || cfg.Height < minImageSide {
		return nil, errcode.InvalidParams.WithMessage(fmt.Sprintf("图片太小，宽高至少 %d 像素", minImageSide))
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errBadImage
	}

	var buf bytes.Buffer
	ext, contentType := "jpg", "image/jpeg"
	if format == "png" || format == "webp" || format == "gif" {
		if hasAlpha(img) {
			ext, contentType = "png", "image/png"
		}
	}
	if ext == "png" {
		err = (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, img)
	} else {
		err = jpeg.Encode(&buf, flatten(img), &jpeg.Options{Quality: jpegQuality})
	}
	if err != nil {
		return nil, fmt.Errorf("encode image: %w", err)
	}

	key := fmt.Sprintf("img/%s/%s.%s", s.now().UTC().Format("200601"), randomName(), ext)
	if err := s.store.PutPublic(ctx, key, bytes.NewReader(buf.Bytes()), int64(buf.Len()), contentType); err != nil {
		return nil, fmt.Errorf("put image: %w", err)
	}
	// 记录图片归属：保存商品或店铺时只接受本店铺上传的图片
	if err := s.rdb.Set(ctx, ownerKey(key), shopID, ownerTTL).Err(); err != nil {
		return nil, err
	}
	b := img.Bounds()
	return &ImageResult{Key: key, URL: s.store.PublicURL(key), Width: b.Dx(), Height: b.Dy()}, nil
}

// OwnedBy 检查图片是否由该店铺在最近 24 小时内上传。
func (s *ImageService) OwnedBy(ctx context.Context, shopID uint64, key string) (bool, error) {
	v, err := s.rdb.Get(ctx, ownerKey(key)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v == strconv.FormatUint(shopID, 10), nil
}

// Claim 图片被引用后删除归属记录，避免被清理任务当作未使用的图片删除。
func (s *ImageService) Claim(ctx context.Context, keys ...string) {
	if len(keys) == 0 {
		return
	}
	rk := make([]string, len(keys))
	for i, k := range keys {
		rk[i] = ownerKey(k)
	}
	_ = s.rdb.Del(ctx, rk...).Err()
}

func hasAlpha(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return !o.Opaque()
	}
	return true
}

// flatten 把带透明通道的图片铺在白底上，再编码为 JPEG。
func flatten(img image.Image) image.Image {
	if !hasAlpha(img) {
		return img
	}
	b := img.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, image.White, image.Point{}, draw.Src)
	draw.Draw(dst, b, img, b.Min, draw.Over)
	return dst
}

var b32 = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)

func randomName() string {
	b := make([]byte, 15)
	_, _ = rand.Read(b)
	return strings.ToLower(b32.EncodeToString(b))
}
