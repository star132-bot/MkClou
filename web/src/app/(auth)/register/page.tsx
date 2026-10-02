import type { Metadata } from "next";
import Link from "next/link";

import { AuthShell } from "@/components/auth/auth-shell";

import { RegisterForm } from "./register-form";

export const metadata: Metadata = { title: "注册" };

export default function RegisterPage() {
  return (
    <AuthShell
      title="创建你的 MkClou 账号"
      description="免费开店，平台不抽成"
      footer={
        <>
          已有账号？
          <Link href="/login" className="font-medium text-text-primary underline-offset-4 hover:underline">
            登录
          </Link>
        </>
      }
    >
      <RegisterForm />
    </AuthShell>
  );
}
