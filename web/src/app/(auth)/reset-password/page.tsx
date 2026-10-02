"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { CircleCheck } from "lucide-react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";

import { AuthShell, FormAlert } from "@/components/auth/auth-shell";
import { Field, PasswordInput, PasswordStrengthBar } from "@/components/auth/form-field";
import { Button } from "@/components/ui/button";
import { ApiError, ErrorCode } from "@/lib/api/client";
import { authApi } from "@/lib/auth/api";
import { passwordSchema } from "@/lib/auth/validation";

const schema = z
  .object({ password: passwordSchema, confirm: z.string() })
  .refine((v) => v.password === v.confirm, { path: ["confirm"], message: "两次输入的密码不一致" });
type FormValues = z.infer<typeof schema>;

export default function ResetPasswordPage() {
  return (
    <Suspense>
      <ResetPassword />
    </Suspense>
  );
}

function ResetPassword() {
  const token = useSearchParams().get("token") ?? "";
  const [state, setState] = useState<"form" | "done" | "invalid">(token ? "form" : "invalid");
  const [formError, setFormError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    setError,
    control,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema), mode: "onTouched", defaultValues: { password: "", confirm: "" } });

  const password = useWatch({ control, name: "password" });

  const onSubmit = async ({ password }: FormValues) => {
    setFormError(null);
    try {
      await authApi.resetPassword(token, password);
      setState("done");
    } catch (e) {
      if (!(e instanceof ApiError)) throw e;
      if (e.code === ErrorCode.TokenInvalid) setState("invalid");
      else if (e.fields.password) setError("password", { message: e.fields.password });
      else setFormError(e.message);
    }
  };

  if (state === "invalid") {
    return (
      <AuthShell title="链接已失效" description="重置链接已过期或已被使用，请重新获取。">
        <Button size="lg" className="w-full" asChild>
          <Link href="/forgot-password">重新获取重置链接</Link>
        </Button>
      </AuthShell>
    );
  }

  if (state === "done") {
    return (
      <AuthShell title="密码已重置">
        <div className="flex flex-col items-start gap-5">
          <CircleCheck className="size-6 text-success" strokeWidth={1.5} aria-hidden />
          <p>为了你的账号安全，所有设备都已退出登录，请使用新密码重新登录。</p>
          <Button size="lg" className="w-full" asChild>
            <Link href="/login">去登录</Link>
          </Button>
        </div>
      </AuthShell>
    );
  }

  return (
    <AuthShell title="设置新密码" description="设置后，所有已登录的设备都会退出。">
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="flex flex-col gap-5">
        {formError && <FormAlert>{formError}</FormAlert>}
        <div className="flex flex-col gap-2">
          <Field label="新密码" error={errors.password?.message} hint="至少 8 位，需包含字母和数字">
            {({ id, describedBy, invalid }) => (
              <PasswordInput
                id={id}
                autoComplete="new-password"
                autoFocus
                aria-describedby={describedBy}
                aria-invalid={invalid}
                {...register("password")}
              />
            )}
          </Field>
          <PasswordStrengthBar password={password} />
        </div>
        <Field label="确认新密码" error={errors.confirm?.message}>
          {({ id, describedBy, invalid }) => (
            <PasswordInput
              id={id}
              autoComplete="new-password"
              aria-describedby={describedBy}
              aria-invalid={invalid}
              {...register("confirm")}
            />
          )}
        </Field>
        <Button type="submit" size="lg" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? "保存中…" : "保存新密码"}
        </Button>
      </form>
    </AuthShell>
  );
}
