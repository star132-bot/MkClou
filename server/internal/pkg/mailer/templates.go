package mailer

import (
	"bytes"
	"fmt"
	"html/template"
)

// 邮件 HTML 布局：单列、内联样式（邮件客户端不支持外部样式表），颜色遵循设计规范。
const layout = `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"></head>
<body style="margin:0;padding:32px 16px;background:#fafafa;font-family:-apple-system,'PingFang SC','Microsoft YaHei',sans-serif;color:#52525b;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center">
<table role="presentation" width="100%" style="max-width:480px;background:#ffffff;border:1px solid #e4e4e7;border-radius:12px;" cellpadding="0" cellspacing="0">
<tr><td style="padding:32px;">
<p style="margin:0 0 24px;font-size:18px;font-weight:600;color:#09090b;">MkClou</p>
<h1 style="margin:0 0 16px;font-size:20px;line-height:28px;font-weight:600;color:#09090b;">{{.Title}}</h1>
{{range .Paragraphs}}<p style="margin:0 0 16px;font-size:14px;line-height:22px;">{{.}}</p>{{end}}
{{if .ActionURL}}<p style="margin:24px 0;"><a href="{{.ActionURL}}" style="display:inline-block;padding:10px 20px;background:#09090b;color:#fafafa;border-radius:8px;text-decoration:none;font-size:14px;font-weight:500;">{{.ActionText}}</a></p>
<p style="margin:0 0 16px;font-size:12px;line-height:18px;color:#71717a;">如果按钮无法点击，请复制以下链接到浏览器打开：<br><span style="word-break:break-all;">{{.ActionURL}}</span></p>{{end}}
{{if .Footnote}}<p style="margin:24px 0 0;padding-top:16px;border-top:1px solid #e4e4e7;font-size:12px;line-height:18px;color:#71717a;">{{.Footnote}}</p>{{end}}
</td></tr></table>
</td></tr></table>
</body></html>`

var layoutTmpl = template.Must(template.New("layout").Parse(layout))

// content 是一封通知邮件的结构化内容，HTML 与纯文本版本由同一份内容生成。
type content struct {
	Title      string
	Paragraphs []string
	ActionText string
	ActionURL  string
	Footnote   string
}

func render(name, to, subject string, c content) (Message, error) {
	var buf bytes.Buffer
	if err := layoutTmpl.Execute(&buf, c); err != nil {
		return Message{}, err
	}

	var text bytes.Buffer
	fmt.Fprintf(&text, "%s\n\n", c.Title)
	for _, p := range c.Paragraphs {
		fmt.Fprintf(&text, "%s\n\n", p)
	}
	if c.ActionURL != "" {
		fmt.Fprintf(&text, "%s：%s\n\n", c.ActionText, c.ActionURL)
	}
	if c.Footnote != "" {
		fmt.Fprintf(&text, "%s\n", c.Footnote)
	}

	return Message{To: to, Subject: subject, HTML: buf.String(), Text: text.String(), Template: name}, nil
}

// VerifyEmail 注册后的邮箱验证邮件（AUTH-02）。
func VerifyEmail(to, link string) (Message, error) {
	return render("VERIFY_EMAIL", to, "验证你的 MkClou 邮箱", content{
		Title:      "验证你的邮箱",
		Paragraphs: []string{"感谢注册 MkClou。点击下方按钮完成邮箱验证，验证后即可上架商品。"},
		ActionText: "验证邮箱",
		ActionURL:  link,
		Footnote:   "链接 24 小时内有效。如果这不是你本人的操作，请忽略这封邮件。",
	})
}

// ResetPassword 找回密码邮件（AUTH-05）。
func ResetPassword(to, link string) (Message, error) {
	return render("RESET_PASSWORD", to, "重置你的 MkClou 密码", content{
		Title:      "重置密码",
		Paragraphs: []string{"我们收到了重置你账号密码的请求。点击下方按钮设置新密码。"},
		ActionText: "设置新密码",
		ActionURL:  link,
		Footnote:   "链接 30 分钟内有效，且只能使用一次。如果这不是你本人的操作，请忽略这封邮件，你的密码不会改变。",
	})
}

// AccountLocked 登录失败次数过多被锁定时的安全提醒（AUTH-08）。
func AccountLocked(to, resetLink string) (Message, error) {
	return render("ACCOUNT_LOCKED", to, "MkClou 安全提醒：登录尝试过多", content{
		Title: "你的账号已被临时锁定",
		Paragraphs: []string{
			"你的账号在短时间内连续登录失败多次，为保护账号安全，已暂停登录 15 分钟。",
			"如果是你本人忘记了密码，可以直接重置密码，重置后锁定会自动解除。",
		},
		ActionText: "重置密码",
		ActionURL:  resetLink,
		Footnote:   "如果这不是你本人的操作，说明有人在尝试登录你的账号，建议尽快修改为更复杂的密码。",
	})
}
