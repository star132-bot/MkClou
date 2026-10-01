// 演示数据：后端公开接口完成前使用，接口就绪后替换为真实请求。
import type { ProductPageData, ProductSummary, PublicProduct, PublicShop } from "./types";

const shop: PublicShop = {
  slug: "starfield",
  name: "StarField 的工具铺",
  description: "独立开发者，分享真实项目里沉淀下来的 AI 编程工作流与工程模板。",
  avatarUrl: null,
  contactEmail: "hello@example.com",
  socialLinks: [{ type: "github", url: "https://github.com/star132-bot" }],
  theme: { color: "#5B5BD6", cardRatio: "4:3", layout: "grid", mode: "light" },
  status: "OPEN",
  pauseNote: null,
  isTestMode: true,
};

const descriptionMd = `
用 AI 写代码，最大的差距不在模型，而在**你怎么告诉它**。

这套实战包整理了我在真实项目中反复使用的提示词和配置。它们解决的不是“让 AI 写一个函数”，而是更常见的场景：**接手陌生代码、重构老模块、补测试、排查线上问题**。

### 为什么做这个

刚开始用 Claude Code 和 Codex 时，我也经历过“AI 改了一堆文件，结果全是错的”。后来发现问题出在三点：

1. 没有给 AI 项目约定，它只能猜
2. 任务描述太模糊，没有验收标准
3. 没有让它先计划、再动手

这套模板就是围绕这三点设计的。

### 使用方式

下载后解压，把 \`AGENTS.md\` 或 \`CLAUDE.md\` 放到项目根目录，按注释修改技术栈信息即可。提示词按场景分类，复制后替换方括号里的内容：

\`\`\`text
请先阅读 [模块路径]，列出你的修改计划，等我确认后再改代码。
验收标准：go test ./... 全部通过，覆盖率不低于 80%。
\`\`\`
`.trim();

const product: PublicProduct = {
  publicId: "p_7Hk2Qa9xLm3c",
  name: "AI 编程实战包：Claude Code & Codex 提示词与配置模板",
  tagline: "120+ 条真实项目验证过的提示词，附 6 套技术栈的项目约定模板",
  price: 4900,
  originalPrice: 6900,
  status: "ON_SALE",
  cover: { url: "/demo/cover-1.svg", width: 1200, height: 900, alt: "终端中运行 AI 编程助手的示意图" },
  images: [
    { url: "/demo/cover-1.svg", width: 1200, height: 900, alt: "终端中运行 AI 编程助手的示意图" },
    { url: "/demo/cover-2.svg", width: 1200, height: 900, alt: "实战包包含的文件列表" },
    { url: "/demo/cover-3.svg", width: 1200, height: 900, alt: "120+ 条提示词覆盖的场景" },
  ],
  descriptionMd,
  detail: {
    includes: [
      "120+ 条按场景分类的提示词",
      "6 套 AGENTS.md / CLAUDE.md 项目约定模板",
      "12 个自动化钩子配置",
      "86 页使用手册（PDF）",
      "后续版本免费更新",
    ],
    audience: "已经在用 AI 编程工具、但效果不稳定的开发者；想把 AI 引入团队工作流的技术负责人。",
    faqs: [
      { q: "适用于哪些工具？", a: "主要针对 Claude Code 和 Codex 编写，大部分提示词同样适用于 Cursor、Copilot 等工具。" },
      { q: "需要什么基础？", a: "会用命令行、了解 Git 即可。模板中的技术栈示例涵盖 Go、Java、Python、TypeScript。" },
      { q: "购买后如何获取？", a: "付款后页面会立即显示下载按钮，同时发送邮件。之后也可以通过“已购查询”随时找回。" },
      { q: "后续更新怎么获取？", a: "更新后会发邮件通知，使用原订单即可下载新版本。" },
    ],
    notice: "虚拟商品，交付后不支持无理由退款。如遇文件无法下载等问题，请通过订单页联系我处理。",
  },
  delivery: { type: "FILE", fileCount: 3, totalSize: 13_002_342 },
  maxPerOrder: 1,
  stockHint: null,
  purchasable: true,
  soldOut: false,
  isNew: true,
};

const moreFromShop: ProductSummary[] = [
  {
    publicId: "p_N2c8Vb4kQe1z",
    name: "独立开发者 Notion 看板模板",
    price: 1900,
    originalPrice: null,
    cover: { url: "/demo/other-1.svg", width: 1200, height: 900, alt: "Notion 看板模板预览" },
    soldOut: false,
    isNew: false,
  },
  {
    publicId: "p_R5t1Hy7uWm0x",
    name: "SaaS 后台 Figma 组件库",
    price: 9900,
    originalPrice: 12900,
    cover: { url: "/demo/other-2.svg", width: 1200, height: 900, alt: "Figma 后台组件预览" },
    soldOut: false,
    isNew: false,
  },
  {
    publicId: "p_K9w3Pz6sJd2a",
    name: "Go 后端项目模板（含支付回调）",
    price: 0,
    originalPrice: null,
    cover: { url: "/demo/other-3.svg", width: 1200, height: 900, alt: "Go 后端项目目录结构" },
    soldOut: true,
    isNew: false,
  },
];

/**
 * 演示场景，通过 URL 参数 ?demo= 切换，用于检查各种购买按钮状态（PRD SF-02）：
 * soldout 已售罄 · paused 店铺暂停 · unavailable 收款失效 · card 卡密商品（数量选择 + 库存提示） · offsale 已下架
 */
export type DemoScenario = "default" | "soldout" | "paused" | "unavailable" | "card" | "offsale";

export async function getProductPage(
  slug: string,
  publicId: string,
  scenario: DemoScenario = "default",
): Promise<ProductPageData | null> {
  if (slug !== shop.slug || publicId !== product.publicId) return null;

  const data: ProductPageData = structuredClone({ shop, product, moreFromShop });
  switch (scenario) {
    case "soldout":
      data.product.soldOut = true;
      break;
    case "paused":
      data.shop.status = "PAUSED";
      data.shop.pauseNote = "国庆假期暂停营业，10 月 8 日恢复。";
      break;
    case "unavailable":
      data.product.purchasable = false;
      break;
    case "card":
      data.product.delivery = { type: "CARD", fileCount: null, totalSize: null };
      data.product.maxPerOrder = 5;
      data.product.stockHint = 3;
      break;
    case "offsale":
      data.product.status = "OFF_SALE";
      break;
  }
  return data;
}
