// 图片上传（docs/api-list.md #28）：服务端校验、去除 EXIF、重新编码后存入公有桶。
import { api, ApiError } from "@/lib/api/client";

export interface UploadedImage {
  key: string;
  url: string;
  width: number;
  height: number;
}

export const IMAGE_TYPES = ["image/jpeg", "image/png", "image/webp", "image/gif"];
export const MAX_IMAGE_BYTES = 5 * 1024 * 1024;

/** 上传前在本地先检查格式和大小，避免无效的上传请求。 */
export function checkImageFile(file: File): string | null {
  if (!IMAGE_TYPES.includes(file.type)) return `“${file.name}”不是 JPG、PNG、WebP 或 GIF 图片`;
  if (file.size > MAX_IMAGE_BYTES) return `“${file.name}”超过 5MB，请压缩后重新上传`;
  return null;
}

export async function uploadImage(file: File): Promise<UploadedImage> {
  const err = checkImageFile(file);
  if (err) throw new ApiError(400, 10001, err);
  const form = new FormData();
  form.append("file", file);
  return api<UploadedImage>("/uploads/images", { method: "POST", body: form });
}
