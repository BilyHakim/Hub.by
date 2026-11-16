import { Marked, Renderer } from 'marked'
import DOMPurify from 'dompurify'

function escapeHTML(text) {
  return text.replace(/[&<>"']/g, character => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character])
}

const markdown = new Marked({
  gfm: true,
  breaks: true,
  renderer: {
    html: ({ text }) => escapeHTML(text),
    image: ({ text }) => escapeHTML(text),
    table(token) {
      return `<div class="ai-markdown-table" tabindex="0" role="region" aria-label="Tabel jawaban AI">${Renderer.prototype.table.call(this, token)}</div>`
    },
    code(token) {
      return Renderer.prototype.code.call(this, token).replace('<pre>', '<pre tabindex="0" aria-label="Blok kode">')
    },
  },
})

export function renderAIMarkdown(content) {
  return DOMPurify.sanitize(markdown.parse(content), {
    ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'del', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'blockquote', 'pre', 'code', 'a', 'hr', 'table', 'thead', 'tbody', 'tr', 'th', 'td', 'div'],
    ALLOWED_ATTR: ['href', 'title', 'start', 'align', 'class', 'tabindex', 'role', 'aria-label'],
    ALLOW_DATA_ATTR: false,
  })
}
