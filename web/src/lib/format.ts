// 展示格式化，规则见 docs/prd/README.md 第 5 节与设计规范第 12 节。

/** 分 → `¥ 29.90`；0 显示为“免费”。 */
export function formatPrice(cents: number): string {
  if (cents === 0) return "免费";
  return `¥ ${(cents / 100).toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

/** 字节 → `12.4 MB` */
export function formatFileSize(bytes: number): string {
  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${unit === 0 ? value : value.toFixed(1)} ${units[unit]}`;
}
