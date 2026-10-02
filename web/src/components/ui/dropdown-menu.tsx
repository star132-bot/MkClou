"use client"

import * as React from "react"
import { DropdownMenu as DropdownMenuPrimitive } from "radix-ui"
import { cn } from "@/lib/utils"

const DropdownMenu = DropdownMenuPrimitive.Root
const DropdownMenuTrigger = DropdownMenuPrimitive.Trigger

function DropdownMenuContent({ className, sideOffset = 6, ...props }: React.ComponentProps<typeof DropdownMenuPrimitive.Content>) {
  return (
    <DropdownMenuPrimitive.Portal>
      <DropdownMenuPrimitive.Content
        sideOffset={sideOffset}
        className={cn(
          "z-50 min-w-[200px] overflow-hidden rounded-lg border border-border bg-background p-1 shadow-md data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95",
          className
        )}
        {...props}
      />
    </DropdownMenuPrimitive.Portal>
  )
}

function DropdownMenuItem({ className, ...props }: React.ComponentProps<typeof DropdownMenuPrimitive.Item>) {
  return (
    <DropdownMenuPrimitive.Item
      className={cn(
        "flex h-9 cursor-pointer items-center gap-2 rounded-md px-2.5 text-body text-text-primary outline-none select-none data-[highlighted]:bg-muted data-[disabled]:pointer-events-none data-[disabled]:opacity-50 [&_svg]:size-4 [&_svg]:text-text-tertiary",
        className
      )}
      {...props}
    />
  )
}

function DropdownMenuLabel({ className, ...props }: React.ComponentProps<typeof DropdownMenuPrimitive.Label>) {
  return <DropdownMenuPrimitive.Label className={cn("px-2.5 py-2 text-caption text-text-tertiary", className)} {...props} />
}

function DropdownMenuSeparator({ className, ...props }: React.ComponentProps<typeof DropdownMenuPrimitive.Separator>) {
  return <DropdownMenuPrimitive.Separator className={cn("my-1 h-px bg-border", className)} {...props} />
}

export { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger }
