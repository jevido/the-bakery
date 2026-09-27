// Markdown for task descriptions and comments. Raw HTML in the text is
// shown as text, never rendered, and the result is sanitised as well, so a
// description can hold nothing that runs.

import { Marked } from 'marked'
import DOMPurify from 'dompurify'
import { Browser } from '@wailsio/runtime'

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

const marked = new Marked({
  gfm: true,
  breaks: true,
  renderer: {
    html: ({ text }) => escapeHtml(text),
  },
})

export function renderMarkdown(source: string): string {
  const html = marked.parse(source, { async: false })
  return DOMPurify.sanitize(html, { ALLOWED_URI_REGEXP: /^(?:https?:|mailto:)/i })
}

// Links in rendered markdown open in the system browser; followed inside
// the app they would replace the app itself.
export function openLinksOutside(event: MouseEvent) {
  const link = (event.target as HTMLElement).closest('a')
  if (!link) return
  event.preventDefault()
  const href = link.getAttribute('href') ?? ''
  if (/^(?:https?:|mailto:)/i.test(href)) Browser.OpenURL(href)
}
