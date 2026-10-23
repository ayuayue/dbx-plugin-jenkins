/**
 * 统一复制入口：宿主提供 copy 能力时优先用，兜底走浏览器剪贴板。
 * 返回是否成功，调用方自行决定是否提示。
 */
export async function copyText(text: string): Promise<boolean> {
  try {
    if (window.dbxPlugin?.copy) {
      await window.dbxPlugin.copy(text);
    } else {
      await navigator.clipboard.writeText(text);
    }
    return true;
  } catch {
    return false;
  }
}
