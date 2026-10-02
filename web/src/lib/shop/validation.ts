import { z } from "zod";

import type { CardRatio, ShopTheme, SocialType } from "@/lib/storefront/types";

// 与后端 internal/shop/validate.go 保持一致（PRD 02 第 4 节）。前端校验只为体验，最终以后端为准。

export const SLUG_MIN = 3;
export const SLUG_MAX = 20;

export const shopNameSchema = z
  .string()
  .trim()
  .min(2, "店铺名称为 2～20 个字符")
  .max(20, "店铺名称为 2～20 个字符");

/** 检查链接格式（不含占用与保留词，那些由服务端判断）。格式正确时返回 null。 */
export function slugFormatError(slug: string): string | null {
  if (slug.length < SLUG_MIN || slug.length > SLUG_MAX) return `链接为 ${SLUG_MIN}～${SLUG_MAX} 个字符`;
  if (!/^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/.test(slug)) return "只能使用小写字母、数字和连字符，且不能以连字符开头或结尾";
  if (slug.includes("--")) return "不能包含连续的连字符";
  return null;
}

/** 把任意输入整理为合法的链接字符：小写字母、数字、单个连字符。 */
export function sanitizeSlug(input: string): string {
  return input
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+/, "")
    .slice(0, SLUG_MAX);
}

export const themeSchema = z.object({
  color: z.string().regex(/^#[0-9A-Fa-f]{6}$/, "主题色格式不正确"),
  cardRatio: z.enum(["4:3", "16:9", "1:1"]),
  layout: z.enum(["grid", "list"]),
  mode: z.enum(["light", "dark", "system"]),
});

export const shopSettingsSchema = z.object({
  name: shopNameSchema,
  slug: z.string().superRefine((v, ctx) => {
    const err = slugFormatError(v);
    if (err) ctx.addIssue({ code: "custom", message: err });
  }),
  description: z.string().trim().max(200, "店铺简介最多 200 个字符"),
  contactEmail: z.string().trim().min(1, "请输入联系邮箱").email("请输入有效的联系邮箱"),
  socialLinks: z
    .array(
      z.object({
        type: z.string(),
        url: z
          .string()
          .trim()
          .max(255, "链接太长")
          .refine((v) => v === "" || /^https?:\/\/[^\s/]+\.[^\s]+$/i.test(v), "请输入以 http:// 或 https:// 开头的有效链接"),
      }),
    )
    .max(4, "最多添加 4 个社交链接"),
  theme: themeSchema,
});

export type ShopSettingsValues = z.infer<typeof shopSettingsSchema>;

export const pauseNoteSchema = z.string().trim().max(100, "暂停说明最多 100 个字符");

// ---------- 装修 ----------

/** 8 个预设主题色，与白色文字的对比度均 ≥ 4.5:1（与后端 PresetColors 一致）。 */
export const PRESET_COLORS = [
  { value: "#5B5BD6", label: "靛蓝" },
  { value: "#2563EB", label: "湖蓝" },
  { value: "#0E7490", label: "青碧" },
  { value: "#047857", label: "松绿" },
  { value: "#B45309", label: "琥珀" },
  { value: "#C2410C", label: "橙红" },
  { value: "#BE123C", label: "玫红" },
  { value: "#7E22CE", label: "紫罗兰" },
] as const;

export const CARD_RATIOS: { value: CardRatio; label: string }[] = [
  { value: "4:3", label: "4:3" },
  { value: "16:9", label: "16:9" },
  { value: "1:1", label: "1:1" },
];

export const LAYOUTS: { value: ShopTheme["layout"]; label: string; hint: string }[] = [
  { value: "grid", label: "网格", hint: "3 列，适合商品较多" },
  { value: "list", label: "列表", hint: "适合少量商品" },
];

export const MODES: { value: ShopTheme["mode"]; label: string }[] = [
  { value: "light", label: "浅色" },
  { value: "dark", label: "深色" },
  { value: "system", label: "跟随买家系统" },
];

export const SOCIAL_TYPES: { value: SocialType; label: string; placeholder: string }[] = [
  { value: "website", label: "个人网站", placeholder: "https://example.com" },
  { value: "github", label: "GitHub", placeholder: "https://github.com/用户名" },
  { value: "bilibili", label: "B 站", placeholder: "https://space.bilibili.com/…" },
  { value: "xiaohongshu", label: "小红书", placeholder: "https://www.xiaohongshu.com/user/profile/…" },
  { value: "weibo", label: "微博", placeholder: "https://weibo.com/…" },
  { value: "douyin", label: "抖音", placeholder: "https://www.douyin.com/user/…" },
  { value: "x", label: "X", placeholder: "https://x.com/…" },
  { value: "youtube", label: "YouTube", placeholder: "https://www.youtube.com/@…" },
];

export const MIN_CONTRAST = 4.5;

function channels(hex: string): [number, number, number] | null {
  const m = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(hex);
  return m ? [parseInt(m[1], 16), parseInt(m[2], 16), parseInt(m[3], 16)] : null;
}

/** 颜色与白色文字的对比度（WCAG 相对亮度公式，与后端 ContrastWithWhite 一致）。 */
export function contrastWithWhite(hex: string): number {
  const c = channels(hex);
  if (!c) return 0;
  const [r, g, b] = c.map((v) => {
    const s = v / 255;
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
  });
  return 1.05 / (0.2126 * r + 0.7152 * g + 0.0722 * b + 0.05);
}

/**
 * 自定义主题色对比度不足时自动加深，直到满足 4.5:1（PRD SHOP-03“提示并自动调整”）。
 * 按比例同时降低三个通道，保持色相不变。
 */
export function ensureContrast(hex: string): string {
  const c = channels(hex);
  if (!c) return hex;
  let factor = 1;
  let out = hex.toUpperCase();
  while (contrastWithWhite(out) < MIN_CONTRAST && factor > 0) {
    factor -= 0.02;
    out = `#${c.map((v) => Math.round(v * factor).toString(16).padStart(2, "0")).join("")}`.toUpperCase();
  }
  return out;
}
