import { formatPrice } from "@/lib/format";
import { cn } from "@/lib/utils";

interface PriceProps {
  price: number;
  originalPrice?: number | null;
  size?: "lg" | "md" | "sm";
  className?: string;
}

const sizes = {
  lg: { price: "text-price", original: "text-body-lg" },
  md: { price: "text-h3", original: "text-body" },
  sm: { price: "text-body", original: "text-caption" },
};

/** 价格展示：等宽数字；有划线价时显示原价（PRD PRD-10）。 */
export function Price({ price, originalPrice, size = "md", className }: PriceProps) {
  const s = sizes[size];
  const showOriginal = originalPrice != null && originalPrice > price;

  return (
    <span className={cn("inline-flex items-baseline gap-2 tabular-nums", className)}>
      <span className={cn(s.price, "font-semibold text-text-primary")}>{formatPrice(price)}</span>
      {showOriginal && (
        <s className={cn(s.original, "text-text-tertiary")}>
          <span className="sr-only">原价</span>
          {formatPrice(originalPrice)}
        </s>
      )}
    </span>
  );
}
