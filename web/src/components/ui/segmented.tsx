"use client"

import * as React from "react"
import { RadioGroup as RadioGroupPrimitive } from "radix-ui"
import { cn } from "@/lib/utils"

interface SegmentedProps<T extends string> {
  value: T
  onValueChange: (value: T) => void
  options: { value: T; label: React.ReactNode }[]
  "aria-label"?: string
  "aria-labelledby"?: string
  className?: string
}

/** 分段选择：少量互斥选项（如卡片比例、明暗模式）。基于 RadioGroup，支持方向键切换。 */
function Segmented<T extends string>({ value, onValueChange, options, className, ...aria }: SegmentedProps<T>) {
  return (
    <RadioGroupPrimitive.Root
      value={value}
      onValueChange={(v) => onValueChange(v as T)}
      orientation="horizontal"
      className={cn("inline-flex h-9 items-center gap-1 rounded-md bg-muted p-1", className)}
      {...aria}
    >
      {options.map((o) => (
        <RadioGroupPrimitive.Item
          key={o.value}
          value={o.value}
          className="inline-flex h-7 items-center justify-center rounded-sm px-3 text-body font-medium text-text-secondary transition-colors duration-150 outline-none hover:text-text-primary focus-visible:ring-2 focus-visible:ring-ring data-[state=checked]:bg-background data-[state=checked]:text-text-primary data-[state=checked]:shadow-sm"
        >
          {o.label}
        </RadioGroupPrimitive.Item>
      ))}
    </RadioGroupPrimitive.Root>
  )
}

export { Segmented }
