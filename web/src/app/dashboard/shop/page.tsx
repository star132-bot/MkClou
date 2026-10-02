"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { Plus, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Controller, useFieldArray, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";

import { Field } from "@/components/auth/form-field";
import { ShopPreview } from "@/components/dashboard/shop-preview";
import { SlugField, useSlugCheck } from "@/components/dashboard/slug-field";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { ThemeEditor } from "@/components/dashboard/theme-editor";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { ApiError, ErrorCode } from "@/lib/api/client";
import { shopApi } from "@/lib/shop/api";
import { useShopStore } from "@/lib/shop/store";
import type { MerchantShop, ShopPatch } from "@/lib/shop/types";
import { pauseNoteSchema, SOCIAL_TYPES, shopSettingsSchema, type ShopSettingsValues } from "@/lib/shop/validation";
import type { PublicShop, SocialLink } from "@/lib/storefront/types";

type FormValues = ShopSettingsValues;

function toValues(s: MerchantShop): FormValues {
  return {
    name: s.name,
    slug: s.slug,
    description: s.description,
    contactEmail: s.contactEmail,
    socialLinks: s.socialLinks,
    theme: s.theme,
  };
}

/** 只提交有变化的字段（PATCH 语义），未修改的链接不会触发 30 天限制。 */
function diff(shop: MerchantShop, v: FormValues): ShopPatch {
  const patch: ShopPatch = {};
  const name = v.name.trim();
  if (name !== shop.name) patch.name = name;
  if (v.slug !== shop.slug) patch.slug = v.slug;
  const description = v.description.trim();
  if (description !== shop.description) patch.description = description;
  const email = v.contactEmail.trim().toLowerCase();
  if (email !== shop.contactEmail) patch.contactEmail = email;
  const links = v.socialLinks.filter((l) => l.url.trim()).map((l) => ({ ...l, url: l.url.trim() })) as SocialLink[];
  if (JSON.stringify(links) !== JSON.stringify(shop.socialLinks)) patch.socialLinks = links;
  if (JSON.stringify(v.theme) !== JSON.stringify(shop.theme)) patch.theme = v.theme;
  return patch;
}

const dateFmt = new Intl.DateTimeFormat("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit" });

