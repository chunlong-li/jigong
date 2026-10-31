// 统一 API 客户端与数据类型

export interface Project {
  id: number
  name: string
  site_address: string
  client: string
  contact: string
  contract_amount: number
  start_date: string
  status: string
  contract_type: string
  remark: string
  created_at: string
  updated_at: string
}

export interface Worker {
  id: number
  name: string
  phone: string
  id_card: string
  internal_rate: number
  external_rate: number
  overtime_rate: number
  remark: string
  created_at: string
}

export type WorkMode = 'day' | 'hour' | 'piece'

export interface WorkLog {
  id: number
  project_id: number
  project_name: string
  worker_id: number
  worker_name: string
  work_date: string
  mode: WorkMode
  day_type: string
  quantity: number
  unit_price: number
  amount: number
  content: string
  remark: string
  settled_at: string
  created_at: string
}

export interface WorkLogGroup {
  month: string
  project_id: number
  project_name: string
  worker_id: number
  worker_name: string
  days: number
  overtime_hours: number
  overtime_count: number
  hour_qty: number
  piece_qty: number
  amount: number
  record_count: number
  last_date: string
}

export interface PriceCategory {
  id: number
  kind: string
  name: string
  created_at: string
}

export interface PriceItem {
  id: number
  kind: 'material' | 'labor_all' | 'labor_only'
  category: string
  name: string
  spec: string
  unit: string
  price: number
  client_price: number
  remark: string
  created_at: string
}

export interface WageSettlement {
  id: number
  project_id: number
  project_name: string
  worker_id: number
  worker_name: string
  period_start: string
  period_end: string
  days: number
  overtime_hours: number
  overtime_count: number
  hour_qty: number
  piece_qty: number
  amount: number
  remark: string
  created_at: string
}

export interface Material {
  id: number
  project_id: number
  project_name: string
  in_date: string
  name: string
  spec: string
  unit: string
  quantity: number
  unit_price: number
  amount: number
  supplier: string
  remark: string
  created_at: string
}

export interface QuotationItem {
  id?: number
  room: string
  name: string
  spec: string
  unit: string
  quantity: number
  unit_price: number
  amount: number
  sort_order?: number
}

export interface Quotation {
  id: number
  project_id: number
  project_name: string
  room: string
  contract_type: string
  quote_type: string
  quote_no: string
  quote_date: string
  client: string
  title: string
  tax_rate: number
  subtotal: number
  tax_amount: number
  total: number
  remark: string
  created_at: string
  items: QuotationItem[]
}

export interface SettlementItem {
  id?: number
  category: string
  room: string
  name: string
  spec: string
  unit: string
  quantity: number
  unit_price: number
  amount: number
  sort_order?: number
}

export interface Settlement {
  id: number
  project_id: number
  project_name: string
  room: string
  settle_no: string
  settle_date: string
  period_start: string
  period_end: string
  payee: string
  contract_type: string
  title: string
  work_amount: number
  material_amount: number
  deduction: number
  payable: number
  received: number
  remark: string
  created_at: string
  items: SettlementItem[]
}

export interface Summary {
  project_count: number
  worker_count: number
  work_amount: number
  material_amount: number
  month_work: number
  month_material: number
}

const base = ''

async function request<T>(url: string, options: RequestInit = {}, redirectOn401 = true): Promise<T> {
  const res = await fetch(base + url, {
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })

  if (res.status === 401) {
    if (redirectOn401 && !location.pathname.startsWith('/login')) {
      location.href = '/login'
    }
    throw new Error('未登录或登录已过期')
  }

  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) {
    throw new Error((data && data.error) || `请求失败 (${res.status})`)
  }
  return data as T
}

const get = <T>(url: string, redirectOn401 = true) => request<T>(url, {}, redirectOn401)
const post = <T>(url: string, body?: unknown) =>
  request<T>(url, { method: 'POST', body: JSON.stringify(body ?? {}) })
const put = <T>(url: string, body?: unknown) =>
  request<T>(url, { method: 'PUT', body: JSON.stringify(body ?? {}) })
const del = <T>(url: string) => request<T>(url, { method: 'DELETE' })

