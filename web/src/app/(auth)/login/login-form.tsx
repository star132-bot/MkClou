"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";

import { FormAlert } from "@/components/auth/auth-shell";
import { CaptchaField, Field, PasswordInput } from "@/components/auth/form-field";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiError, ErrorCode, refreshSession } from "@/lib/api/client";
import { authApi } from "@/lib/auth/api";
import { useAuthStore } from "@/lib/auth/store";
import { emailSchema, safeRedirect } from "@/lib/auth/validation";

const schema = z.object({
  email: emailSchema,
  password: z.string().min(1, "请输入密码"),
  remember: z.boolean(),
});
type FormValues = z.infer<typeof schema>;

export function LoginForm() {
  const router = useRouter();
  const params = useSearchParams();
  const redirectTo = safeRedirect(params.get("redirect"));
  const signIn = useAuthStore((s) => s.signIn);

  const [formError, setFormError] = useState<string | null>(null);
  const [captchaRequired, setCaptchaRequired] = useState(false);
  const [captchaId, setCaptchaId] = useState("");
  const [captchaCode, setCaptchaCode] = useState("");
  const [captchaError, setCaptchaError] = useState<string>();
  const [captchaRefresh, setCaptchaRefresh] = useState(0);
  const [lockedFor, setLockedFor] = useState(0); // 剩余锁定秒数

  const {
    register,
    handleSubmit,
    setValue,
    control,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { email: "", password: "", remember: false } });

  const remember = useWatch({ control, name: "remember" });

  // 已登录（Cookie 中有有效会话）时直接进入后台
  useEffect(() => {
    if (useAuthStore.getState().status !== "unknown") return;
    void refreshSession().then((ok) => ok && router.replace(redirectTo));
  }, [router, redirectTo]);

  // 锁定倒计时
  useEffect(() => {
    if (lockedFor <= 0) return;
    const t = setInterval(() => setLockedFor((s) => Math.max(0, s - 1)), 1000);
    return () => clearInterval(t);
  }, [lockedFor]);

  const onSubmit = async (values: FormValues) => {
    setFormError(null);
    setCaptchaError(undefined);
    if (captchaRequired && !captchaCode.trim()) {
      setCaptchaError("请输入图形验证码");
      return;
    }
    try {
      const r = await authApi.login({
        ...values,
        ...(captchaRequired ? { captchaId, captchaCode } : {}),
      });
      signIn(r.accessToken, r.merchant);
      router.replace(redirectTo);
    } catch (e) {
      if (!(e instanceof ApiError)) throw e;
      const data = (e.data ?? {}) as { captchaRequired?: boolean; retryAfter?: number };

      // 验证码每次校验后都会失效，出错后重新获取
      if (captchaRequired) setCaptchaRefresh((n) => n + 1);

      switch (e.code) {
        case ErrorCode.CaptchaRequired:
          setCaptchaRequired(true);
          if (captchaRequired) setCaptchaError(e.message);
          else setFormError("为了你的账号安全，请输入图形验证码");
          break;
        case ErrorCode.AccountLocked:
          setLockedFor(data.retryAfter ?? 900);
          setFormError(null);
          break;
        case ErrorCode.BadCredentials:
          if (data.captchaRequired) setCaptchaRequired(true);
          setFormError("邮箱或密码错误");
          break;
        default:
          setFormError(e.message);
      }
    }
  };

  const locked = lockedFor > 0;

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="flex flex-col gap-5">
      {locked && (
        <FormAlert>
          登录尝试过多，请 {Math.ceil(lockedFor / 60)} 分钟后再试，或
          <Link href="/forgot-password" className="font-medium underline underline-offset-4">
            找回密码
          </Link>
        </FormAlert>
      )}
      {formError && !locked && <FormAlert>{formError}</FormAlert>}

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

      <Field
        label="密码"
        error={errors.password?.message}
        aside={
          <Link href="/forgot-password" className="text-caption text-text-secondary underline-offset-4 hover:text-text-primary hover:underline">
            忘记密码？
          </Link>
        }
      >
        {({ id, describedBy, invalid }) => (
          <PasswordInput
            id={id}
            autoComplete="current-password"
            aria-describedby={describedBy}
            aria-invalid={invalid}
            {...register("password")}
          />
        )}
      </Field>

      {captchaRequired && (
        <CaptchaField
          value={captchaCode}
          onChange={setCaptchaCode}
          onIdChange={setCaptchaId}
          error={captchaError}
          refreshKey={captchaRefresh}
        />
      )}

      <div className="flex items-center gap-2">
        <Checkbox
          id="remember"
          checked={remember}
          onCheckedChange={(v) => setValue("remember", v === true)}
        />
        <Label htmlFor="remember" className="font-normal text-text-secondary">
          30 天内保持登录
        </Label>
      </div>

      <Button type="submit" size="lg" className="w-full" disabled={isSubmitting || locked}>
        {isSubmitting ? "登录中…" : "登录"}
      </Button>
    </form>
  );
}
