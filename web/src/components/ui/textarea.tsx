import * as React from "react"
import { cn } from "@/lib/utils"

// 与 Input 保持一致的边框、焦点与错误样式（设计规范 8.2）
function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "min-h-20 w-full min-w-0 resize-y rounded-md border border-input bg-background px-3 py-2 text-body-lg text-text-primary transition-[border-color,box-shadow] duration-150 outline-none placeholder:text-text-tertiary focus-visible:border-brand focus-visible:ring-3 focus-visible:ring-brand/15 disabled:cursor-not-allowed disabled:bg-muted disabled:opacity-50 aria-invalid:border-danger aria-invalid:ring-danger/15 md:text-body",
        className
      )}
      {...props}
    />
  )
}

export { Textarea }
