import * as React from "react"
import { cn } from "@/lib/utils"

// 设计规范 8.2：高 36px；手机端 16px 字号，避免 iOS 聚焦时自动放大页面
function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "h-9 w-full min-w-0 rounded-md border border-input bg-background px-3 py-1 text-body-lg text-text-primary transition-[border-color,box-shadow] duration-150 outline-none placeholder:text-text-tertiary focus-visible:border-brand focus-visible:ring-3 focus-visible:ring-brand/15 disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-muted disabled:opacity-50 aria-invalid:border-danger aria-invalid:ring-danger/15 md:text-body",
        className
      )}
      {...props}
    />
  )
}

export { Input }
