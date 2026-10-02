"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { MailCheck } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { AuthShell, FormAlert } from "@/components/auth/auth-shell";
import { Field } from "@/components/auth/form-field";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api/client";
import { authApi } from "@/lib/auth/api";
import { emailSchema } from "@/lib/auth/validation";

const schema = z.object({ email: emailSchema });
type FormValues = z.infer<typeof schema>;

export default function ForgotPasswordPage() {
  const [sentTo, setSentTo] = useState<string | null>(null);
  const [cooldown, setCooldown] = useState(0);
  const [formError, setFormError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    getValues,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { email: "" } });

  useEffect(() => {
    if (cooldown <= 0) return;
    const t = setInterval(() => setCooldown((s) => Math.max(0, s - 1)), 1000);
    return () => clearInterval(t);
  }, [cooldown]);

  const send = async ({ email }: FormValues) => {
    setFormError(null);
    try {
      await authApi.forgotPassword(email.trim());
      setSentTo(email.trim());
      setCooldown(60);
    } catch (e) {
      if (!(e instanceof ApiError)) throw e;
      const retry = (e.data as { retryAfter?: number } | null)?.retryAfter;
      if (retry) setCooldown(retry);
      setFormError(e.message);
    }
  };

  if (sentTo) {
    return (
      <AuthShell title="请查收邮件">
        <div className="flex flex-col items-start gap-5">
          <MailCheck className="size-6 text-text-primary" strokeWidth={1.5} aria-hidden />
          {/* 无论邮箱是否注册都显示相同提示，防止邮箱枚举（PRD SEC-03） */}
          <p>
            如果 <span className="font-medium text-text-primary">{sentTo}</span> 已注册，你将收到一封重置密码的邮件，链接 30
            分钟内有效。
          </p>
          <p className="text-text-tertiary">没有收到？请检查垃圾邮件文件夹。</p>
          {formError && <FormAlert>{formError}</FormAlert>}
          <div className="flex w-full flex-col gap-3">
            <Button variant="outline" className="w-full" disabled={cooldown > 0 || isSubmitting} onClick={() => send(getValues())}>
              {cooldown > 0 ? `重新发送（${cooldown} 秒）` : "重新发送"}
            </Button>
            <Button variant="ghost" className="w-full" asChild>
              <Link href="/login">返回登录</Link>
            </Button>
          </div>
        </div>
      </AuthShell>
    );
  }

  return (
    <AuthShell
      title="找回密码"
      description="输入注册时使用的邮箱，我们会发送重置密码的链接。"
      footer={
        <Link href="/login" className="font-medium text-text-primary underline-offset-4 hover:underline">
          返回登录
        </Link>
      }
    >
      <form onSubmit={handleSubmit(send)} noValidate className="flex flex-col gap-5">
        {formError && <FormAlert>{formError}</FormAlert>}
        <Field label="邮箱" error={errors.email?.message}>
          {({ id, describedBy, invalid }) => (
            <Input
              id={id}
              type="email"
              autoComplete="email"
              inputMode="email"
              autoFocus
              aria-describedby={describedBy}
              aria-invalid={invalid}
              {...register("email")}
            />
          )}
        </Field>
        <Button type="submit" size="lg" className="w-full" disabled={isSubmitting || cooldown > 0}>
          {isSubmitting ? "发送中…" : cooldown > 0 ? `请 ${cooldown} 秒后再试` : "发送重置链接"}
        </Button>
      </form>
    </AuthShell>
  );
}
