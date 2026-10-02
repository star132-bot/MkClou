import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";

import { AuthShell } from "@/components/auth/auth-shell";

import { RegisterForm } from "./register-form";

export const metadata: Metadata = { title: "注册" };

export default async function RegisterPage({ searchParams }: PageProps<"/register">) {
  const { redirect } = await searchParams;
  const query = typeof redirect === "string" ? `?redirect=${encodeURIComponent(redirect)}` : "";
  return (
    <AuthShell
      title="创建你的 MkClou 账号"
      description="收藏好作品，也可以免费开店，平台不抽成"
      footer={
        <>
          已有账号？
          <Link href={`/login${query}`} className="font-medium text-text-primary underline-offset-4 hover:underline">
            登录
          </Link>
        </>
      }
    >
      <Suspense>
        <RegisterForm />
      </Suspense>
    </AuthShell>
  );
}
