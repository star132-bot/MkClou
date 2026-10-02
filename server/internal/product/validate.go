package product

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"mkclou/server/internal/pkg/errcode"
)

// 字段限制（PRD 03 4.2）
const (
	nameMinLen        = 2
	nameMaxLen        = 60
	taglineMaxLen     = 80
	maxImages         = 6
	maxPrice          = 5_000_000 // ¥50,000.00
	descriptionMaxLen = 5000
	includesMax       = 10
	includeItemMaxLen = 50
	audienceMaxLen    = 200
	faqsMax           = 8
	faqQMaxLen        = 50
	faqAMaxLen        = 300
	noticeMaxLen      = 300
	linksMax          = 5
	linkNameMaxLen    = 20
	linkCodeMaxLen    = 20
	linkURLMaxLen     = 1000
	textMaxLen        = 5000
	noteMaxLen        = 500
	maxPerOrderMax    = 20
)

func fieldError(field, msg string) *errcode.Error {
	return errcode.InvalidParams.WithMessage(msg).WithData(map[string]any{
		"fields": map[string]string{field: msg},
	})
}

func runes(s string) int { return utf8.RuneCountInString(s) }

func validateName(name string) error {
	if n := runes(name); n < nameMinLen || n > nameMaxLen {
		return fieldError("name", fmt.Sprintf("商品名称为 %d～%d 个字符", nameMinLen, nameMaxLen))
	}
	return nil
}

func isHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// validate 校验字段格式（草稿也必须满足），不检查上架所需的完整性。
func (in *UpdateInput) validate() error {
	if err := validateName(in.Name); err != nil {
		return err
	}
	if runes(in.Tagline) > taglineMaxLen {
		return fieldError("tagline", fmt.Sprintf("一句话卖点最多 %d 个字符", taglineMaxLen))
	}
	if in.Category != nil && !validCategory(*in.Category) {
		return fieldError("category", "请选择有效的商品分类")
	}
	if in.Price < 0 || in.Price > maxPrice {
		return fieldError("price", "价格为 0（免费）或 ¥0.01～¥50,000.00")
	}
	if in.OriginalPrice != nil && (*in.OriginalPrice <= in.Price || *in.OriginalPrice > maxPrice) {
		return fieldError("originalPrice", "划线原价必须大于售价")
	}
	if len(in.Images) > maxImages {
		return fieldError("images", fmt.Sprintf("最多上传 %d 张封面图", maxImages))
	}
	if runes(in.DescriptionMD) > descriptionMaxLen {
		return fieldError("descriptionMd", fmt.Sprintf("商品介绍最多 %d 个字符", descriptionMaxLen))
	}
	if err := in.Detail.validate(); err != nil {
		return err
	}
	if in.MaxPerOrder < 1 || in.MaxPerOrder > maxPerOrderMax {
		return fieldError("maxPerOrder", fmt.Sprintf("单次限购为 1～%d", maxPerOrderMax))
	}
	return in.DeliveryConfig.validate()
}

func (d *Detail) validate() error {
	if len(d.Includes) > includesMax {
		return fieldError("detail.includes", fmt.Sprintf("“包含内容”最多 %d 项", includesMax))
	}
	for _, s := range d.Includes {
		if runes(s) > includeItemMaxLen {
			return fieldError("detail.includes", fmt.Sprintf("“包含内容”每项最多 %d 个字符", includeItemMaxLen))
		}
	}
	if runes(d.Audience) > audienceMaxLen {
		return fieldError("detail.audience", fmt.Sprintf("“适合谁”最多 %d 个字符", audienceMaxLen))
	}
	if len(d.FAQs) > faqsMax {
		return fieldError("detail.faqs", fmt.Sprintf("常见问题最多 %d 组", faqsMax))
	}
	for _, f := range d.FAQs {
		if f.Q == "" || f.A == "" {
			return fieldError("detail.faqs", "常见问题的问题和回答都需要填写")
		}
		if runes(f.Q) > faqQMaxLen || runes(f.A) > faqAMaxLen {
			return fieldError("detail.faqs", fmt.Sprintf("问题最多 %d 个字符，回答最多 %d 个字符", faqQMaxLen, faqAMaxLen))
		}
	}
	if runes(d.Notice) > noticeMaxLen {
		return fieldError("detail.notice", fmt.Sprintf("购买须知最多 %d 个字符", noticeMaxLen))
	}
	return nil
}

