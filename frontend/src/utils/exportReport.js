const rupiahFormat = '[$Rp-421] #,##0;[Red]-[$Rp-421] #,##0'

export function rupiah(value) {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(Number(value || 0)).replace(/\u00a0/g, ' ')
}

export async function exportPdf({ fileName, title, subtitle, columns, rows, summary = [] }) {
  const [{ jsPDF }, { autoTable }] = await Promise.all([
    import('jspdf'),
    import('jspdf-autotable'),
  ])
  const landscape = columns.length > 5
  const document = new jsPDF({ orientation: landscape ? 'landscape' : 'portrait' })

  document.setFontSize(16)
  document.text(title, 14, 17)
  document.setFontSize(9)
  document.setTextColor(100)
  document.text(subtitle, 14, 24)
  if (summary.length) document.text(summary.join('   |   '), 14, 30)

  autoTable(document, {
    startY: summary.length ? 36 : 30,
    head: [columns],
    body: rows,
    theme: 'grid',
    styles: { fontSize: 8, cellPadding: 2.5, overflow: 'linebreak' },
    headStyles: { fillColor: [73, 104, 92], textColor: 255, fontStyle: 'bold' },
    alternateRowStyles: { fillColor: [247, 249, 247] },
  })

  document.save(`${fileName}.pdf`)
}

export async function exportExcel({ fileName, sheetName, columns, rows, summary = [] }) {
  const { default: writeExcelFile } = await import('write-excel-file/browser')
  const columnCount = columns.length
  const fullRow = (value, style = {}) => [
    { value, columnSpan: columnCount, ...style },
    ...Array.from({ length: columnCount - 1 }, () => null),
  ]
  const header = columns.map((column) => ({
    value: column.label,
    fontWeight: 'bold',
    textColor: '#FFFFFF',
    backgroundColor: '#49685C',
  }))
  const dataRows = rows.map((row) => columns.map((column) => {
    const value = row[column.key] ?? ''
    if (column.currency) return { value: Number(value || 0), type: Number, format: rupiahFormat }
    return value
  }))
  const summaryRows = summary.length
    ? [fullRow(summary.join(' | '), { fontWeight: 'bold', backgroundColor: '#E8F0EB' })]
    : []
  const sheetData = [
    fullRow(title, { fontWeight: 'bold', fontSize: 16 }),
    fullRow(sheetName, { textColor: '#68736D' }),
    ...summaryRows,
    header,
    ...dataRows,
  ]

  await writeExcelFile(sheetData, {
    sheet: sheetName.slice(0, 31),
    columns: columns.map((column) => ({ width: column.width || 18 })),
    stickyRowsCount: summary.length ? 4 : 3,
  }).toFile(`${fileName}.xlsx`)
}
