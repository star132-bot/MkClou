"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowLeft, ExternalLink, Plus, Trash2 } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { Controller, useFieldArray, useForm, useWatch } from "react-hook-form";
import { RadioGroup } from "radix-ui";
import { toast } from "sonner";
import { z } from "zod";

import { Field } from "@/components/auth/form-field";
import { GuardedLink } from "@/components/dashboard/dashboard-header";
import { MarkdownBody } from "@/components/storefront/product-content";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Segmented } from "@/components/ui/segmented";
import { Textarea } from "@/components/ui/textarea";
import { ApiError, ErrorCode } from "@/lib/api/client";
import { productApi } from "@/lib/product/api";
import { CATEGORIES, centsToYuan, DELIVERY_TYPES, yuanToCents } from "@/lib/product/meta";
import type { CategoryValue, EditableProduct, ProductImageRef, ProductSavePayload, PublishIssue } from "@/lib/product/types";
import { useShopStore } from "@/lib/shop/store";
import type { DeliveryType } from "@/lib/storefront/types";
import { cn } from "@/lib/utils";

import { ImageManager } from "./image-manager";
import { ProductStatusBadge } from "./status-badge";

// 与后端 internal/product/validate.go 一致的格式校验；上架所需的完整性由服务端检查（PRD-08）
const priceText = (msg: string) =>
  z.string().trim().refine((v) => {
    const c = yuanToCents(v);
    return c !== null && c <= 5_000_000;
  }, msg);

const schema = z
  .object({
    name: z.string().trim().min(2, "商品名称为 2～60 个字符").max(60, "商品名称为 2～60 个字符"),
    tagline: z.string().trim().max(80, "一句话卖点最多 80 个字符"),
    category: z.string(),
    price: priceText("请输入 0～50000 之间的价格，最多两位小数"),
    originalPrice: z.union([z.literal(""), priceText("请输入有效的划线原价")]),
    descriptionMd: z.string().max(5000, "商品介绍最多 5000 个字符"),
    includes: z.array(z.object({ value: z.string().trim().max(50, "每项最多 50 个字符") })).max(10),
    audience: z.string().trim().max(200, "最多 200 个字符"),
    faqs: z
      .array(z.object({ q: z.string().trim().max(50, "问题最多 50 个字符"), a: z.string().trim().max(300, "回答最多 300 个字符") }))
      .max(8),
    notice: z.string().trim().max(300, "最多 300 个字符"),
    deliveryType: z.enum(["LINK", "TEXT", "FILE", "CARD"]),
    links: z
      .array(
        z.object({
          name: z.string().trim().max(20, "最多 20 个字符"),
          url: z
            .string()
            .trim()
            .refine((v) => v === "" || /^https?:\/\/[^\s/]+/i.test(v), "请输入以 http:// 或 https:// 开头的链接"),
          code: z.string().trim().max(20, "最多 20 个字符"),
        }),
      )
      .max(5),
    text: z.string().max(5000, "最多 5000 个字符"),
    note: z.string().trim().max(500, "交付附言最多 500 个字符"),
    images: z.array(z.object({ key: z.string(), url: z.string(), width: z.number(), height: z.number() })).max(6),
  })
  .superRefine((v, ctx) => {
    const op = v.originalPrice ? yuanToCents(v.originalPrice) : null;
    const p = yuanToCents(v.price);
    if (op !== null && p !== null && op <= p) {
      ctx.addIssue({ code: "custom", path: ["originalPrice"], message: "划线原价必须大于售价" });
    }
    v.faqs.forEach((f, i) => {
      if ((f.q === "") !== (f.a === "")) ctx.addIssue({ code: "custom", path: ["faqs", i, "a"], message: "问题和回答都需要填写" });
    });
  });

type FormValues = z.infer<typeof schema>;

