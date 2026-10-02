"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";

import { ApiError } from "@/lib/api/client";
import { authApi } from "@/lib/auth/api";
import { useAuthStore } from "@/lib/auth/store";

/** 邮箱脱敏：seller@qq.com → s***@qq.com（PRD 01 4.3） */
function maskEmail(email: string) {
  const at = email.indexOf("@");
  return at <= 1 ? email : `${email[0]}***${email.slice(at)}`;
}

/** 未验证邮箱的商家在后台顶部看到的提示条，支持重新发送（60 秒冷却）。 */
export function VerifyEmailBanner() {
  const merchant = useAuthStore((s) => s.merchant);
  const [cooldown, setCooldown] = useState(0);
  const [sending, setSending] = useState(false);

  useEffect(() => {
    if (cooldown <= 0) return;
    const t = setInterval(() => setCooldown((s) => Math.max(0, s - 1)), 1000);
    return () => clearInterval(t);
  }, [cooldown]);

  if (!merchant || merchant.emailVerified) return null;

  const resend = async () => {
    setSending(true);
    try {
      await authApi.resendVerification();
      toast.success("验证邮件已发送", { description: "请检查收件箱和垃圾邮件文件夹" });
      setCooldown(60);
    } catch (e) {
      const retry = e instanceof ApiError ? (e.data as { retryAfter?: number } | null)?.retryAfter : undefined;
      if (retry) setCooldown(retry);
      toast.error(e instanceof ApiError ? e.message : "发送失败，请稍后重试");
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="border-b border-warning/20 bg-warning/10 px-6 py-2.5 text-warning" role="status">
      验证邮箱后才能上架商品。验证邮件已发送至 {maskEmail(merchant.email)}
      <button
        type="button"
        onClick={resend}
        disabled={sending || cooldown > 0}
        className="ml-3 font-medium underline underline-offset-4 disabled:no-underline disabled:opacity-60"
      >
        {cooldown > 0 ? `重新发送（${cooldown} 秒）` : "重新发送"}
      </button>
    </div>
  );
}
