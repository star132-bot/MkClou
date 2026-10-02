"use client";

import { ChevronLeft, ChevronRight, ImagePlus, LoaderCircle, Trash2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import { ApiError } from "@/lib/api/client";
import type { ProductImageRef } from "@/lib/product/types";
import { IMAGE_TYPES, uploadImage } from "@/lib/upload";
import { cn } from "@/lib/utils";

const MAX = 6;

/**
 * 商品封面图（PRD-02）：1～6 张，第一张为主图；支持多选上传、左右移动调整顺序、删除。
 * 上传中的图片用本地预览占位，完成后替换为服务端地址。
 */
export function ImageManager({
  value,
  onChange,
  invalid,
}: {
  value: ProductImageRef[];
  onChange: (images: ProductImageRef[]) => void;
  invalid?: boolean;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState<{ id: string; preview: string }[]>([]);
  // 上传是异步的，用 ref 保存最新的图片列表，避免多张同时完成时互相覆盖
  const latest = useRef(value);
  useEffect(() => {
    latest.current = value;
  }, [value]);

  const remaining = MAX - value.length - uploading.length;

  const pick = async (files: FileList | null) => {
    if (!files?.length) return;
    const list = Array.from(files).slice(0, Math.max(0, remaining));
    if (files.length > list.length) toast.error(`最多上传 ${MAX} 张封面图`);
    await Promise.all(
      list.map(async (file) => {
        const id = crypto.randomUUID();
        const preview = URL.createObjectURL(file);
        setUploading((u) => [...u, { id, preview }]);
        try {
          const img = await uploadImage(file);
          latest.current = [...latest.current, img];
          onChange(latest.current);
        } catch (e) {
          toast.error(e instanceof ApiError ? e.message : `“${file.name}”上传失败，请重试`);
        } finally {
          URL.revokeObjectURL(preview);
          setUploading((u) => u.filter((x) => x.id !== id));
        }
      }),
    );
  };

  const move = (i: number, d: -1 | 1) => {
    const next = [...value];
    [next[i], next[i + d]] = [next[i + d], next[i]];
    onChange(next);
  };

  return (
    <div className="flex flex-col gap-2">
      <ul className="grid grid-cols-3 gap-3 sm:grid-cols-4">
        {value.map((img, i) => (
          <li key={img.key} className="group relative aspect-[4/3] overflow-hidden rounded-md border border-border bg-muted">
            {/* eslint-disable-next-line @next/next/no-img-element -- 编辑页缩略图 */}
            <img src={img.url} alt={`封面图 ${i + 1}`} className="size-full object-cover" />
            {i === 0 && (
              <span className="absolute top-1.5 left-1.5 rounded-sm bg-background/90 px-1.5 text-caption font-medium text-text-primary">主图</span>
            )}
            <div className="absolute inset-x-0 bottom-0 flex justify-between bg-black/50 p-1 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100">
              <div className="flex gap-1">
                <IconButton label="向前移动" disabled={i === 0} onClick={() => move(i, -1)}>
                  <ChevronLeft className="size-4" strokeWidth={1.5} />
                </IconButton>
                <IconButton label="向后移动" disabled={i === value.length - 1} onClick={() => move(i, 1)}>
                  <ChevronRight className="size-4" strokeWidth={1.5} />
                </IconButton>
              </div>
              <IconButton label="删除这张图片" onClick={() => onChange(value.filter((_, j) => j !== i))}>
                <Trash2 className="size-4" strokeWidth={1.5} />
              </IconButton>
            </div>
          </li>
        ))}
        {uploading.map((u) => (
          <li key={u.id} className="relative aspect-[4/3] overflow-hidden rounded-md border border-border bg-muted" aria-busy="true">
            {/* eslint-disable-next-line @next/next/no-img-element -- 本地预览 */}
            <img src={u.preview} alt="" className="size-full object-cover opacity-50" />
            <LoaderCircle className="absolute inset-0 m-auto size-5 animate-spin text-text-primary" strokeWidth={1.5} aria-label="正在上传" />
          </li>
        ))}
        {remaining > 0 && (
          <li>
            <button
              type="button"
              onClick={() => inputRef.current?.click()}
              className={cn(
                "flex aspect-[4/3] w-full flex-col items-center justify-center gap-1 rounded-md border border-dashed text-caption text-text-tertiary transition-colors outline-none hover:border-text-tertiary hover:text-text-primary focus-visible:ring-2 focus-visible:ring-ring",
                invalid ? "border-danger" : "border-border",
              )}
            >
              <ImagePlus className="size-5" strokeWidth={1.5} aria-hidden />
              添加图片
            </button>
          </li>
        )}
      </ul>
      <input
        ref={inputRef}
        type="file"
        accept={IMAGE_TYPES.join(",")}
        multiple
        className="hidden"
        onChange={(e) => {
          void pick(e.target.files);
          e.target.value = "";
        }}
      />
      <p className="text-caption text-text-tertiary">JPG、PNG、WebP、GIF，单张不超过 5MB，最多 {MAX} 张。第一张为主图，商城中按 4:3 展示。</p>
    </div>
  );
}

function IconButton({ label, onClick, disabled, children }: { label: string; onClick: () => void; disabled?: boolean; children: React.ReactNode }) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      title={label}
      className="flex size-7 items-center justify-center rounded-sm text-white transition-colors hover:bg-white/20 focus-visible:ring-2 focus-visible:ring-white focus-visible:outline-none disabled:opacity-30"
    >
      {children}
    </button>
  );
}
