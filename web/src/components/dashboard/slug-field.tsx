"use client";

import { Check, LoaderCircle, X } from "lucide-react";
import { useEffect, useState, useSyncExternalStore } from "react";

import { Field } from "@/components/auth/form-field";
import { Input } from "@/components/ui/input";
import { shopApi } from "@/lib/shop/api";
import { sanitizeSlug, slugFormatError } from "@/lib/shop/validation";
import { cn } from "@/lib/utils";

export type SlugStatus =
  | { state: "idle" }
  | { state: "checking" }
  | { state: "available" }
  | { state: "unavailable"; reason: string; suggestions: string[] };

/** 链接输入时防抖 500ms 检查可用性（PRD SHOP-01）。skip 为 true 时不检查（如链接未修改）。 */
export function useSlugCheck(slug: string, skip = false): SlugStatus {
  const [status, setStatus] = useState<SlugStatus>({ state: "idle" });
  const formatError = slugFormatError(slug);

  useEffect(() => {
    if (skip || !slug || formatError) return;
    let cancelled = false;
    const t = setTimeout(async () => {
      setStatus({ state: "checking" });
      try {
        const r = await shopApi.checkSlug(slug);
        if (cancelled) return;
        setStatus(r.available ? { state: "available" } : { state: "unavailable", reason: r.reason, suggestions: r.suggestions });
      } catch {
        if (!cancelled) setStatus({ state: "idle" }); // 检查失败不阻塞，提交时由服务端兜底
      }
    }, 500);
    return () => {
      cancelled = true;
      clearTimeout(t);
    };
  }, [slug, skip, formatError]);

  if (skip || !slug || formatError) return { state: "idle" };
  return status;
}

const noopSubscribe = () => () => {};

/** 店铺链接输入框：固定前缀 + 可用性标识 + 不可用时的推荐备选。 */
export function SlugField({
  value,
  onChange,
  onBlur,
  status,
  error,
  hint,
  disabled,
  suggestions: extraSuggestions,
}: {
  value: string;
  onChange: (v: string) => void;
  onBlur?: () => void;
  status: SlugStatus;
  error?: string;
  hint?: React.ReactNode;
  disabled?: boolean;
  /** 提交时服务端返回的推荐（并发占用的情况） */
  suggestions?: string[];
}) {
  const origin = useSyncExternalStore(noopSubscribe, () => window.location.host, () => "mkclou.com");

  const unavailable = status.state === "unavailable" ? status : null;
  const message = error ?? unavailable?.reason;
  const suggestions = unavailable?.suggestions.length ? unavailable.suggestions : (extraSuggestions ?? []);

  return (
    <Field label="店铺链接" error={message} hint={hint}>
      {({ id, describedBy, invalid }) => (
        <div className="flex flex-col gap-2">
          <div
            className={cn(
              "flex h-9 items-center rounded-md border border-input bg-background transition-[border-color,box-shadow] duration-150 focus-within:border-brand focus-within:ring-3 focus-within:ring-brand/15",
              invalid && "border-danger focus-within:border-danger focus-within:ring-danger/15",
              disabled && "bg-muted opacity-50",
            )}
          >
            <span className="shrink-0 pl-3 text-text-tertiary select-none">{origin}/s/</span>
            <Input
              id={id}
              value={value}
              onChange={(e) => onChange(sanitizeSlug(e.target.value))}
              onBlur={onBlur}
              disabled={disabled}
              autoComplete="off"
              autoCapitalize="off"
              spellCheck={false}
              maxLength={20}
              aria-describedby={describedBy}
              aria-invalid={invalid || !!unavailable}
              className="h-full border-0 pl-0.5 focus-visible:ring-0 disabled:bg-transparent disabled:opacity-100"
            />
            <span className="flex w-9 shrink-0 justify-center" aria-live="polite">
              {status.state === "checking" && (
                <LoaderCircle className="size-4 animate-spin text-text-tertiary" strokeWidth={1.5} aria-label="正在检查" />
              )}
              {status.state === "available" && !error && (
                <Check className="size-4 text-success" strokeWidth={2} aria-label="链接可用" />
              )}
              {unavailable && <X className="size-4 text-danger" strokeWidth={2} aria-label="链接不可用" />}
            </span>
          </div>
          {suggestions.length > 0 && !disabled && (
            <div className="flex flex-wrap items-center gap-2 text-caption">
              <span className="text-text-tertiary">可用的链接：</span>
              {suggestions.map((s) => (
                <button
                  key={s}
                  type="button"
                  onClick={() => onChange(s)}
                  className="rounded-sm bg-muted px-2 py-0.5 font-medium text-text-primary transition-colors hover:text-brand focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
                >
                  {s}
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </Field>
  );
}