// 店铺设置（PRD SHOP-02 基本信息、SHOP-03 装修、SHOP-06 营业状态）
export default function ShopSettingsPage() {
  const shop = useShopStore((s) => s.shop)!;
  const setShop = useShopStore((s) => s.setShop);
  const setDirty = useShopStore((s) => s.setDirty);
  const [serverSuggestions, setServerSuggestions] = useState<string[]>([]);

  const {
    register,
    control,
    handleSubmit,
    reset,
    setError,
    clearErrors,
    formState: { errors, isDirty, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(shopSettingsSchema),
    mode: "onTouched",
    defaultValues: toValues(shop),
  });
  const links = useFieldArray({ control, name: "socialLinks" });
  const values = useWatch({ control }) as FormValues;

  const slugStatus = useSlugCheck(values.slug ?? "", values.slug === shop.slug);

  // 同步“有未保存修改”状态：站内导航由 GuardedLink 弹窗确认，关闭或刷新页面由浏览器提示
  useEffect(() => {
    setDirty(isDirty);
    if (!isDirty) return;
    const onBeforeUnload = (e: BeforeUnloadEvent) => e.preventDefault();
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, [isDirty, setDirty]);
  useEffect(() => () => setDirty(false), [setDirty]);

  const preview = useMemo<PublicShop>(
    () => ({
      slug: values.slug || shop.slug,
      name: values.name?.trim() || shop.name,
      description: values.description?.trim() ?? "",
      avatarUrl: shop.avatarUrl,
      coverUrl: shop.coverUrl,
      contactEmail: values.contactEmail ?? "",
      socialLinks: (values.socialLinks ?? []).filter((l) => l.url?.trim()) as SocialLink[],
      theme: values.theme,
      status: shop.status,
      pauseNote: shop.pauseNote,
      isTestMode: false,
    }),
    [values, shop],
  );

  const onSubmit = async (v: FormValues) => {
    if (slugStatus.state === "unavailable") return;
    const patch = diff(shop, v);
    if (Object.keys(patch).length === 0) {
      reset(toValues(shop));
      return;
    }
    try {
      const updated = await shopApi.update(patch);
      setShop(updated);
      reset(toValues(updated));
      setServerSuggestions([]);
      toast.success("店铺设置已保存", patch.slug ? { description: "旧链接在 90 天内会自动跳转到新链接" } : undefined);
    } catch (e) {
      if (!(e instanceof ApiError)) {
        toast.error("保存失败，请检查网络后重试");
        return;
      }
      const fields = e.fields;
      if (e.code === ErrorCode.SlugUnavailable || e.code === ErrorCode.SlugChangeTooSoon) {
        setError("slug", { message: e.message });
        setServerSuggestions((e.data as { suggestions?: string[] } | null)?.suggestions ?? []);
      } else if (Object.keys(fields).length > 0) {
        for (const [k, msg] of Object.entries(fields)) {
          setError((k.startsWith("theme") ? "root" : k) as keyof FormValues, { message: msg });
        }
        toast.error(Object.values(fields)[0]);
      } else {
        toast.error(e.message);
      }
    }
  };

  const nextSlugChange = shop.nextSlugChangeAt ? dateFmt.format(new Date(shop.nextSlugChangeAt)) : null;
  const usedTypes = new Set((values.socialLinks ?? []).map((l) => l.type));

  return (
    <main className="mx-auto w-full max-w-[1280px] px-4 pt-8 pb-32 md:px-8">
      <h1 className="text-h1">店铺设置</h1>

      <div className="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] xl:grid-cols-[minmax(0,5fr)_minmax(0,6fr)]">
        <div className="flex min-w-0 flex-col gap-6">
          <form id="shop-settings" onSubmit={handleSubmit(onSubmit)} noValidate className="flex flex-col gap-6">
          <Card title="基本信息">
            <Field label="店铺名称" error={errors.name?.message}>
              {({ id, describedBy, invalid }) => (
                <Input id={id} maxLength={20} aria-describedby={describedBy} aria-invalid={invalid} {...register("name")} />
              )}
            </Field>

            <Controller
              control={control}
              name="slug"
              render={({ field }) => (
                <SlugField
                  value={field.value}
                  onChange={(v) => {
                    field.onChange(v);
                    clearErrors("slug");
                    setServerSuggestions([]);
                  }}
                  onBlur={field.onBlur}
                  status={slugStatus}
                  error={errors.slug?.message}
                  suggestions={serverSuggestions}
                  disabled={!shop.slugChangeAllowed}
                  hint={
                    shop.slugChangeAllowed
                      ? "每 30 天只能修改一次，修改后旧链接 90 天内会自动跳转"
                      : `链接每 30 天只能修改一次，${nextSlugChange} 后可再次修改`
                  }
                />
              )}
            />

            <Field
              label="店铺简介（可选）"
              error={errors.description?.message}
              hint={`${values.description?.length ?? 0}/200，展示在店铺页店名下方`}
            >
              {({ id, describedBy, invalid }) => (
                <Textarea id={id} rows={3} maxLength={200} aria-describedby={describedBy} aria-invalid={invalid} {...register("description")} />
              )}
            </Field>

            <Field label="联系邮箱" error={errors.contactEmail?.message} hint="展示在订单交付页和邮件中，方便买家联系你处理售后">
              {({ id, describedBy, invalid }) => (
                <Input id={id} type="email" autoComplete="email" aria-describedby={describedBy} aria-invalid={invalid} {...register("contactEmail")} />
              )}
            </Field>

            <fieldset className="flex flex-col gap-2">
              <legend className="mb-2 text-body font-medium text-text-primary">社交链接（可选）</legend>
              {links.fields.map((f, i) => (
                <div key={f.id} className="flex flex-col gap-1">
                  <div className="flex gap-2">
                    <select
                      aria-label="链接类型"
                      className="h-9 w-28 shrink-0 rounded-md border border-input bg-background px-2 text-body text-text-primary outline-none focus-visible:border-brand focus-visible:ring-3 focus-visible:ring-brand/15"
                      {...register(`socialLinks.${i}.type`)}
                    >
                      {SOCIAL_TYPES.map((t) => (
                        <option key={t.value} value={t.value}>
                          {t.label}
                        </option>
                      ))}
                    </select>
                    <Input
                      aria-label="链接地址"
                      placeholder={SOCIAL_TYPES.find((t) => t.value === values.socialLinks?.[i]?.type)?.placeholder}
                      aria-invalid={!!errors.socialLinks?.[i]?.url}
                      {...register(`socialLinks.${i}.url`)}
                    />
                    <Button type="button" variant="ghost" size="icon" onClick={() => links.remove(i)} aria-label="删除这个链接">
                      <Trash2 strokeWidth={1.5} />
                    </Button>
                  </div>
                  {errors.socialLinks?.[i]?.url && <p className="text-caption text-danger">{errors.socialLinks[i].url.message}</p>}
                </div>
              ))}
              {errors.socialLinks?.message && <p className="text-caption text-danger">{errors.socialLinks.message}</p>}
              {links.fields.length < 4 && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="self-start"
                  onClick={() =>
                    links.append({ type: SOCIAL_TYPES.find((t) => !usedTypes.has(t.value))?.value ?? "website", url: "" })
                  }
                >
                  <Plus strokeWidth={1.5} data-icon="inline-start" />
                  添加链接
                </Button>
              )}
            </fieldset>
          </Card>

          <Card title="店铺装修" description="只提供经过设计的配置项，任何组合都能保持清晰好看。">
            <Controller
              control={control}
              name="theme"
              render={({ field }) => <ThemeEditor value={field.value} onChange={field.onChange} />}
            />
            {errors.root?.message && <p className="text-caption text-danger">{errors.root.message}</p>}
          </Card>
          </form>

          {/* 营业状态独立提交，放在表单外，避免在暂停说明中按回车时提交店铺设置 */}
          <StatusCard />
        </div>

        <aside className="min-w-0">
          <div className="lg:sticky lg:top-6">
            <ShopPreview shop={preview} />
          </div>
        </aside>
      </div>

      {isDirty && (
        <div className="fixed inset-x-0 bottom-0 z-40 border-t border-border bg-background/95 backdrop-blur-sm">
          <div className="mx-auto flex max-w-[1280px] items-center justify-end gap-3 px-4 py-3 md:px-8">
            <span className="mr-auto text-text-tertiary">有未保存的修改</span>
            <Button type="button" variant="outline" onClick={() => reset(toValues(shop))} disabled={isSubmitting}>
              取消
            </Button>
            <Button type="submit" form="shop-settings" disabled={isSubmitting || slugStatus.state === "checking"}>
              {isSubmitting ? "保存中…" : "保存修改"}
            </Button>
          </div>
        </div>
      )}
    </main>
  );
}

