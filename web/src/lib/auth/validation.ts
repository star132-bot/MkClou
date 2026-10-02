import { z } from "zod";

// 与后端 internal/user/validate.go 保持一致（PRD AUTH-01）。前端校验只为体验，最终以后端为准。
export const emailSchema = z
  .string()
  .trim()
  .min(1, "请输入邮箱")
  .max(254, "请输入有效的邮箱地址")
  .email("请输入有效的邮箱地址");

export const passwordSchema = z
  .string()
  .min(8, "密码至少 8 位，需包含字母和数字")
  .max(64, "密码最多 64 位")
  .regex(/\p{L}/u, "密码至少 8 位，需包含字母和数字")
  .regex(/\p{N}/u, "密码至少 8 位，需包含字母和数字");

export type PasswordStrength = 0 | 1 | 2 | 3;

/** 密码强度：0 未输入 / 1 弱 / 2 中 / 3 强。仅用于提示，不作为校验规则。 */
export function passwordStrength(pw: string): PasswordStrength {
  if (!pw) return 0;
  let score = 0;
  if (pw.length >= 8) score++;
  if (pw.length >= 12) score++;
  if (/[a-z]/.test(pw) && /[A-Z]/.test(pw)) score++;
  if (/\d/.test(pw)) score++;
  if (/[^A-Za-z0-9]/.test(pw)) score++;
  if (score <= 2) return 1;
  if (score <= 3) return 2;
  return 3;
}

/** 常见邮箱域名拼写纠错（PRD SF-03）。 */
const typoDomains: Record<string, string> = {
  "qq.con": "qq.com",
  "qq.cm": "qq.com",
  "163.con": "163.com",
  "126.con": "126.com",
  "gmial.com": "gmail.com",
  "gmai.com": "gmail.com",
  "gmail.con": "gmail.com",
  "hotmial.com": "hotmail.com",
  "outlook.con": "outlook.com",
};

export function suggestEmail(email: string): string | null {
  const at = email.lastIndexOf("@");
  if (at < 0) return null;
  const fix = typoDomains[email.slice(at + 1).toLowerCase()];
  return fix ? `${email.slice(0, at + 1)}${fix}` : null;
}

/** 登录后跳转地址只允许站内相对路径，防止开放重定向（PRD SEC-05）。 */
export function safeRedirect(target: string | null, fallback = "/dashboard"): string {
  if (!target || !target.startsWith("/") || target.startsWith("//") || target.startsWith("/\\")) return fallback;
  return target;
}
