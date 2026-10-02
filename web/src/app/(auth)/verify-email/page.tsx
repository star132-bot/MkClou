"use client";

import { CircleCheck, CircleX, LoaderCircle } from "lucide-react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef, useState } from "react";

import { AuthShell } from "@/components/auth/auth-shell";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api/client";
import { authApi } from "@/lib/auth/api";
import { useAuthStore } from "@/lib/auth/store";

export default function VerifyEmailPage() {
  return (
    <Suspense>
      <VerifyEmail />
    </Suspense>
  );
}

type State = { kind: "loading" } | { kind: "success" | "already" } | { kind: "invalid"; message: string };

function VerifyEmail() {
  const token = useSearchParams().get("token") ?? "";
  const [state, setState] = useState<State>(token ? { kind: "loading" } : { kind: "invalid", message: "链接不完整" });
  const started = useRef(false);

  useEffect(() => {
    // 验证令牌是一次性的：开发模式下 React 会执行两次 effect，用 ref 保证只提交一次
    if (!token || started.current) return;
    started.current = true;
    authApi
      .verifyEmail(token)
      .then((r) => {
        setState({ kind: r.alreadyVerified ? "already" : "success" });
        useAuthStore.getState().updateMerchant({ emailVerified: true });
      })
      .catch((e: unknown) =>
        setState({ kind: "invalid", message: e instanceof ApiError ? e.message : "验证失败，请稍后重试" }),
      );
  }, [token]);

  if (state.kind === "loading") {
    return (
      <AuthShell title="正在验证邮箱">
        <LoaderCircle className="size-6 animate-spin text-text-tertiary" strokeWidth={1.5} aria-label="验证中" />
      </AuthShell>
    );
  }

  if (state.kind === "invalid") {
    return (
      <AuthShell title="验证失败" description={state.message}>
        <div className="flex flex-col gap-5">
          <CircleX className="size-6 text-danger" strokeWidth={1.5} aria-hidden />
          <p>登录后，可以在后台顶部的提示条中重新发送验证邮件。</p>
          <Button size="lg" className="w-full" asChild>
            <Link href="/dashboard">进入卖家中心</Link>
          </Button>
        </div>
      </AuthShell>
    );
  }

  return (
    <AuthShell title={state.kind === "success" ? "邮箱验证成功" : "邮箱已验证"}>
      <div className="flex flex-col gap-5">
        <CircleCheck className="size-6 text-success" strokeWidth={1.5} aria-hidden />
        <p>现在可以上架商品了。</p>
        <Button size="lg" className="w-full" asChild>
          <Link href="/dashboard">进入卖家中心</Link>
        </Button>
      </div>
    </AuthShell>
  );
}
