import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";

import { AuthShell } from "@/components/auth/auth-shell";

import { LoginForm } from "./login-form";

export const metadata: Metadata = { title: "登录" };

export default async function LoginPage({ searchParams }: PageProps<"/login">) {
  const { redirect } = await searchParams;
  const query = typeof redirect === "string" ? `?redirect=${encodeURIComponent(redirect)}` : "";
  return (
    <AuthShell
      title="登录 MkClou"
      description="收藏喜欢的作品，或管理你的店铺"
      footer={
        <>
          还没有账号？
          <Link href={`/register${query}`} className="font-medium text-text-primary underline-offset-4 hover:underline">
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
