/**
 * 复制文本到剪贴板。优先使用 Clipboard API；在非 HTTPS、内嵌页面或权限被拒绝时，
 * 退回到选中隐藏文本框 + execCommand 的方式。返回是否复制成功。
 */
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    const el = document.createElement("textarea");
    el.value = text;
    el.setAttribute("readonly", "");
    el.style.position = "fixed";
    el.style.opacity = "0";
    document.body.appendChild(el);
    el.select();
    try {
      return document.execCommand("copy");
    } catch {
      return false;
    } finally {
      el.remove();
    }
  }
}