function toValues(p: EditableProduct): FormValues {
  return {
    name: p.name,
    tagline: p.tagline,
    category: p.category ?? "",
    price: centsToYuan(p.price),
    originalPrice: p.originalPrice === null ? "" : centsToYuan(p.originalPrice),
    descriptionMd: p.descriptionMd,
    includes: p.detail.includes.map((value) => ({ value })),
    audience: p.detail.audience,
    faqs: p.detail.faqs,
    notice: p.detail.notice,
    deliveryType: p.deliveryType,
    links: p.deliveryConfig.links,
    text: p.deliveryConfig.text,
    note: p.deliveryConfig.note,
    images: p.images,
  };
}

function toPayload(v: FormValues, version: number): ProductSavePayload {
  return {
    version,
    name: v.name.trim(),
    tagline: v.tagline.trim(),
    category: (v.category || null) as CategoryValue | null,
    price: yuanToCents(v.price)!,
    originalPrice: v.originalPrice ? yuanToCents(v.originalPrice) : null,
    deliveryType: v.deliveryType,
    descriptionMd: v.descriptionMd,
    detail: {
      includes: v.includes.map((i) => i.value),
      audience: v.audience,
      faqs: v.faqs,
      notice: v.notice,
    },
    deliveryConfig: { links: v.links, text: v.text, cardInstructions: "", note: v.note },
    maxPerOrder: 1,
    images: v.images.map(({ key, width, height }) => ({ key, width, height })),
  };
}

/** 服务端字段名 → 表单中可定位的元素 id。 */
const fieldAnchor: Record<string, string> = {
  name: "pe-name",
  category: "pe-category",
  images: "pe-images",
  descriptionMd: "pe-description",
  "deliveryConfig.links": "pe-delivery",
  "deliveryConfig.text": "pe-delivery",
  account: "",
  payment: "",
};

