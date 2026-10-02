"use client";

import { Check } from "lucide-react";
import { RadioGroup } from "radix-ui";
import { useId, useState } from "react";

import { Label } from "@/components/ui/label";
import { Segmented } from "@/components/ui/segmented";
import { CARD_RATIOS, ensureContrast, LAYOUTS, MODES, PRESET_COLORS } from "@/lib/shop/validation";
import type { ShopTheme } from "@/lib/storefront/types";

const CUSTOM = "custom";

/** 店铺装修配置（PRD SHOP-03）：主题色、商品卡片比例、布局模板、明暗模式。 */
export function ThemeEditor({ value, onChange }: { value: ShopTheme; onChange: (t: ShopTheme) => void }) {
  const ids = useId();
  const [customMode, setCustomMode] = useState(false);
  const isCustom = customMode || !PRESET_COLORS.some((c) => c.value === value.color);
  const [hex, setHex] = useState(value.color);
  const [adjusted, setAdjusted] = useState<string | null>(null);
  const set = (patch: Partial<ShopTheme>) => onChange({ ...value, ...patch });

  // 自定义颜色：对比度不足时自动加深并提示（PRD SHOP-03）
  const applyCustom = (raw: string) => {
    setHex(raw);
    if (!/^#[0-9a-f]{6}$/i.test(raw)) return;
    const safe = ensureContrast(raw);
    setAdjusted(safe !== raw.toUpperCase() ? raw.toUpperCase() : null);
    set({ color: safe });
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-3">
        <Label id={`${ids}-color`}>主题色</Label>
        <RadioGroup.Root
          aria-labelledby={`${ids}-color`}
          value={isCustom ? CUSTOM : value.color}
          onValueChange={(v) => {
            if (v === CUSTOM) {
              setCustomMode(true);
              setHex(value.color);
              return;
            }
            setCustomMode(false);
            setAdjusted(null);
            set({ color: v });
          }}
          orientation="horizontal"
          className="flex flex-wrap items-center gap-2"
        >
          {PRESET_COLORS.map((c) => (
            <RadioGroup.Item
              key={c.value}
              value={c.value}
              aria-label={c.label}
              title={c.label}
              style={{ backgroundColor: c.value }}
              className="flex size-8 items-center justify-center rounded-full ring-offset-2 ring-offset-background transition-transform duration-150 outline-none hover:scale-105 focus-visible:ring-2 focus-visible:ring-ring data-[state=checked]:ring-2 data-[state=checked]:ring-text-primary"
            >
              <RadioGroup.Indicator>
                <Check className="size-4 text-white" strokeWidth={2} />
              </RadioGroup.Indicator>
            </RadioGroup.Item>
          ))}
          <RadioGroup.Item
            value={CUSTOM}
            className="h-8 rounded-full border border-border px-3 text-body font-medium text-text-secondary transition-colors outline-none hover:text-text-primary focus-visible:ring-2 focus-visible:ring-ring data-[state=checked]:border-text-primary data-[state=checked]:text-text-primary"
          >
            自定义
          </RadioGroup.Item>
        </RadioGroup.Root>

        {isCustom && (
          <div className="flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <input
                type="color"
                value={value.color.toLowerCase()}
                onChange={(e) => applyCustom(e.target.value)}
                aria-label="选择颜色"
                className="size-9 shrink-0 cursor-pointer rounded-md border border-input bg-background p-1"
              />
              <input
                value={hex}
                onChange={(e) => applyCustom(e.target.value.trim())}
                maxLength={7}
                spellCheck={false}
                aria-label="颜色值"
                className="h-9 w-28 rounded-md border border-input bg-background px-3 font-mono text-body text-text-primary uppercase outline-none focus-visible:border-brand focus-visible:ring-3 focus-visible:ring-brand/15"
              />
            </div>
            {adjusted ? (
              <p className="text-caption text-warning" role="status">
                {adjusted} 太浅，按钮上的白色文字会看不清，已自动加深为 {value.color}
              </p>
            ) : (
              <p className="text-caption text-text-tertiary">输入 6 位十六进制颜色，如 #5B5BD6</p>
            )}
          </div>
        )}
      </div>

      <div className="flex flex-col gap-3">
        <Label id={`${ids}-ratio`}>商品卡片比例</Label>
        <Segmented
          aria-labelledby={`${ids}-ratio`}
          value={value.cardRatio}
          onValueChange={(cardRatio) => set({ cardRatio })}
          options={CARD_RATIOS}
          className="self-start"
        />
      </div>

      <div className="flex flex-col gap-3">
        <Label id={`${ids}-layout`}>布局模板</Label>
        <RadioGroup.Root
          aria-labelledby={`${ids}-layout`}
          value={value.layout}
          onValueChange={(v) => set({ layout: v as ShopTheme["layout"] })}
          className="grid grid-cols-2 gap-3"
        >
          {LAYOUTS.map((l) => (
            <RadioGroup.Item
              key={l.value}
              value={l.value}
              aria-label={`${l.label}：${l.hint}`}
              className="flex flex-col items-start gap-2 rounded-lg border border-border p-3 text-left transition-colors outline-none hover:border-text-tertiary focus-visible:ring-2 focus-visible:ring-ring data-[state=checked]:border-brand data-[state=checked]:ring-1 data-[state=checked]:ring-brand"
            >
              <LayoutGlyph layout={l.value} />
              <span className="font-medium text-text-primary">{l.label}</span>
              <span className="text-caption text-text-tertiary">{l.hint}</span>
            </RadioGroup.Item>
          ))}
        </RadioGroup.Root>
      </div>

      <div className="flex flex-col gap-3">
        <Label id={`${ids}-mode`}>明暗模式</Label>
        <Segmented
          aria-labelledby={`${ids}-mode`}
          value={value.mode}
          onValueChange={(mode) => set({ mode })}
          options={MODES}
          className="self-start"
        />
      </div>
    </div>
  );
}

function LayoutGlyph({ layout }: { layout: ShopTheme["layout"] }) {
  return layout === "grid" ? (
    <span className="grid w-full grid-cols-3 gap-1" aria-hidden>
      {[0, 1, 2, 3, 4, 5].map((i) => (
        <span key={i} className="aspect-[4/3] rounded-sm bg-muted" />
      ))}
    </span>
  ) : (
    <span className="flex w-full flex-col gap-1" aria-hidden>
      {[0, 1, 2].map((i) => (
        <span key={i} className="flex items-center gap-1.5">
          <span className="aspect-[4/3] w-6 rounded-sm bg-muted" />
          <span className="h-1.5 flex-1 rounded-sm bg-muted" />
        </span>
      ))}
    </span>
  );
}
