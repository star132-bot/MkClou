"use client";

import { Monitor } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";

import { Field, PasswordInput } from "@/components/auth/form-field";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api/client";
import { authApi, type SessionView } from "@/lib/auth/api";
import { useAuthStore } from "@/lib/auth/store";
import { passwordSchema } from "@/lib/auth/validation";
import { formatTime } from "@/lib/format";

// 账号设置（AUTH-07）：昵称、修改密码、登录设备
export default function AccountPage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-h2">账号设置</h1>
      <ProfileCard />
      <PasswordCard />
      <SessionsCard />
    </div>
  );
}

function Card({ title, description, children }: { title: string; description?: string; children: React.ReactNode }) {
  return (
    <section className="flex flex-col gap-5 rounded-lg border border-border p-6">
      <div>
        <h2 className="text-h3">{title}</h2>
        {description && <p className="mt-1 text-caption text-text-tertiary">{description}</p>}
      </div>
      {children}
    </section>
  );
}

function ProfileCard() {
  const merchant = useAuthStore((s) => s.merchant);
  const updateMerchant = useAuthStore((s) => s.updateMerchant);
  const [nickname, setNickname] = useState(merchant?.nickname ?? "");
  const [error, setError] = useState<string>();
  const [pending, setPending] = useState(false);

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    const v = nickname.trim();
    if (v.length < 1 || v.length > 20) {
      setError("昵称为 1～20 个字符");
      return;
    }
    setPending(true);
    try {
      await authApi.updateNickname(v);
      updateMerchant({ nickname: v });
      setError(undefined);
      toast.success("昵称已保存");
    } catch (err) {
      setError(err instanceof ApiError ? (err.fields.nickname ?? err.message) : "保存失败，请稍后重试");
    } finally {
      setPending(false);
    }
  };

  return (
    <Card title="基本资料">
      <form onSubmit={save} noValidate className="flex flex-col gap-5">
        <Field label="邮箱" hint={merchant?.emailVerified ? "已验证" : "未验证，验证后才能上架商品"}>
          {({ id, describedBy }) => <Input id={id} value={merchant?.email ?? ""} disabled aria-describedby={describedBy} />}
        </Field>
        <Field label="昵称" error={error}>
          {({ id, describedBy, invalid }) => (
            <Input id={id} value={nickname} onChange={(e) => setNickname(e.target.value)} maxLength={20} aria-describedby={describedBy} aria-invalid={invalid} />
          )}
        </Field>
        <Button type="submit" className="self-start" disabled={pending || nickname.trim() === (merchant?.nickname ?? "")}>
          {pending ? "保存中…" : "保存昵称"}
        </Button>
      </form>
    </Card>
  );
}

function PasswordCard() {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [errors, setErrors] = useState<{ currentPassword?: string; newPassword?: string }>({});
  const [pending, setPending] = useState(false);

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    const check = passwordSchema.safeParse(next);
    if (!current || !check.success) {
      setErrors({ currentPassword: current ? undefined : "请输入当前密码", newPassword: check.success ? undefined : check.error.issues[0].message });
      return;
    }
    setPending(true);
    try {
      await authApi.changePassword(current, next);
      setCurrent("");
      setNext("");
      setErrors({});
      toast.success("密码已修改", { description: "其他设备已退出登录" });
    } catch (err) {
      if (err instanceof ApiError && Object.keys(err.fields).length > 0) setErrors(err.fields);
      else toast.error(err instanceof ApiError ? err.message : "修改失败，请稍后重试");
    } finally {
      setPending(false);
    }
  };

  return (
    <Card title="修改密码" description="修改后，除当前设备外的其他设备都会退出登录。">
      <form onSubmit={save} noValidate className="flex flex-col gap-5">
        <Field label="当前密码" error={errors.currentPassword}>
          {({ id, describedBy, invalid }) => (
            <PasswordInput id={id} value={current} onChange={(e) => setCurrent(e.target.value)} autoComplete="current-password" aria-describedby={describedBy} aria-invalid={invalid} />
          )}
        </Field>
        <Field label="新密码" error={errors.newPassword} hint="至少 8 位，需包含字母和数字">
          {({ id, describedBy, invalid }) => (
            <PasswordInput id={id} value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" aria-describedby={describedBy} aria-invalid={invalid} />
          )}
        </Field>
        <Button type="submit" className="self-start" disabled={pending}>
          {pending ? "修改中…" : "修改密码"}
        </Button>
      </form>
    </Card>
  );
}

/** 从 User-Agent 中提取浏览器与系统的简短描述。 */
function describeAgent(ua: string) {
  const browser = /Edg\//.test(ua) ? "Edge" : /Chrome\//.test(ua) ? "Chrome" : /Firefox\//.test(ua) ? "Firefox" : /Safari\//.test(ua) ? "Safari" : "浏览器";
  const os = /iPhone|iPad/.test(ua) ? "iOS" : /Android/.test(ua) ? "Android" : /Mac OS X/.test(ua) ? "macOS" : /Windows/.test(ua) ? "Windows" : /Linux/.test(ua) ? "Linux" : "";
  return os ? `${browser} · ${os}` : browser;
}

function SessionsCard() {
  const [items, setItems] = useState<SessionView[] | null>(null);
  const [reload, setReload] = useState(0);

  useEffect(() => {
    let cancelled = false;
    authApi
      .sessions()
      .then((r) => !cancelled && setItems(r.items))
      .catch(() => !cancelled && setItems([]));
    return () => {
      cancelled = true;
    };
  }, [reload]);

  const revoke = async (id?: string) => {
    try {
      if (id) await authApi.revokeSession(id);
      else await authApi.revokeOtherSessions();
      toast.success(id ? "已退出该设备" : "已退出其他所有设备");
      setReload((n) => n + 1);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "操作失败，请稍后重试");
    }
  };

  const others = items?.filter((s) => !s.current).length ?? 0;

  return (
    <Card title="登录设备">
      {!items ? (
        <div className="h-16 animate-pulse rounded-md bg-muted" aria-busy="true" />
      ) : (
        <ul className="flex flex-col divide-y divide-border">
          {items.map((s) => (
            <li key={s.id} className="flex items-center gap-3 py-3">
              <Monitor className="size-5 shrink-0 text-text-tertiary" strokeWidth={1.5} aria-hidden />
              <div className="min-w-0 flex-1">
                <p className="font-medium text-text-primary">
                  {describeAgent(s.userAgent)}
                  {s.current && <span className="ml-2 rounded-sm bg-success/10 px-1.5 text-caption text-success">当前设备</span>}
                </p>
                <p className="text-caption text-text-tertiary">
                  {s.ip} · 最近活动 {formatTime(s.lastActiveAt)}
                </p>
              </div>
              {!s.current && (
                <Button variant="ghost" size="sm" onClick={() => revoke(s.id)}>
                  退出
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}
      {others > 0 && (
        <Button variant="outline" size="sm" className="self-start" onClick={() => revoke()}>
          退出其他所有设备
        </Button>
      )}
    </Card>
  );
}
