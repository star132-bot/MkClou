package user

import (
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"

	"mkclou/server/internal/pkg/errcode"
)

// 常见一次性邮箱域名（PRD AUTH-01）。上线后可迁移到 system_configs 由后台维护。
var disposableDomains = map[string]bool{
	"mailinator.com": true, "10minutemail.com": true, "guerrillamail.com": true,
	"temp-mail.org": true, "yopmail.com": true, "trashmail.com": true,
	"sharklasers.com": true, "getnada.com": true, "maildrop.cc": true,
	"dispostable.com": true, "tempmail.com": true, "throwawaymail.com": true,
}

// NormalizeEmail 去除首尾空格并转为小写（PRD 01 第 5 节：大小写不同视为同一邮箱）。
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func fieldError(field, msg string) *errcode.Error {
	return errcode.InvalidParams.WithMessage(msg).WithData(map[string]any{
		"fields": map[string]string{field: msg},
	})
}

// validateEmail 校验已规范化的邮箱。
func validateEmail(email string) error {
	if email == "" || len(email) > 254 {
		return fieldError("email", "请输入有效的邮箱地址")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return fieldError("email", "请输入有效的邮箱地址")
	}
	at := strings.LastIndexByte(email, '@')
	domain := email[at+1:]
	if !strings.Contains(domain, ".") {
		return fieldError("email", "请输入有效的邮箱地址")
	}
	if disposableDomains[domain] {
		return fieldError("email", "暂不支持临时邮箱")
	}
	return nil
}

// validatePassword：8～64 个字符，至少包含字母和数字，不能与邮箱相同。
func validatePassword(password, email string) error {
	const msg = "密码至少 8 位，需包含字母和数字"
	n := utf8.RuneCountInString(password)
	if n < 8 || n > 64 {
		return fieldError("password", msg)
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fieldError("password", msg)
	}
	if strings.EqualFold(password, email) {
		return fieldError("password", "密码不能与邮箱相同")
	}
	return nil
}

func validateNickname(nickname string) error {
	n := utf8.RuneCountInString(nickname)
	if n < 1 || n > 20 {
		return fieldError("nickname", "昵称为 1～20 个字符")
	}
	return nil
}
