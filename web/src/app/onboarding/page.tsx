"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import { FormAlert } from "@/components/auth/auth-shell";
import { Field } from "@/components/auth/form-field";
import { SlugField, useSlugCheck } from "@/components/dashboard/slug-field";
import { ShopAvatar } from "@/components/storefront/shop-chrome";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiError, ErrorCode } from "@/lib/api/client";
import { shopApi } from "@/lib/shop/api";
import { slugFromName } from "@/lib/shop/slug";
import { useShopStore } from "@/lib/shop/store";
import { shopNameSchema, slugFormatError } from "@/lib/shop/validation";

type Phase = "loading" | "form" | "error";

// 创建店铺向导（PRD SHOP-01）。头像上传在图片上传接口（#28）完成后加入，未上传时使用店铺名首字作为头像。
export default function OnboardingPage() {
  const router = useRouter();
  const setShop = useShopStore((s) => s.setShop);
  const [phase, setPhase] = useState<Phase>("loading");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    shopApi
      .get()
      .then((s) => {
        if (cancelled) return;
        setShop(s);
        router.replace("/dashboard"); // 已有店铺，直接进入后台
      })
      .catch((e) => {
        if (!cancelled) setPhase(e instanceof ApiError && e.code === ErrorCode.ShopNotCreated ? "form" : "error");
      });
    return () => {
      cancelled = true;
    };
  }, [attempt, router, setShop]);

  const retry = () => {
    setPhase("loading");
    setAttempt((n) => n + 1);
  };

  return (
    <div className="flex min-h-full flex-1 flex-col bg-surface">
      <header className="px-6 py-5">
        <span className="text-h3 font-semibold tracking-tight text-text-primary">MkClou</span>
      </header>
      <main className="flex flex-1 justify-center px-4 pt-4 pb-16 md:pt-12">
        {phase === "loading" && (
          <div className="flex w-full max-w-[880px] flex-col gap-4" aria-busy="true">
            <span className="sr-only">正在加载</span>
            <div className="h-8 w-56 animate-pulse rounded-md bg-muted" />
            <div className="h-72 animate-pulse rounded-lg bg-muted" />
          </div>
        )}
        {phase === "error" && (
          <div className="flex max-w-[400px] flex-col items-center gap-4 pt-16 text-center">
            <h1 className="text-h3">加载失败</h1>
            <p>可能是网络不稳定，请稍后重试。</p>
            <Button variant="outline" onClick={retry}>
              重新加载
            </Button>
          </div>
        )}
        {phase === "form" && <CreateShopForm />}
      </main>
    </div>
  );
}

function CreateShopForm() {
  const router = useRouter();
  const setShop = useShopStore((s) => s.setShop);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [errors, setErrors] = useState<{ name?: string; slug?: string }>({});
  const [serverSuggestions, setServerSuggestions] = useState<string[]>([]);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  // 商家手动修改过链接后，不再根据店铺名自动生成
  const slugEdited = useRef(false);
  const status = useSlugCheck(slug);

  useEffect(() => {
    if (slugEdited.current) return;
    let cancelled = false;
    const t = setTimeout(async () => {
      const s = await slugFromName(name);
      if (!cancelled && !slugEdited.current) setSlug(s);
    }, 300);
    return () => {
      cancelled = true;
      clearTimeout(t);
    };
  }, [name]);

  const validate = () => {
    const next: typeof errors = {};
    const n = shopNameSchema.safeParse(name);
    if (!n.success) next.name = n.error.issues[0].message;
    const s = slugFormatError(slug);
    if (s) next.slug = slug ? s : "请输入店铺链接";
    setErrors(next);
    return !next.name && !next.slug;
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    if (!validate() || status.state === "unavailable") return;
    setSubmitting(true);
    try {
      const shop = await shopApi.create({ name: name.trim(), slug });
      setShop(shop);
      toast.success("店铺创建成功", { description: "接下来完成新手清单，开始售卖吧" });
      router.replace("/dashboard");
    } catch (err) {
      setSubmitting(false);
      if (!(err instanceof ApiError)) {
        setFormError("创建失败，请稍后重试");
        return;
      }
      if (err.code === ErrorCode.StateConflict) {
        router.replace("/dashboard");
        return;
      }
      const f = err.fields;
      if (f.name || f.slug) setErrors({ name: f.name, slug: f.slug });
      else setFormError(err.message);
      const d = err.data as { suggestions?: string[] } | null;
      setServerSuggestions(d?.suggestions ?? []);
    }
  };

  const displayName = name.trim() || "我的店铺";

  return (
    <div className="grid w-full max-w-[880px] gap-8 md:grid-cols-[minmax(0,1fr)_320px]">
      <div className="rounded-lg border border-border bg-background p-6 sm:p-8">
        <h1 className="text-h2">创建你的店铺</h1>
        <p className="mt-2">起个名字，选一个好记的链接。这些之后都可以在店铺设置中修改。</p>

        <form onSubmit={submit} noValidate className="mt-8 flex flex-col gap-6">
          {formError && <FormAlert>{formError}</FormAlert>}
          <Field label="店铺名称" error={errors.name} hint="2～20 个字符，例如“阿杰的工具铺”">
            {({ id, describedBy, invalid }) => (
              <Input
                id={id}
                value={name}
                onChange={(e) => setName(e.target.value)}
                onBlur={() => name && setErrors((x) => ({ ...x, name: shopNameSchema.safeParse(name).error?.issues[0].message }))}
                maxLength={20}
                autoFocus
                aria-describedby={describedBy}
                aria-invalid={invalid}
              />
            )}
          </Field>
          <SlugField
            value={slug}
            onChange={(v) => {
              slugEdited.current = true;
              setSlug(v);
              setErrors((x) => ({ ...x, slug: undefined }));
              setServerSuggestions([]);
            }}
            onBlur={() => slug && setErrors((x) => ({ ...x, slug: slugFormatError(slug) ?? undefined }))}
            status={status}
            error={errors.slug}
            suggestions={serverSuggestions}
            hint="3～20 个字符，只能使用小写字母、数字和连字符"
          />
          <Button type="submit" size="lg" disabled={submitting || status.state === "checking"} className="mt-2">
            {submitting ? "创建中…" : "创建店铺"}
          </Button>
        </form>
      </div>

      {/* 桌面端：店铺页缩略预览 */}
      <aside className="hidden md:block" aria-label="店铺预览">
        <div className="sticky top-8 flex flex-col gap-3">
          <span className="text-caption text-text-tertiary">买家看到的样子</span>
          <div className="overflow-hidden rounded-lg border border-border bg-background">
            <div className="flex flex-col items-center gap-3 px-6 pt-10 pb-8 text-center">
              <ShopAvatar shop={{ name: displayName, avatarUrl: null }} size="lg" />
              <p className="text-h3 break-all text-text-primary">{displayName}</p>
              <p className="text-caption break-all text-text-tertiary">/s/{slug || "your-shop"}</p>
            </div>
            <div className="grid grid-cols-2 gap-3 border-t border-border p-4">
              {[0, 1, 2, 3].map((i) => (
                <div key={i} className="flex flex-col gap-2">
                  <div className="aspect-[4/3] rounded-md bg-muted" />
                  <div className="h-2.5 w-3/4 rounded-sm bg-muted" />
                </div>
              ))}
            </div>
          </div>
        </div>
      </aside>
    </div>
  );
}
