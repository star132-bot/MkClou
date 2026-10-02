"use client";

import { Eye, EyeOff, RefreshCw } from "lucide-react";
import { forwardRef, useEffect, useId, useState } from "react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { authApi } from "@/lib/auth/api";
import { type PasswordStrength, passwordStrength } from "@/lib/auth/validation";
import { cn } from "@/lib/utils";

interface FieldProps {
  label: string;
  error?: string;
  hint?: React.ReactNode;
  /** 标签右侧的附加内容，如“忘记密码？” */
  aside?: React.ReactNode;
  children: (ids: { id: string; describedBy?: string; invalid: boolean }) => React.ReactNode;
}

/** 表单字段：标签在上，错误在下（12px 红字），辅助说明在下（设计规范 8.2）。 */
export function Field({ label, error, hint, aside, children }: FieldProps) {
  const id = useId();
  const msgId = `${id}-msg`;
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <Label htmlFor={id}>{label}</Label>
        {aside}
      </div>
      {children({ id, describedBy: error || hint ? msgId : undefined, invalid: !!error })}
      {error ? (
        <p id={msgId} className="text-caption text-danger">
          {error}
        </p>
      ) : (
        hint && (
          <p id={msgId} className="text-caption text-text-tertiary">
            {hint}
          </p>
        )
      )}
    </div>
  );
}

type InputProps = React.ComponentProps<typeof Input>;

/** 密码输入框：右侧显示 / 隐藏切换。 */
export const PasswordInput = forwardRef<HTMLInputElement, InputProps>(function PasswordInput(props, ref) {
  const [visible, setVisible] = useState(false);
  return (
    <div className="relative">
      <Input ref={ref} type={visible ? "text" : "password"} className="pr-10" {...props} />
      <button
        type="button"
        onClick={() => setVisible((v) => !v)}
        className="absolute inset-y-0 right-0 flex w-10 items-center justify-center rounded-r-md text-text-tertiary transition-colors hover:text-text-primary focus-visible:text-text-primary focus-visible:outline-none"
        aria-label={visible ? "隐藏密码" : "显示密码"}
        aria-pressed={visible}
      >
        {visible ? <EyeOff className="size-4" strokeWidth={1.5} /> : <Eye className="size-4" strokeWidth={1.5} />}
      </button>
    </div>
  );
});

const strengthMeta: Record<Exclude<PasswordStrength, 0>, { label: string; color: string }> = {
  1: { label: "弱", color: "bg-danger" },
  2: { label: "中", color: "bg-warning" },
  3: { label: "强", color: "bg-success" },
};

/** 密码强度条（PRD AUTH-01）。 */
export function PasswordStrengthBar({ password }: { password: string }) {
  const level = passwordStrength(password);
  if (level === 0) return null;
  const meta = strengthMeta[level];
  return (
    <div className="flex items-center gap-3" aria-live="polite">
      <div className="flex flex-1 gap-1">
        {[1, 2, 3].map((i) => (
          <span
            key={i}
            className={cn("h-1 flex-1 rounded-full transition-colors duration-150", i <= level ? meta.color : "bg-muted")}
          />
        ))}
      </div>
      <span className="w-12 text-right text-caption text-text-tertiary">强度：{meta.label}</span>
    </div>
  );
}

/** 图形验证码：点击图片刷新。连续登录失败 3 次后出现（PRD AUTH-08）。 */
export function CaptchaField({
  value,
  onChange,
  onIdChange,
  error,
  refreshKey,
}: {
  value: string;
  onChange: (v: string) => void;
  onIdChange: (id: string) => void;
  error?: string;
  /** 变化时重新获取验证码（验证码校验后即失效） */
  refreshKey: number;
}) {
  const [image, setImage] = useState<string | null>(null);
  const [manualRefresh, setManualRefresh] = useState(0);

  useEffect(() => {
    let cancelled = false;
    authApi
      .captcha()
      .then((c) => {
        if (cancelled) return;
        setImage(c.image);
        onIdChange(c.captchaId);
        onChange("");
      })
      .catch(() => !cancelled && setImage(null));
    return () => {
      cancelled = true;
    };
  }, [onChange, onIdChange, refreshKey, manualRefresh]);

  return (
    <Field label="图形验证码" error={error}>
      {({ id, describedBy, invalid }) => (
        <div className="flex gap-3">
          <Input
            id={id}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            autoComplete="off"
            inputMode="text"
            maxLength={8}
            aria-describedby={describedBy}
            aria-invalid={invalid}
            className="flex-1"
          />
          <button
            type="button"
            onClick={() => setManualRefresh((n) => n + 1)}
            className="relative h-9 w-[105px] shrink-0 overflow-hidden rounded-md border border-border bg-muted"
            aria-label="看不清，换一张"
            title="看不清，换一张"
          >
            {image ? (
              // eslint-disable-next-line @next/next/no-img-element -- data URI 验证码图片，无需 next/image 优化
              <img src={image} alt="图形验证码" className="size-full object-cover" />
            ) : (
              <RefreshCw className="mx-auto size-4 text-text-tertiary" strokeWidth={1.5} />
            )}
          </button>
        </div>
      )}
    </Field>
  );
}
