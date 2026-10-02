"use client";

import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useId, useState } from "react";
import { RadioGroup } from "radix-ui";

import { Field } from "@/components/auth/form-field";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiError } from "@/lib/api/client";
import { productApi } from "@/lib/product/api";
import { DELIVERY_TYPES, yuanToCents } from "@/lib/product/meta";
import type { DeliveryType } from "@/lib/storefront/types";
import { cn } from "@/lib/utils";

/** 新建商品：先确定名称、交付类型和价格，创建草稿后进入编辑页（PRD-02）。 */
export function CreateProductDialog({ trigger }: { trigger?: React.ReactNode }) {
  const router = useRouter();
  const ids = useId();
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [type, setType] = useState<DeliveryType>("LINK");
  const [price, setPrice] = useState("");
  const [errors, setErrors] = useState<{ name?: string; price?: string; form?: string }>({});
  const [pending, setPending] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    const next: typeof errors = {};
    const n = name.trim();
    if (n.length < 2 || n.length > 60) next.name = "商品名称为 2～60 个字符";
    const cents = yuanToCents(price);
    if (cents === null || cents > 5_000_000) next.price = "请输入 0～50000 之间的价格，最多两位小数";
    setErrors(next);
    if (next.name || next.price) return;

    setPending(true);
    try {
      const p = await productApi.create({ name: n, deliveryType: type, price: cents! });
      setOpen(false);
      router.push(`/dashboard/products/${p.publicId}`);
    } catch (err) {
      setPending(false);
      if (err instanceof ApiError) {
        const f = err.fields;
        setErrors({ name: f.name, price: f.price, form: f.name || f.price ? undefined : err.message });
      } else {
        setErrors({ form: "创建失败，请稍后重试" });
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus strokeWidth={1.5} data-icon="inline-start" />
            新建商品
          </Button>
        )}
      </DialogTrigger>
      <DialogContent size="md">
        <DialogTitle>新建商品</DialogTitle>
        <DialogDescription>先填写基本信息，创建草稿后可以继续完善图片、介绍和交付内容。</DialogDescription>
        <form onSubmit={submit} noValidate className="mt-2 flex flex-col gap-5">
          {errors.form && <p className="rounded-md bg-danger/10 px-3 py-2 text-danger">{errors.form}</p>}
          <Field label="商品名称" error={errors.name}>
            {({ id, describedBy, invalid }) => (
              <Input id={id} value={name} onChange={(e) => setName(e.target.value)} maxLength={60} autoFocus aria-describedby={describedBy} aria-invalid={invalid} />
            )}
          </Field>
          <div className="flex flex-col gap-2">
            <Label id={`${ids}-type`}>交付类型</Label>
            <RadioGroup.Root
              aria-labelledby={`${ids}-type`}
              value={type}
              onValueChange={(v) => setType(v as DeliveryType)}
              className="grid grid-cols-2 gap-2"
            >
              {DELIVERY_TYPES.map((d) => (
                <RadioGroup.Item
                  key={d.value}
                  value={d.value}
                  disabled={!d.available}
                  aria-label={`${d.label}：${d.available ? d.desc : "即将支持"}`}
                  className={cn(
                    "flex flex-col items-start gap-1 rounded-lg border border-border p-3 text-left transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring data-[state=checked]:border-brand data-[state=checked]:ring-1 data-[state=checked]:ring-brand",
                    d.available ? "hover:border-text-tertiary" : "cursor-not-allowed opacity-50",
                  )}
                >
                  <span className="flex items-center gap-2 font-medium text-text-primary">
                    {d.label}
                    {!d.available && <span className="rounded-sm bg-muted px-1.5 text-caption font-normal text-text-tertiary">即将支持</span>}
                  </span>
                  <span className="text-caption text-text-tertiary">{d.desc}</span>
                </RadioGroup.Item>
              ))}
            </RadioGroup.Root>
            <p className="text-caption text-text-tertiary">商品上架后不能修改交付类型。</p>
          </div>
          <Field label="价格（元）" error={errors.price} hint="填 0 表示免费商品，买家填写邮箱即可领取">
            {({ id, describedBy, invalid }) => (
              <Input id={id} value={price} onChange={(e) => setPrice(e.target.value)} inputMode="decimal" placeholder="29.90" aria-describedby={describedBy} aria-invalid={invalid} />
            )}
          </Field>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              取消
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? "创建中…" : "创建草稿"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
