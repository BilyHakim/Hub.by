export async function pdfEngine() {
  const pdf = await import('pdfjs-dist')
  const worker = await import('pdfjs-dist/build/pdf.worker.min.mjs?url')
  pdf.GlobalWorkerOptions.workerSrc = worker.default
  return pdf
}

export function bookPage(page, count, total) {
  return Math.max(1, Math.min(total, Math.round(page / count * total)))
}

export function readingTime(seconds = 0) {
  return [Math.floor(seconds / 3600), Math.floor(seconds / 60) % 60, Math.floor(seconds % 60)].map(value => String(value).padStart(2, '0')).join(':')
}