func (c *DeliveryConfig) validate() error {
	if len(c.Links) > linksMax {
		return fieldError("deliveryConfig.links", fmt.Sprintf("最多添加 %d 个链接", linksMax))
	}
	for _, l := range c.Links {
		if !isHTTPURL(l.URL) || len(l.URL) > linkURLMaxLen {
			return fieldError("deliveryConfig.links", "请输入以 http:// 或 https:// 开头的有效链接")
		}
		if runes(l.Name) > linkNameMaxLen || runes(l.Code) > linkCodeMaxLen {
			return fieldError("deliveryConfig.links", fmt.Sprintf("链接名称与提取码最多 %d 个字符", linkNameMaxLen))
		}
	}
	if runes(c.Text) > textMaxLen {
		return fieldError("deliveryConfig.text", fmt.Sprintf("文本内容最多 %d 个字符", textMaxLen))
	}
	if runes(c.Note) > noteMaxLen {
		return fieldError("deliveryConfig.note", fmt.Sprintf("交付附言最多 %d 个字符", noteMaxLen))
	}
	return nil
}

// normalize 去除首尾空格、丢弃空项，并清掉与交付类型无关的配置。
func (in *UpdateInput) normalize(deliveryType string) {
	in.Name = strings.TrimSpace(in.Name)
	in.Tagline = strings.TrimSpace(in.Tagline)
	in.DescriptionMD = strings.TrimSpace(in.DescriptionMD)
	if in.Category != nil && *in.Category == "" {
		in.Category = nil
	}
	if in.MaxPerOrder == 0 {
		in.MaxPerOrder = 1
	}

	inc := make([]string, 0, len(in.Detail.Includes))
	for _, s := range in.Detail.Includes {
		if s = strings.TrimSpace(s); s != "" {
			inc = append(inc, s)
		}
	}
	in.Detail.Includes = inc
	faqs := make([]FAQ, 0, len(in.Detail.FAQs))
	for _, f := range in.Detail.FAQs {
		f.Q, f.A = strings.TrimSpace(f.Q), strings.TrimSpace(f.A)
		if f.Q != "" || f.A != "" {
			faqs = append(faqs, f)
		}
	}
	in.Detail.FAQs = faqs
	in.Detail.Audience = strings.TrimSpace(in.Detail.Audience)
	in.Detail.Notice = strings.TrimSpace(in.Detail.Notice)

	c := &in.DeliveryConfig
	c.Note = strings.TrimSpace(c.Note)
	links := make([]Link, 0, len(c.Links))
	for _, l := range c.Links {
		l.Name, l.URL, l.Code = strings.TrimSpace(l.Name), strings.TrimSpace(l.URL), strings.TrimSpace(l.Code)
		if l.URL != "" {
			links = append(links, l)
		}
	}
	c.Links = links
	c.Text = strings.TrimSpace(c.Text)
	switch deliveryType {
	case DeliveryLink:
		c.Text, c.CardInstructions = "", ""
	case DeliveryText:
		c.Links, c.CardInstructions = []Link{}, ""
	}
}

// completenessIssues 返回上架前必须满足的条件中未通过的项（PRD-08），不含账号、收款与敏感词。
func completenessIssues(p *Product, imageCount int) []Issue {
	var issues []Issue
	if p.Category == nil {
		issues = append(issues, Issue{"category", "请选择商品分类"})
	}
	if imageCount == 0 {
		issues = append(issues, Issue{"images", "请上传至少 1 张封面图"})
	}
	if strings.TrimSpace(p.Description()) == "" {
		issues = append(issues, Issue{"descriptionMd", "请填写商品介绍"})
	}
	switch p.DeliveryType {
	case DeliveryLink:
		if len(p.DeliveryConfig.Links) == 0 {
			issues = append(issues, Issue{"deliveryConfig.links", "请添加至少 1 个交付链接"})
		}
	case DeliveryText:
		if strings.TrimSpace(p.DeliveryConfig.Text) == "" {
			issues = append(issues, Issue{"deliveryConfig.text", "请填写交付的文本内容"})
		}
	}
	return issues
}

// reviewText 是参与敏感词检查的买家可见文本。
func reviewText(p *Product) string {
	parts := []string{p.Name, p.Tagline, p.Description(), p.Detail.Audience, p.Detail.Notice}
	parts = append(parts, p.Detail.Includes...)
	for _, f := range p.Detail.FAQs {
		parts = append(parts, f.Q, f.A)
	}
	return strings.Join(parts, "\n")
}