// 新建 / 编辑商品（PRD-02）
export function ProductEditor({ initial }: { initial: EditableProduct }) {
  const ids = useId();
  const setDirty = useShopStore((s) => s.setDirty);
  const shopSlug = useShopStore((s) => s.shop?.slug);
  const [product, setProduct] = useState(initial);
  const [issues, setIssues] = useState<PublishIssue[] | null>(null);
  const [busy, setBusy] = useState<"save" | "publish" | "unpublish" | null>(null);
  const [descTab, setDescTab] = useState<"edit" | "preview">("edit");

  const {
    register,
    control,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isDirty },
  } = useForm<FormValues>({ resolver: zodResolver(schema), mode: "onTouched", defaultValues: toValues(initial) });
  const includes = useFieldArray({ control, name: "includes" });
  const faqs = useFieldArray({ control, name: "faqs" });
  const links = useFieldArray({ control, name: "links" });
  const [deliveryType, descriptionMd, price] = useWatch({ control, name: ["deliveryType", "descriptionMd", "price"] });

  // 未保存离开提醒：站内链接由 GuardedLink 弹窗确认，关闭页面由浏览器提示
  useEffect(() => {
    setDirty(isDirty);
    if (!isDirty) return;
    const onBeforeUnload = (e: BeforeUnloadEvent) => e.preventDefault();
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, [isDirty, setDirty]);
  useEffect(() => () => setDirty(false), [setDirty]);

  const applySaved = (p: EditableProduct) => {
    setProduct(p);
    reset(toValues(p));
  };

  const handleError = (e: unknown) => {
    if (!(e instanceof ApiError)) {
      toast.error("网络连接异常，请检查网络后重试");
      return;
    }
    if (e.code === ErrorCode.PublishCheckFailed) {
      setIssues((e.data as { issues: PublishIssue[] }).issues);
      return;
    }
    if (e.code === ErrorCode.VersionConflict) {
      toast.error(e.message, {
        action: { label: "加载最新版本", onClick: () => productApi.get(product.publicId).then(applySaved) },
        duration: 10_000,
      });
      return;
    }
    const f = e.fields;
    const map: Record<string, keyof FormValues> = {
      name: "name", tagline: "tagline", category: "category", price: "price", originalPrice: "originalPrice",
      descriptionMd: "descriptionMd", images: "images", "deliveryConfig.links": "links", "deliveryConfig.text": "text",
      "deliveryConfig.note": "note", "detail.includes": "includes", "detail.faqs": "faqs", "detail.audience": "audience",
      "detail.notice": "notice", deliveryType: "deliveryType",
    };
    for (const [k, msg] of Object.entries(f)) {
      if (map[k]) setError(map[k], { message: msg });
    }
    toast.error(e.message);
  };

  // 保存当前表单；返回保存后的商品，校验或保存失败返回 null
  const save = (): Promise<EditableProduct | null> =>
    new Promise((resolve) => {
      void handleSubmit(
        async (v) => {
          try {
            const p = await productApi.save(product.publicId, toPayload(v, product.version));
            applySaved(p);
            resolve(p);
          } catch (e) {
            handleError(e);
            resolve(null);
          }
        },
        () => {
          toast.error("请先修正标红的内容");
          resolve(null);
        },
      )();
    });

  const onSave = async () => {
    setBusy("save");
    const p = await save();
    setBusy(null);
    if (p) toast.success(p.status === "PENDING_REVIEW" ? "已保存，内容需要审核，通过后自动上架" : "已保存");
  };

  const onPublish = async () => {
    setBusy("publish");
    try {
      const saved = isDirty ? await save() : product;
      if (!saved) return;
      const p = await productApi.publish(saved.publicId);
      applySaved(p);
      toast.success(p.status === "PENDING_REVIEW" ? "已提交审核，通过后自动上架" : "商品已上架");
    } catch (e) {
      handleError(e);
    } finally {
      setBusy(null);
    }
  };

  const onUnpublish = async () => {
    setBusy("unpublish");
    try {
      applySaved(await productApi.unpublish(product.publicId));
      toast.success("商品已下架");
    } catch (e) {
      handleError(e);
    } finally {
      setBusy(null);
    }
  };

  const onSale = product.status === "ON_SALE" || product.status === "PENDING_REVIEW";
  const priceCents = yuanToCents(price ?? "");

  return (
    <main className="mx-auto w-full max-w-[880px] px-4 pb-24 md:px-8">
      {/* 顶部操作栏 */}
      <div className="sticky top-0 z-30 -mx-4 flex flex-wrap items-center gap-3 border-b border-border bg-surface/95 px-4 py-3 backdrop-blur-sm md:-mx-8 md:px-8">
        <Button variant="ghost" size="icon-sm" asChild>
          <GuardedLink href="/dashboard/products">
            <ArrowLeft className="size-4" strokeWidth={1.5} aria-label="返回商品列表" />
          </GuardedLink>
        </Button>
        <span className="min-w-0 flex-1 truncate font-medium text-text-primary">{product.name}</span>
        <ProductStatusBadge status={product.status} />
        {product.status === "ON_SALE" && shopSlug && (
          <Button variant="ghost" size="sm" asChild>
            <a href={`/s/${shopSlug}/p/${product.publicId}`} target="_blank" rel="noopener noreferrer">
              查看
              <ExternalLink strokeWidth={1.5} data-icon="inline-end" />
            </a>
          </Button>
        )}
        {onSale ? (
          <>
            <Button variant="outline" size="sm" onClick={onUnpublish} disabled={busy !== null}>
              {busy === "unpublish" ? "下架中…" : "下架"}
            </Button>
            <Button size="sm" onClick={onSave} disabled={busy !== null || !isDirty}>
              {busy === "save" ? "保存中…" : "保存"}
            </Button>
          </>
        ) : (
          <>
            <Button variant="outline" size="sm" onClick={onSave} disabled={busy !== null || !isDirty}>
              {busy === "save" ? "保存中…" : "保存草稿"}
            </Button>
            <Button size="sm" onClick={onPublish} disabled={busy !== null || product.status === "BANNED"}>
              {busy === "publish" ? "上架中…" : "上架"}
            </Button>
          </>
        )}
      </div>

      {product.status === "BANNED" && (
        <p className="mt-6 rounded-md bg-danger/10 px-4 py-3 text-danger" role="alert">
          该商品已被平台下架{product.banReason ? `：${product.banReason}` : ""}。修改后请联系平台申诉。
        </p>
      )}
      {product.status === "PENDING_REVIEW" && (
        <p className="mt-6 rounded-md bg-info/10 px-4 py-3 text-info" role="status">
          商品内容命中了平台审核规则，正在等待人工审核，通过后自动上架。修改相关内容后保存，可以重新自动审核。
        </p>
      )}

      <form onSubmit={(e) => e.preventDefault()} noValidate className="mt-6 flex flex-col gap-6">
        <Card title="基本信息">
          <Field label="商品名称" error={errors.name?.message}>
            {({ id, describedBy, invalid }) => (
              <Input id={id} maxLength={60} aria-describedby={describedBy} aria-invalid={invalid} {...register("name")} />
            )}
          </Field>
          <Field label="一句话卖点（可选）" error={errors.tagline?.message} hint="显示在商品名称下方，例如“3 套模板，拿来即用”">
            {({ id, describedBy, invalid }) => (
              <Input id={id} maxLength={80} aria-describedby={describedBy} aria-invalid={invalid} {...register("tagline")} />
            )}
          </Field>
          <div className="flex flex-col gap-2" id="pe-category">
            <Label id={`${ids}-cat`}>商品分类</Label>
            <Controller
              control={control}
              name="category"
              render={({ field }) => (
                <RadioGroup.Root
                  aria-labelledby={`${ids}-cat`}
                  value={field.value}
                  onValueChange={field.onChange}
                  className="flex flex-wrap gap-2"
                >
                  {CATEGORIES.map((c) => (
                    <RadioGroup.Item
                      key={c.value}
                      value={c.value}
                      className="h-8 rounded-full border border-border px-3 text-body font-medium text-text-secondary transition-colors outline-none hover:text-text-primary focus-visible:ring-2 focus-visible:ring-ring data-[state=checked]:border-text-primary data-[state=checked]:bg-text-primary data-[state=checked]:text-background"
                    >
                      {c.label}
                    </RadioGroup.Item>
                  ))}
                </RadioGroup.Root>
              )}
            />
            <p className="text-caption text-text-tertiary">用于商城首页和搜索的分类筛选，上架前必须选择。</p>
          </div>
        </Card>

        <Card title="封面图" id="pe-images">
          <Controller
            control={control}
            name="images"
            render={({ field }) => (
              <ImageManager value={field.value as ProductImageRef[]} onChange={field.onChange} invalid={!!errors.images} />
            )}
          />
          {errors.images?.message && <p className="text-caption text-danger">{errors.images.message}</p>}
        </Card>

        <Card title="价格">
          <div className="grid gap-4 sm:grid-cols-2">
            <Field
              label="售价（元）"
              error={errors.price?.message}
              hint={priceCents === 0 ? "将作为免费商品，买家填写邮箱即可领取" : "0 表示免费；最高 50000"}
            >
              {({ id, describedBy, invalid }) => (
                <Input id={id} inputMode="decimal" aria-describedby={describedBy} aria-invalid={invalid} {...register("price")} />
              )}
            </Field>
            <Field label="划线原价（可选）" error={errors.originalPrice?.message} hint="需大于售价，用于展示优惠">
              {({ id, describedBy, invalid }) => (
                <Input id={id} inputMode="decimal" aria-describedby={describedBy} aria-invalid={invalid} {...register("originalPrice")} />
              )}
            </Field>
          </div>
        </Card>

        <Card title="商品描述" description="平台按固定版式排版这些内容，保证商品页整洁一致。">
          <div className="flex flex-col gap-2" id="pe-description">
            <div className="flex items-center justify-between">
              <Label htmlFor={`${ids}-desc`}>商品介绍</Label>
              <Segmented
                aria-label="商品介绍编辑方式"
                value={descTab}
                onValueChange={setDescTab}
                options={[
                  { value: "edit", label: "编辑" },
                  { value: "preview", label: "预览" },
                ]}
              />
            </div>
            {descTab === "edit" ? (
              <Textarea
                id={`${ids}-desc`}
                rows={10}
                maxLength={5000}
                aria-invalid={!!errors.descriptionMd}
                className="font-mono text-body"
                placeholder={"支持 Markdown：## 标题、**加粗**、- 列表、[链接](https://…)"}
                {...register("descriptionMd")}
              />
            ) : (
              <div className="min-h-[200px] rounded-md border border-border p-4">
                {descriptionMd?.trim() ? <MarkdownBody>{descriptionMd}</MarkdownBody> : <p className="text-text-tertiary">还没有内容</p>}
              </div>
            )}
            <p className={cn("text-caption", errors.descriptionMd ? "text-danger" : "text-text-tertiary")}>
              {errors.descriptionMd?.message ?? `${descriptionMd?.length ?? 0}/5000，上架前必须填写`}
            </p>
          </div>

          <fieldset className="flex flex-col gap-2">
            <legend className="mb-2 font-medium text-text-primary">包含内容（可选）</legend>
            {includes.fields.map((f, i) => (
              <div key={f.id} className="flex gap-2">
                <Input aria-label={`第 ${i + 1} 项`} maxLength={50} placeholder="例如：3 套 Figma 模板" aria-invalid={!!errors.includes?.[i]?.value} {...register(`includes.${i}.value`)} />
                <RemoveButton onClick={() => includes.remove(i)} />
              </div>
            ))}
            {includes.fields.length < 10 && <AddButton onClick={() => includes.append({ value: "" })}>添加一项</AddButton>}
          </fieldset>

          <Field label="适合谁（可选）" error={errors.audience?.message}>
            {({ id, describedBy, invalid }) => (
              <Textarea id={id} rows={2} maxLength={200} aria-describedby={describedBy} aria-invalid={invalid} {...register("audience")} />
            )}
          </Field>

          <fieldset className="flex flex-col gap-3">
            <legend className="mb-2 font-medium text-text-primary">常见问题（可选）</legend>
            {faqs.fields.map((f, i) => (
              <div key={f.id} className="flex gap-2">
                <div className="flex flex-1 flex-col gap-2 rounded-md border border-border p-3">
                  <Input aria-label={`问题 ${i + 1}`} placeholder="问题" maxLength={50} {...register(`faqs.${i}.q`)} />
                  <Textarea aria-label={`回答 ${i + 1}`} placeholder="回答" rows={2} maxLength={300} aria-invalid={!!errors.faqs?.[i]?.a} {...register(`faqs.${i}.a`)} />
                  {errors.faqs?.[i]?.a && <p className="text-caption text-danger">{errors.faqs[i].a.message}</p>}
                </div>
                <RemoveButton onClick={() => faqs.remove(i)} />
              </div>
            ))}
            {faqs.fields.length < 8 && <AddButton onClick={() => faqs.append({ q: "", a: "" })}>添加问答</AddButton>}
          </fieldset>

          <Field label="购买须知（可选）" error={errors.notice?.message} hint="例如“虚拟商品，交付后不支持退款”">
            {({ id, describedBy, invalid }) => (
              <Textarea id={id} rows={2} maxLength={300} aria-describedby={describedBy} aria-invalid={invalid} {...register("notice")} />
            )}
          </Field>
        </Card>

        <Card title="交付内容" id="pe-delivery" description="只有付款成功的买家能看到，会显示在交付页和邮件中。">
          <div className="flex flex-col gap-2">
            <Label id={`${ids}-dtype`}>交付类型</Label>
            {product.deliveryTypeLocked ? (
              <p className="text-text-primary">
                {DELIVERY_TYPES.find((d) => d.value === deliveryType)?.label}
                <span className="ml-2 text-caption text-text-tertiary">商品上架过，不能修改交付类型；如需修改请新建商品</span>
              </p>
            ) : (
              <Controller
                control={control}
                name="deliveryType"
                render={({ field }) => (
                  <Segmented
                    aria-labelledby={`${ids}-dtype`}
                    value={field.value}
                    onValueChange={(v) => field.onChange(v as DeliveryType)}
                    options={DELIVERY_TYPES.filter((d) => d.available).map((d) => ({ value: d.value, label: d.label }))}
                    className="self-start"
                  />
                )}
              />
            )}
          </div>

          {deliveryType === "LINK" ? (
            <fieldset className="flex flex-col gap-3">
              <legend className="mb-2 font-medium text-text-primary">链接（最多 5 个）</legend>
              {links.fields.map((f, i) => (
                <div key={f.id} className="flex gap-2">
                  <div className="grid flex-1 gap-2 rounded-md border border-border p-3 sm:grid-cols-[120px_minmax(0,1fr)_100px]">
                    <Input aria-label="链接名称" placeholder="百度网盘" maxLength={20} {...register(`links.${i}.name`)} />
                    <Input aria-label="链接地址" placeholder="https://pan.baidu.com/s/…" aria-invalid={!!errors.links?.[i]?.url} {...register(`links.${i}.url`)} />
                    <Input aria-label="提取码" placeholder="提取码" maxLength={20} {...register(`links.${i}.code`)} />
                    {errors.links?.[i]?.url && <p className="text-caption text-danger sm:col-span-3">{errors.links[i].url.message}</p>}
                  </div>
                  <RemoveButton onClick={() => links.remove(i)} />
                </div>
              ))}
              {links.fields.length < 5 && <AddButton onClick={() => links.append({ name: "", url: "", code: "" })}>添加链接</AddButton>}
            </fieldset>
          ) : (
            <Field label="文本内容" error={errors.text?.message} hint="支持 Markdown，适合发送使用说明、群号、激活方法等">
              {({ id, describedBy, invalid }) => (
                <Textarea id={id} rows={6} maxLength={5000} className="font-mono text-body" aria-describedby={describedBy} aria-invalid={invalid} {...register("text")} />
              )}
            </Field>
          )}

          <Field label="交付附言（可选）" error={errors.note?.message} hint="显示在交付内容上方，例如“感谢购买！有问题请邮件联系我”">
            {({ id, describedBy, invalid }) => (
              <Textarea id={id} rows={2} maxLength={500} aria-describedby={describedBy} aria-invalid={invalid} {...register("note")} />
            )}
          </Field>
        </Card>
      </form>

      <Dialog open={issues !== null} onOpenChange={(o) => !o && setIssues(null)}>
        <DialogContent size="md">
          <DialogTitle>还有内容需要完善</DialogTitle>
          <DialogDescription>完成以下项目后才能上架：</DialogDescription>
          <ul className="flex flex-col gap-2">
            {issues?.map((is) => {
              const anchor = fieldAnchor[is.field];
              return (
                <li key={is.field} className="flex items-center justify-between gap-3 rounded-md bg-muted px-3 py-2">
                  <span className="text-text-primary">{is.message}</span>
                  {anchor ? (
                    <Button
                      variant="link"
                      size="sm"
                      onClick={() => {
                        setIssues(null);
                        document.getElementById(anchor)?.scrollIntoView({ behavior: "smooth", block: "center" });
                      }}
                    >
                      去填写
                    </Button>
                  ) : is.field === "account" ? (
                    <span className="shrink-0 text-caption text-text-tertiary">见页面顶部提示</span>
                  ) : (
                    <span className="shrink-0 text-caption text-text-tertiary">收款设置即将开放</span>
                  )}
                </li>
              );
            })}
          </ul>
          <DialogFooter>
            <DialogClose asChild>
              <Button>知道了</Button>
            </DialogClose>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </main>
  );
}

function Card({ title, description, id, children }: { title: string; description?: string; id?: string; children: React.ReactNode }) {
  return (
    <section id={id} className="flex scroll-mt-20 flex-col gap-6 rounded-lg border border-border bg-background p-6">
      <div>
        <h2 className="text-h3">{title}</h2>
        {description && <p className="mt-1 text-caption text-text-tertiary">{description}</p>}
      </div>
      {children}
    </section>
  );
}

function AddButton({ onClick, children }: { onClick: () => void; children: React.ReactNode }) {
  return (
    <Button type="button" variant="outline" size="sm" className="self-start" onClick={onClick}>
      <Plus strokeWidth={1.5} data-icon="inline-start" />
      {children}
    </Button>
  );
}

function RemoveButton({ onClick }: { onClick: () => void }) {
  return (
    <Button type="button" variant="ghost" size="icon" onClick={onClick} aria-label="删除这一项">
      <Trash2 strokeWidth={1.5} />
    </Button>
  );
}
