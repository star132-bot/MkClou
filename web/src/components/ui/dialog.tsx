"use client"

import * as React from "react"
import { XIcon } from "lucide-react"
import { Dialog as DialogPrimitive } from "radix-ui"
import { cn } from "@/lib/utils"

// 设计规范 8.7：遮罩 rgba(0,0,0,0.4) + 模糊，宽度 sm 400 / md 560，radius-lg、shadow-lg

const Dialog = DialogPrimitive.Root
const DialogTrigger = DialogPrimitive.Trigger
const DialogClose = DialogPrimitive.Close

function DialogContent({
  className,
  children,
  size = "sm",
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Content> & { size?: "sm" | "md" }) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/40 backdrop-blur-sm data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=closed]:animate-out data-[state=closed]:fade-out-0" />
      <DialogPrimitive.Content
        data-slot="dialog-content"
        className={cn(
          "fixed top-1/2 left-1/2 z-50 flex max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 flex-col gap-4 overflow-y-auto rounded-lg border border-border bg-background p-6 shadow-lg outline-none data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95",
          size === "sm" ? "max-w-[400px]" : "max-w-[560px]",
          className
        )}
        {...props}
      >
        {children}
        <DialogPrimitive.Close
          className="absolute top-4 right-4 rounded-sm p-1 text-text-tertiary transition-colors hover:bg-muted hover:text-text-primary focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
          aria-label="关闭"
        >
          <XIcon className="size-4" strokeWidth={1.5} />
        </DialogPrimitive.Close>
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  )
}

function DialogTitle({ className, ...props }: React.ComponentProps<typeof DialogPrimitive.Title>) {
  return <DialogPrimitive.Title className={cn("pr-6 text-h3 text-text-primary", className)} {...props} />
}

function DialogDescription({ className, ...props }: React.ComponentProps<typeof DialogPrimitive.Description>) {
  return <DialogPrimitive.Description className={cn("text-text-secondary", className)} {...props} />
}

function DialogFooter({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("mt-2 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end", className)} {...props} />
}

export { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogTitle, DialogTrigger }
