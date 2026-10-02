package shop

import (
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"mkclou/server/internal/pkg/errcode"
)

// 字段长度限制（PRD 02 第 4 节）
const (
	nameMinLen        = 2
	nameMaxLen        = 20
	slugMinLen        = 3
	slugMaxLen        = 20
	descriptionMaxLen = 200
	pauseNoteMaxLen   = 100
	maxSocialLinks    = 4
	socialURLMaxLen   = 255
)

// PresetColors 是 8 个预设主题色，与白色文字的对比度均 ≥ 4.5:1（PRD SHOP-03）。
var PresetColors = []string{
	"#5B5BD6", "#2563EB", "#0E7490", "#047857", "#B45309", "#C2410C", "#BE123C", "#7E22CE",
}

// minBrandContrast 是主题色与白色文字的最低对比度（设计规范 13）。
const minBrandContrast = 4.5

var (
	slugPattern  = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)
	colorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

var (
	cardRatios = map[string]bool{"4:3": true, "16:9": true, "1:1": true}
	layouts    = map[string]bool{"grid": true, "list": true}
	modes      = map[string]bool{"light": true, "dark": true, "system": true}
	// socialTypes 是支持的社交链接类型，前端按类型显示图标
	socialTypes = map[string]bool{
		"website": true, "github": true, "bilibili": true, "xiaohongshu": true,
		"weibo": true, "douyin": true, "x": true, "youtube": true,
	}
)

func fieldError(field, msg string) *errcode.Error {
	return errcode.InvalidParams.WithMessage(msg).WithData(map[string]any{
		"fields": map[string]string{field: msg},
	})
}

// NormalizeSlug 去除首尾空格并转为小写。
func NormalizeSlug(slug string) string { return strings.ToLower(strings.TrimSpace(slug)) }

func validateName(name string) error {
	n := utf8.RuneCountInString(name)
	if n < nameMinLen || n > nameMaxLen {
		return fieldError("name", fmt.Sprintf("店铺名称为 %d～%d 个字符", nameMinLen, nameMaxLen))
	}
	return nil
}

// slugFormatError 检查链接格式，不涉及数据库。返回面向用户的原因，格式正确时返回空字符串。
func slugFormatError(slug string, reserved map[string]bool) string {
	switch {
	case len(slug) < slugMinLen || len(slug) > slugMaxLen:
		return fmt.Sprintf("链接为 %d～%d 个字符", slugMinLen, slugMaxLen)
	case !slugPattern.MatchString(slug):
		return "只能使用小写字母、数字和连字符，且不能以连字符开头或结尾"
	case strings.Contains(slug, "--"):
		return "不能包含连续的连字符"
	case reserved[slug]:
		return "这是系统保留的链接，请换一个"
	}
	return ""
}

func validateDescription(desc string) error {
	if utf8.RuneCountInString(desc) > descriptionMaxLen {
		return fieldError("description", fmt.Sprintf("店铺简介最多 %d 个字符", descriptionMaxLen))
	}
	return nil
}

func validateContactEmail(email string) error {
	const msg = "请输入有效的联系邮箱"
	if email == "" || len(email) > 254 {
		return fieldError("contactEmail", msg)
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email[strings.LastIndexByte(email, '@')+1:], ".") {
		return fieldError("contactEmail", msg)
	}
	return nil
}

func validateSocialLinks(links []SocialLink) error {
	if len(links) > maxSocialLinks {
		return fieldError("socialLinks", fmt.Sprintf("最多添加 %d 个社交链接", maxSocialLinks))
	}
	for _, l := range links {
		if !socialTypes[l.Type] {
			return fieldError("socialLinks", "不支持的社交链接类型")
		}
		u, err := url.Parse(l.URL)
		if err != nil || len(l.URL) > socialURLMaxLen || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fieldError("socialLinks", "请输入以 http:// 或 https:// 开头的有效链接")
		}
	}
	return nil
}

func validateTheme(t Theme) error {
	if !colorPattern.MatchString(t.Color) {
		return fieldError("theme.color", "主题色格式不正确")
	}
	if ContrastWithWhite(t.Color) < minBrandContrast {
		return fieldError("theme.color", "这个颜色太浅，按钮上的白色文字会看不清，请选择更深的颜色")
	}
	if !cardRatios[t.CardRatio] {
		return fieldError("theme.cardRatio", "不支持的商品卡片比例")
	}
	if !layouts[t.Layout] {
		return fieldError("theme.layout", "不支持的布局模板")
	}
	if !modes[t.Mode] {
		return fieldError("theme.mode", "不支持的明暗模式")
	}
	return nil
}

func validatePauseNote(note string) error {
	if utf8.RuneCountInString(note) > pauseNoteMaxLen {
		return fieldError("pauseNote", fmt.Sprintf("暂停说明最多 %d 个字符", pauseNoteMaxLen))
	}
	return nil
}

// ContrastWithWhite 计算 #RRGGBB 颜色与白色的对比度（WCAG 2.x 相对亮度公式）。
func ContrastWithWhite(hex string) float64 {
	if !colorPattern.MatchString(hex) {
		return 0
	}
	var lum float64
	weights := [3]float64{0.2126, 0.7152, 0.0722}
	for i := range 3 {
		v, err := strconv.ParseUint(hex[1+i*2:3+i*2], 16, 8)
		if err != nil {
			return 0
		}
		c := float64(v) / 255
		if c <= 0.03928 {
			c /= 12.92
		} else {
			c = math.Pow((c+0.055)/1.055, 2.4)
		}
		lum += weights[i] * c
	}
	return 1.05 / (lum + 0.05)
}