function Card({ title, description, children }: { title: string; description?: string; children: React.ReactNode }) {
  return (
    <section className="flex flex-col gap-6 rounded-lg border border-border bg-background p-6">
      <div>
        <h2 className="text-h3">{title}</h2>
        {description && <p className="mt-1 text-caption text-text-tertiary">{description}</p>}
      </div>
      {children}
    </section>
  );
}

/** 营业状态（PRD SHOP-06）：独立于表单，操作后立即生效。 */
function StatusCard() {
  const shop = useShopStore((s) => s.shop)!;
  const setShop = useShopStore((s) => s.setShop);
  const [note, setNote] = useState("");
  const [error, setError] = useState<string>();
  const [pending, setPending] = useState(false);

  const change = async (status: "OPEN" | "PAUSED") => {
    const parsed = pauseNoteSchema.safeParse(note);
    if (status === "PAUSED" && !parsed.success) {
      setError(parsed.error.issues[0].message);
      return;
    }
    setPending(true);
    try {
      setShop(await shopApi.setStatus(status, status === "PAUSED" ? note.trim() : ""));
      setNote("");
      setError(undefined);
      toast.success(status === "PAUSED" ? "店铺已暂停营业" : "店铺已恢复营业");
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "操作失败，请稍后重试");
    } finally {
      setPending(false);
    }
  };

  return (
    <Card title="营业状态">
      <div className="flex items-center gap-2">
        <span className="text-text-primary">当前状态</span>
        <StatusBadge status={shop.status} />
      </div>
      {shop.status === "BANNED" ? (
        <p>店铺已被平台封禁，买家无法访问。如有疑问请联系平台客服。</p>
      ) : shop.status === "PAUSED" ? (
        <>
          <p>
            买家可以浏览商品，但无法下单。
            {shop.pauseNote && <>暂停说明：{shop.pauseNote}</>}
          </p>
          <Button type="button" className="self-start" onClick={() => change("OPEN")} disabled={pending}>
            {pending ? "处理中…" : "恢复营业"}
          </Button>
        </>
      ) : (
        <>
          <p>暂停营业后，买家仍可浏览商品但无法下单；已创建的待支付订单仍可完成支付。</p>
          <Field label="暂停说明（可选）" error={error} hint={`${note.length}/100，展示给买家，例如“国庆休息，10 月 8 日恢复”`}>
            {({ id, describedBy, invalid }) => (
              <Input
                id={id}
                value={note}
                maxLength={100}
                onChange={(e) => setNote(e.target.value)}
                aria-describedby={describedBy}
                aria-invalid={invalid}
              />
            )}
          </Field>
          <Button type="button" variant="outline" className="self-start" onClick={() => change("PAUSED")} disabled={pending}>
            {pending ? "处理中…" : "暂停营业"}
          </Button>
        </>
      )}
    </Card>
  );
}
