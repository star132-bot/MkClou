import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";

import { AuthShell } from "@/components/auth/auth-shell";

import { LoginForm } from "./login-form";

export const metadata: Metadata = { title: "登录" };

export default function LoginPage() {
  return (
    <AuthShell
      title="登录 MkClou"
      description="管理你的店铺、商品和订单"
      footer={
        <>
          还没有账号？
          <Link href="/register" className="font-medium text-text-primary underline-offset-4 hover:underline">
            免费注册
          </Link>
        </>
      }
    >
      <Suspense>
        <LoginForm />
      </Suspense>
    </AuthShell>
  );
}
