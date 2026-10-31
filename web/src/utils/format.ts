export function money(v: number | null | undefined): string {
  const n = Number(v ?? 0)
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

export function qty(v: number | null | undefined): string {
  const n = Number(v ?? 0)
  return n.toLocaleString('zh-CN', { maximumFractionDigits: 3 })
}

export function today(): string {
  const d = new Date()
  const p = (x: number) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

export function monthStart(): string {
  const d = new Date()
  const p = (x: number) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-01`
}

export const modeText: Record<string, string> = {
  day: '按天',
  hour: '按时',
  piece: '计件',
}

export const modeUnit: Record<string, string> = {
  day: '天',
  hour: '小时',
  piece: '件',
}
