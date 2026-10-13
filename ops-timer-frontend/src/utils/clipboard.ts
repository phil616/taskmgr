/** Copy the complete original text, preserving the editor's focus/selection. */
export async function copyText(text: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    try { await navigator.clipboard.writeText(text); return } catch { /* Qt/HTTP fallback below */ }
  }
  const focused = document.activeElement as HTMLElement | null
  const selection = window.getSelection()
  const ranges = selection ? Array.from({ length: selection.rangeCount }, (_, i) => selection.getRangeAt(i).cloneRange()) : []
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.style.cssText = 'position:fixed;left:-9999px;top:0;opacity:0'
  // Keep it inside the active dialog so Vuetify's focus trap permits selection.
  const parent = focused?.closest('.v-overlay__content') || document.body
  parent.appendChild(textarea)
  try {
    textarea.focus()
    textarea.select()
    if (!document.execCommand('copy')) throw new Error('浏览器未允许复制，请检查剪贴板权限')
  } finally {
    textarea.remove()
    focused?.focus({ preventScroll: true })
    selection?.removeAllRanges()
    ranges.forEach(range => selection?.addRange(range))
  }
}