export const api = {
  login: (username: string, password: string) =>
    post<{ username: string }>('/api/login', { username, password }),
  logout: () => post<{ status: string }>('/api/logout'),
  me: () => get<{ username: string }>('/api/me', false),
  initStatus: () => get<{ first_time: boolean }>('/api/init-status', false),
  changePassword: (old_password: string, new_password: string) =>
    post<{ status: string }>('/api/password', { old_password, new_password }),

  summary: () => get<Summary>('/api/summary'),

  projects: (params: Record<string, string> = {}) => {
    const qs = new URLSearchParams(params).toString()
    return get<Project[]>(`/api/projects${qs ? `?${qs}` : ''}`)
  },
  createProject: (p: Partial<Project>) => post<Project>('/api/projects', p),
  updateProject: (id: number, p: Partial<Project>) => put<{ status: string }>(`/api/projects/${id}`, p),
  deleteProject: (id: number) => del<{ status: string }>(`/api/projects/${id}`),
  projectWorkerRates: (id: number) =>
    get<{ count: number; avg_internal: number; avg_external: number }>(`/api/projects/${id}/worker-rates`),

  workers: (q = '') => get<Worker[]>(`/api/workers${q ? `?q=${encodeURIComponent(q)}` : ''}`),
  createWorker: (w: Partial<Worker>) => post<Worker>('/api/workers', w),
  updateWorker: (id: number, w: Partial<Worker>) => put<{ status: string }>(`/api/workers/${id}`, w),
  deleteWorker: (id: number) => del<{ status: string }>(`/api/workers/${id}`),

  worklogs: (params: Record<string, string> = {}) => {
    const qs = new URLSearchParams(params).toString()
    return get<WorkLog[]>(`/api/worklogs${qs ? `?${qs}` : ''}`)
  },
  createWorkLog: (w: Partial<WorkLog>) => post<WorkLog>('/api/worklogs', w),
  batchWorkLogs: (payload: {
    project_id: number
    worker_ids: number[]
    dates: string[]
    mode: string
    day_type?: string
    quantity?: number
    hours?: number
    unit_price: number
    content: string
    remark?: string
  }) => post<{ created: number }>('/api/worklogs/batch', payload),
  settleWorkLog: (project_id: number, worker_id: number) =>
    post<WageSettlement>('/api/worklogs/settle', { project_id, worker_id }),
  settleWorkLogThrough: (project_id: number, worker_id: number, through: string) =>
    post<WageSettlement>('/api/worklogs/settle', { project_id, worker_id, through }),
  wageSettlements: (params: Record<string, string> = {}) => {
    const qs = new URLSearchParams(params).toString()
    return get<WageSettlement[]>(`/api/wage-settlements${qs ? `?${qs}` : ''}`)
  },
  groupedWorkLogs: (params: Record<string, string> = {}) => {
    const qs = new URLSearchParams(params).toString()
    return get<WorkLogGroup[]>(`/api/worklogs/grouped${qs ? `?${qs}` : ''}`)
  },
  updateWorkLog: (id: number, w: Partial<WorkLog>) => put<{ status: string }>(`/api/worklogs/${id}`, w),
  deleteWorkLog: (id: number) => del<{ status: string }>(`/api/worklogs/${id}`),

  priceCategories: (kind: string) =>
    get<PriceCategory[]>(`/api/price-categories?kind=${encodeURIComponent(kind)}`),
  createPriceCategory: (kind: string, name: string) =>
    post<{ id: number; kind: string; name: string }>('/api/price-categories', { kind, name }),
  renamePriceCategory: (id: number, name: string) =>
    put<{ status: string; kind: string; name: string }>(`/api/price-categories/${id}`, { name }),

  priceItems: (kind: string, q = '', category = '') => {
    const p = new URLSearchParams({ kind })
    if (q) p.set('q', q)
    if (category) p.set('category', category)
    return get<PriceItem[]>(`/api/price-items?${p.toString()}`)
  },
  createPriceItem: (x: Partial<PriceItem>) => post<PriceItem>('/api/price-items', x),
  updatePriceItem: (id: number, x: Partial<PriceItem>) => put<{ status: string }>(`/api/price-items/${id}`, x),
  deletePriceItem: (id: number) => del<{ status: string }>(`/api/price-items/${id}`),

  materials: (params: Record<string, string> = {}) => {
    const qs = new URLSearchParams(params).toString()
    return get<Material[]>(`/api/materials${qs ? `?${qs}` : ''}`)
  },
  createMaterial: (m: Partial<Material>) => post<Material>('/api/materials', m),
  updateMaterial: (id: number, m: Partial<Material>) => put<{ status: string }>(`/api/materials/${id}`, m),
  deleteMaterial: (id: number) => del<{ status: string }>(`/api/materials/${id}`),

  quotations: (projectId?: number) =>
    get<Quotation[]>(`/api/quotations${projectId ? `?project_id=${projectId}` : ''}`),
  quotation: (id: number) => get<Quotation>(`/api/quotations/${id}`),
  createQuotation: (q: Partial<Quotation>) => post<Quotation>('/api/quotations', q),
  updateQuotation: (id: number, q: Partial<Quotation>) => put<{ status: string }>(`/api/quotations/${id}`, q),
  deleteQuotation: (id: number) => del<{ status: string }>(`/api/quotations/${id}`),

  settlements: (projectId?: number) =>
    get<Settlement[]>(`/api/settlements${projectId ? `?project_id=${projectId}` : ''}`),
  settlement: (id: number) => get<Settlement>(`/api/settlements/${id}`),
  dailySummary: (projectId: number, from = '', to = '') => {
    const p = new URLSearchParams({ project_id: String(projectId) })
    if (from) p.set('from', from)
    if (to) p.set('to', to)
    return get<{ date: string; workers: number; days: number }[]>(`/api/worklogs/daily?${p.toString()}`)
  },
  previewSettlement: (payload: { project_id: number; period_start: string; period_end: string; deduction: number }) =>
    post<{ items: SettlementItem[]; work_amount: number; material_amount: number; deduction: number; payable: number }>(
      '/api/settlements/preview',
      payload,
    ),
  createSettlement: (s: Partial<Settlement>) => post<Settlement>('/api/settlements', s),
  deleteSettlement: (id: number) => del<{ status: string }>(`/api/settlements/${id}`),
  updateSettlementReceived: (id: number, received: number) =>
    put<{ status: string; received: number }>(`/api/settlements/${id}/received`, { received }),
}

export function downloadUrl(path: string) {
  return base + path
}
