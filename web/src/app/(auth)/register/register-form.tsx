"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";

import { FormAlert } from "@/components/auth/auth-shell";
import { Field, PasswordInput, PasswordStrengthBar } from "@/components/auth/form-field";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { ApiError, ErrorCode } from "@/lib/api/client";
import { authApi } from "@/lib/auth/api";
import { useAuthStore } from "@/lib/auth/store";
import { emailSchema, passwordSchema, suggestEmail } from "@/lib/auth/validation";

const schema = z
  .object({
    email: emailSchema,
    password: passwordSchema,
    agreeTerms: z.boolean().refine((v) => v, "请先阅读并同意用户协议和隐私政策"),
  })
  .refine((v) => v.password.toLowerCase() !== v.email.toLowerCase(), {
    path: ["password"],
    message: "密码不能与邮箱相同",
  });
type FormValues = z.infer<typeof schema>;

export function RegisterForm() {
  const router = useRouter();
  const signIn = useAuthStore((s) => s.signIn);
  const [formError, setFormError] = useState<React.ReactNode>(null);

  const {
    register,
    handleSubmit,
    setValue,
    setError,
    control,
    trigger,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    mode: "onTouched", // 失焦时校验（PRD 01 4.1）
    defaultValues: { email: "", password: "", agreeTerms: false },
  });

  const [email, password, agreeTerms] = useWatch({ control, name: ["email", "password", "agreeTerms"] });
  const suggestion = suggestEmail(email);

  const onSubmit = async (values: FormValues) => {
    setFormError(null);
    try {
      const r = await authApi.register({ ...values, email: values.email.trim() });
      signIn(r.accessToken, r.merchant);
      // 店铺模块完成后改为跳转创建店铺向导 /onboarding（PRD SHOP-01）
      router.replace("/dashboard");
    } catch (e) {
      if (!(e instanceof ApiError)) throw e;
      if (e.code === ErrorCode.EmailRegistered) {
        setFormError(
          <>
            该邮箱已注册，可以直接
            <Link href="/login" className="font-medium underline underline-offset-4">登录</Link>或
            <Link href="/forgot-password" className="font-medium underline underline-offset-4">找回密码</Link>
          </>,
        );
        return;
      }
      const fields = e.fields;
      if (Object.keys(fields).length > 0) {
        for (const [name, message] of Object.entries(fields)) {
          setError(name as keyof FormValues, { message });
        }
        return;
      }
      setFormError(e.message);
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="flex flex-col gap-5">
      {formError && <FormAlert>{formError}</FormAlert>}

      <Field
        label="邮箱"
        error={errors.email?.message}
        hint={
          suggestion ? (
            <button
              type="button"
              className="text-left text-brand underline-offset-4 hover:underline"
              onClick={() => {
                setValue("email", suggestion);
                void trigger("email");
              }}
            >
              你是不是想输入 {suggestion}？
            </button>
          ) : (
            "用于登录和接收订单通知"
          )
        }
      >
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

      <div className="flex flex-col gap-2">
        <Field label="密码" error={errors.password?.message} hint="至少 8 位，需包含字母和数字">
          {({ id, describedBy, invalid }) => (
            <PasswordInput
              id={id}
              autoComplete="new-password"
              aria-describedby={describedBy}
              aria-invalid={invalid}
              {...register("password")}
            />
          )}
        </Field>
        <PasswordStrengthBar password={password} />
      </div>

      <div className="flex flex-col gap-2">
        {/* Radix 复选框渲染为 button，外层包裹 label 不会成为它的可读名称，需用 htmlFor + aria-labelledby 关联 */}
        <div className="flex items-start gap-2 text-text-secondary">
          <Checkbox
            id="agree-terms"
            checked={agreeTerms}
            onCheckedChange={(v) => setValue("agreeTerms", v === true, { shouldValidate: true })}
            aria-labelledby="agree-terms-label"
            aria-describedby={errors.agreeTerms ? "agree-terms-error" : undefined}
            aria-invalid={!!errors.agreeTerms}
            className="mt-0.5"
          />
          <label htmlFor="agree-terms" id="agree-terms-label">
            我已阅读并同意
            <Link href="/terms" target="_blank" className="text-text-primary underline underline-offset-4">
              用户协议
            </Link>
            和
            <Link href="/privacy" target="_blank" className="text-text-primary underline underline-offset-4">
              隐私政策
            </Link>
          </label>
        </div>
        {errors.agreeTerms && (
          <p id="agree-terms-error" className="text-caption text-danger">
            {errors.agreeTerms.message}
          </p>
        )}
      </div>

      <Button type="submit" size="lg" className="w-full" disabled={isSubmitting}>
        {isSubmitting ? "创建中…" : "创建账号"}
      </Button>
    </form>
  );
}
