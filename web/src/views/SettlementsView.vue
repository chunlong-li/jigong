<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">结算单</h2>
        <div class="page-sub">对甲方结算：明细按报价单选，点工可按记工自动带工数</div>
      </div>
      <el-button type="primary" @click="openCreate">
        <el-icon><Plus /></el-icon>新建结算单
      </el-button>
    </div>

    <div class="card">
      <el-table :data="list" v-loading="loading" empty-text="还没有结算单">
        <el-table-column prop="settle_no" label="结算编号" width="150" />
        <el-table-column prop="project_name" label="项目" min-width="170" show-overflow-tooltip />
        <el-table-column label="承包方式" width="110" align="center">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ row.contract_type || '包工包料' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="结算期间" width="195">
          <template #default="{ row }">{{ row.period_start || '-' }} 至 {{ row.period_end || '-' }}</template>
        </el-table-column>
        <el-table-column label="应付金额" width="120" align="right">
          <template #default="{ row }"><span class="money">{{ money(row.payable) }}</span></template>
        </el-table-column>
        <el-table-column label="实收金额" width="120" align="right">
          <template #default="{ row }">
            <span class="money" :style="{ color: row.received < row.payable ? '#e6a23c' : '' }">
              {{ money(row.received) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="250" align="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openView(row)">查看</el-button>
            <el-button link type="warning" @click="openReceived(row)">实收</el-button>
            <el-button link type="success" @click="exportExcel(row)">导出 Excel</el-button>
            <el-button link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialog" title="新建结算单" width="960px" top="5vh">
      <el-form :model="form" label-width="90px">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="项目">
              <el-radio-group v-model="projectMode">
                <el-radio-button value="existing">已有项目</el-radio-button>
                <el-radio-button value="custom">自定义</el-radio-button>
              </el-radio-group>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="承包方式">
              <el-radio-group v-model="form.contract_type" @change="onContractTypeChange">
                <el-radio-button value="包工包料">包工包料</el-radio-button>
                <el-radio-button value="包工">包工</el-radio-button>
                <el-radio-button value="点工">点工</el-radio-button>
              </el-radio-group>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item v-if="projectMode === 'existing'" label="选择项目" required>
              <el-select v-model="form.project_id" style="width: 100%" placeholder="选择工地" @change="onProjectChange">
                <el-option v-for="p in projects" :key="p.id" :label="p.name" :value="p.id" />
              </el-select>
            </el-form-item>
            <el-form-item v-else label="项目名称" required>
              <el-input v-model="form.project_name" placeholder="自定义项目名称" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="收款方">
              <el-input v-model="form.payee" placeholder="甲方 / 甲方代表" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="结算期间">
              <el-date-picker
                v-model="period"
                type="daterange"
                value-format="YYYY-MM-DD"
                start-placeholder="开始"
                end-placeholder="结束"
                style="width: 100%"
                @change="onPeriodChange"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="扣款">
              <el-input-number v-model="form.deduction" :min="0" :precision="2" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="结算编号">
              <el-input v-model="form.settle_no" placeholder="自动生成" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="结算日期">
              <el-date-picker v-model="form.settle_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>

        <div class="page-header" style="margin: 6px 0 10px">
          <strong style="font-size: 14px">结算明细</strong>
          <div style="display: flex; gap: 8px; align-items: center">
            <el-radio-group v-model="priceBase" size="small" @change="onPriceBaseChange">
              <el-radio-button value="external">对外单价</el-radio-button>
              <el-radio-button value="internal">内部单价</el-radio-button>
            </el-radio-group>
            <template v-if="form.contract_type === '点工'">
              <span class="form-tip">点工单价</span>
              <el-input-number v-model="dayRate" :min="0" :precision="2" size="small" style="width: 110px" />
              <el-button size="small" :disabled="projectMode !== 'existing' || !form.project_id" @click="fillDayTotal">
                按记工汇总
              </el-button>
              <el-button size="small" :disabled="projectMode !== 'existing' || !form.project_id" @click="fillDayDetail">
                按日期明细
              </el-button>
            </template>
            <el-button size="small" @click="addItem">
              <el-icon><Plus /></el-icon>手动加一行
            </el-button>
          </div>
        </div>

        <el-table :data="form.items" size="small" border empty-text="点工可自动带出，其它承包方式从施工价格选择">
          <el-table-column label="序号" width="52" align="center">
            <template #default="{ $index }">{{ $index + 1 }}</template>
          </el-table-column>
          <el-table-column label="房间" width="95">
            <template #default="{ row }">
              <el-input v-model="row.room" size="small" placeholder="客厅" />
            </template>
          </el-table-column>
          <el-table-column label="名称" min-width="220">
            <template #default="{ row }">
              <el-cascader
                v-if="form.contract_type !== '点工'"
                v-model="row.path"
                :options="cascaderOptions"
                :props="{ expandTrigger: 'hover' }"
                size="small"
                clearable
                filterable
                style="width: 100%"
                placeholder="选择分类 / 条目"
                @change="(val: any) => onPickItem(row, val)"
              />
              <el-input v-else v-model="row.name" size="small" placeholder="如：2026-10-25（3 人）" />
            </template>
          </el-table-column>
          <el-table-column label="尺寸" width="140">
            <template #default="{ row }">
              <el-input v-model="row.size" size="small" placeholder="5*6 或 5+6" @change="onSizeChange(row)" />
            </template>
          </el-table-column>
          <el-table-column label="单位" width="80">
            <template #default="{ row }">
              <el-input v-model="row.unit" size="small" placeholder="㎡" />
            </template>
          </el-table-column>
          <el-table-column label="数量" width="100">
            <template #default="{ row }">
              <el-input-number v-model="row.quantity" size="small" :min="0" :precision="3" :controls="false" style="width: 100%" />
            </template>
          </el-table-column>
          <el-table-column label="单价(元)" width="110">
            <template #default="{ row }">
              <el-input-number v-model="row.unit_price" size="small" :min="0" :precision="2" :controls="false" style="width: 100%" />
            </template>
          </el-table-column>
          <el-table-column label="金额(元)" width="110" align="right">
            <template #default="{ row }">
              <span class="money">{{ money(row.quantity * row.unit_price) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="" width="48" align="center">
            <template #default="{ $index }">
              <el-button link type="danger" @click="form.items.splice($index, 1)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="table-footer">
          <span>明细合计：<span class="money">¥ {{ money(subtotal) }}</span></span>
          <span>扣款：<span class="money">¥ {{ money(form.deduction || 0) }}</span></span>
          <span>应付：<span class="money pos" style="font-size: 16px">¥ {{ money(payable) }}</span></span>
        </div>

        <el-form-item label="备注" style="margin-top: 10px">
          <el-input v-model="form.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="receivedDialog" title="登记实收金额" width="420px">
      <el-form label-width="90px">
        <el-form-item label="应付金额">
          <span class="money">¥ {{ money(receivedRow?.payable) }}</span>
        </el-form-item>
        <el-form-item label="实收金额">
          <el-input-number v-model="receivedValue" :min="0" :precision="2" style="width: 100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="receivedDialog = false">取消</el-button>
        <el-button type="primary" :loading="receivedSaving" @click="saveReceived">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="viewDialog" :title="'结算单 · ' + (viewRow?.settle_no || '')" width="760px">
      <el-descriptions :column="2" border size="small">
        <el-descriptions-item label="项目">{{ viewRow?.project_name }}</el-descriptions-item>
        <el-descriptions-item label="承包方式">{{ viewRow?.contract_type || '包工包料' }}</el-descriptions-item>
        <el-descriptions-item label="期间">
          {{ viewRow?.period_start || '-' }} 至 {{ viewRow?.period_end || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="结算日期">{{ viewRow?.settle_date }}</el-descriptions-item>
        <el-descriptions-item label="收款方">{{ viewRow?.payee || '-' }}</el-descriptions-item>
        <el-descriptions-item label="扣款">¥ {{ money(viewRow?.deduction) }}</el-descriptions-item>
        <el-descriptions-item label="应付金额">
          <span class="money pos">¥ {{ money(viewRow?.payable) }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="实收金额">
          <span class="money">¥ {{ money(viewRow?.received) }}</span>
        </el-descriptions-item>
      </el-descriptions>
      <el-table :data="viewRow?.items ?? []" size="small" border style="margin-top: 12px" max-height="320">
        <el-table-column prop="room" label="房间" width="90" show-overflow-tooltip />
        <el-table-column prop="name" label="名称" min-width="200" show-overflow-tooltip />
        <el-table-column prop="spec" label="尺寸" width="120" show-overflow-tooltip />
        <el-table-column prop="unit" label="单位" width="70" align="center" />
        <el-table-column prop="quantity" label="数量" width="100" align="right" />
        <el-table-column label="单价" width="110" align="right">
          <template #default="{ row }">{{ money(row.unit_price) }}</template>
        </el-table-column>
        <el-table-column label="金额" width="120" align="right">
          <template #default="{ row }"><span class="money">{{ money(row.amount) }}</span></template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type PriceItem, type Project, type Settlement } from '../api'
import { money, monthStart, today } from '../utils/format'

interface Row {
  room: string
  name: string
  size: string
  unit: string
  path: string[]
  quantity: number
  unit_price: number
  total?: boolean // 点工「总工数」行：唯一带价的行
}

const list = ref<Settlement[]>([])
const projects = ref<Project[]>([])
const materialPrices = ref<PriceItem[]>([])
const laborAll = ref<PriceItem[]>([])
const laborOnly = ref<PriceItem[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = ref(false)
const viewDialog = ref(false)
const viewRow = ref<Settlement | null>(null)
const projectMode = ref<'existing' | 'custom'>('existing')
const dayRate = ref(0)
const priceBase = ref<'external' | 'internal'>('external')
const avgInternal = ref(0)
const avgExternal = ref(0)
const ratesFor = ref(0)
const period = ref<[string, string] | null>([monthStart(), today()])

const form = reactive<Partial<Settlement> & { items: Row[] }>({
  project_id: undefined,
  project_name: '',
  contract_type: '包工包料',
  settle_no: '',
  settle_date: today(),
  period_start: monthStart(),
  period_end: today(),
  payee: '',
  deduction: 0,
  remark: '',
  items: [],
})

// 三套价目：材料 / 包工包料 / 包工；第一级在级联里显示，名称只记「分类/条目」
const priceSources = computed(() => [
  { key: 'material', label: '材料', items: materialPrices.value },
  { key: 'labor_all', label: '包工包料', items: laborAll.value },
  { key: 'labor_only', label: '包工', items: laborOnly.value },
])

function priceFor(p: PriceItem, kind: string): number {
  if (kind === 'material') return p.price
  if (priceBase.value === 'external') return p.client_price || p.price
  return p.price
}

const cascaderOptions = computed(() =>
  priceSources.value.map((src) => {
    const map = new Map<string, { value: string; label: string }[]>()
    for (const p of src.items) {
      const cat = p.category || '未分类'
      if (!map.has(cat)) map.set(cat, [])
      map.get(cat)!.push({
        value: p.name,
        label: `${p.name}${p.spec ? '（' + p.spec + '）' : ''}`,
      })
    }
    return {
      value: src.key,
      label: src.label,
      children: [...map.entries()].map(([cat, children]) => ({ value: cat, label: cat, children })),
    }
  }),
)

const subtotal = computed(() =>
  Number(form.items.reduce((s, it) => s + Number(it.quantity || 0) * Number(it.unit_price || 0), 0).toFixed(2)),
)
const payable = computed(() => Number((subtotal.value - Number(form.deduction || 0)).toFixed(2)))

function emptyRow(): Row {
  return { room: '', name: '', size: '', unit: '', path: [], quantity: 1, unit_price: 0 }
}

// 尺寸算式：5*6=30、5+6=11，结果直接填数量
function parseSize(input: string): number | null {
  if (!input) return null
  const expr = input.replace(/×/g, '*').replace(/÷/g, '/').replace(/\s+/g, '')
  if (!/^[0-9.+\-*/()]+$/.test(expr)) return null
  if (/[+\-*/.]$/.test(expr)) return null
  try {
    const v = Function('"use strict";return (' + expr + ')')()
    if (typeof v === 'number' && isFinite(v) && v > 0) return Math.round(v * 1000) / 1000
  } catch {
    /* ignore */
  }
  return null
}

function onSizeChange(row: Row) {
  const v = parseSize(row.size)
  if (v !== null) {
    row.quantity = v
    ElMessage.success(`尺寸计算：${row.size} = ${v}`)
  }
}

function onPickItem(row: Row, val: string[] | null) {
  if (!val || val.length < 3) {
    row.name = ''
    return
  }
  const [kind, cat, name] = val
  const src = priceSources.value.find((s2) => s2.key === kind)
  const p = src?.items.find((x) => (x.category || '未分类') === cat && x.name === name)
  if (!p) return
  row.name = `${cat}/${p.name}`
  row.unit = p.unit
  row.unit_price = priceFor(p, kind)
}

async function loadRates(id: number) {
  if (!id || ratesFor.value === id) return
  try {
    const rates = await api.projectWorkerRates(id)
    avgInternal.value = rates.avg_internal
    avgExternal.value = rates.avg_external
  } catch {
    avgInternal.value = 0
    avgExternal.value = 0
  }
  ratesFor.value = id
}

async function onProjectChange(id: number) {
  const p = projects.value.find((x) => x.id === id)
  if (!p) return

  // 期间：项目开工日期 → 今天（没填开工日期则用本月 1 号）
  const start = p.start_date || monthStart()
  form.period_start = start
  form.period_end = today()
  period.value = [start, today()]

  // 承包方式跟项目走，但点工单价要用工人均价，先拉数据
  await loadRates(id)
  if (p.contract_type) form.contract_type = p.contract_type
  applyDayRate()
  if (form.contract_type === '点工' && dayRate.value === 0) {
    ElMessage.info('该项目暂无记工工人，点工单价请手动填写')
  }
}

function applyDayRate() {
  if (form.contract_type !== '点工') return
  dayRate.value = priceBase.value === 'external' ? avgExternal.value : avgInternal.value
}

function onPriceBaseChange() {
  if (form.contract_type === '点工') {
    applyDayRate()
    // 只有「总工数 / 点工合计」行带价，明细行保持 0
    for (const it of form.items) {
      if (it.total) it.unit_price = dayRate.value
    }
    return
  }
  // 包工 / 包工包料 / 材料：按新口径重算已选条目
  for (const it of form.items) {
    if (!it.path || it.path.length < 3) continue
    const [kind, cat, name] = it.path
    const src = priceSources.value.find((s2) => s2.key === kind)
    const p = src?.items.find((x) => (x.category || '未分类') === cat && x.name === name)
    if (p) it.unit_price = priceFor(p, kind)
  }
}

async function onContractTypeChange() {
  if (form.contract_type !== '点工') return
  if (projectMode.value === 'existing' && form.project_id) {
    await loadRates(form.project_id)
  }
  applyDayRate()
}

function onPeriodChange(v: [string, string] | null) {
  form.period_start = v?.[0] ?? ''
  form.period_end = v?.[1] ?? ''
}

function genNo(): string {
  const d = new Date()
  const p = (x: number) => String(x).padStart(2, '0')
  return `JS${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}`
}

// 点工：按记工汇总成一行
async function fillDayTotal() {
  if (!form.project_id) return
  try {
    const rows = await api.dailySummary(form.project_id, form.period_start, form.period_end)
    if (!rows.length) {
      ElMessage.info('该期间没有按天记工记录')
      return
    }
    const days = rows.reduce((s, r) => s + r.days, 0)
    const workers = Math.max(...rows.map((r) => r.workers))
    form.items = [
      {
        room: '',
        name: `点工合计（${workers} 人 / ${rows.length} 天）`,
        size: '',
        unit: '天',
        path: [],
        quantity: Number(days.toFixed(2)),
        unit_price: dayRate.value,
        total: true,
      },
    ]
    ElMessage.success(`已汇总 ${days} 个工`)
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

// 点工：按日期逐条明细
async function fillDayDetail() {
  if (!form.project_id) return
  try {
    const rows = await api.dailySummary(form.project_id, form.period_start, form.period_end)
    if (!rows.length) {
      ElMessage.info('该期间没有按天记工记录')
      return
    }
    const total = Number(rows.reduce((s, r) => s + r.days, 0).toFixed(2))
    const totalWorkers = Math.max(...rows.map((r) => r.workers))
    form.items = [
      {
        room: '',
        name: `总工数（${totalWorkers} 人 / ${rows.length} 天）`,
        size: '',
        unit: '天',
        path: [],
        quantity: total,
        unit_price: dayRate.value,
        total: true,
      },
      ...rows.map((r) => ({
        room: '',
        name: `${r.date}（${r.workers} 人）`,
        size: '',
        unit: '天',
        path: [],
        quantity: r.days,
        unit_price: 0,
        total: false,
      })),
    ]
    ElMessage.success(`已按 ${rows.length} 天填入明细，总工数 ${total}`)
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

async function load() {
  loading.value = true
  try {
    list.value = await api.settlements()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

async function openCreate() {
  Object.assign(form, {
    id: undefined,
    project_id: projects.value[0]?.id,
    project_name: '',
    contract_type: projects.value[0]?.contract_type || '包工包料',
    settle_no: genNo(),
    settle_date: today(),
    period_start: monthStart(),
    period_end: today(),
    payee: '',
    deduction: 0,
    remark: '',
    items: [],
  })
  projectMode.value = 'existing'
  period.value = [monthStart(), today()]
  dayRate.value = 0
  ratesFor.value = 0
  avgInternal.value = 0
  avgExternal.value = 0
  dialog.value = true
  if (form.project_id) {
    await onProjectChange(form.project_id)
  }
}

async function save() {
  if (projectMode.value === 'existing' && !form.project_id) {
    ElMessage.warning('请选择项目')
    return
  }
  if (projectMode.value === 'custom' && !form.project_name?.trim()) {
    ElMessage.warning('请填写项目名称')
    return
  }
  const items = form.items.filter((it) => it.name?.trim())
  if (!items.length) {
    ElMessage.warning('请至少添加一条结算明细')
    return
  }
  saving.value = true
  try {
    const payload = {
      ...form,
      project_id: projectMode.value === 'custom' ? 0 : form.project_id,
      project_name: projectMode.value === 'custom' ? form.project_name : '',
      work_amount: subtotal.value,
      material_amount: 0,
      payable: payable.value,
      items: items.map((it, i) => ({
        category: '记工',
        room: it.room,
        name: it.name,
        spec: it.size,
        unit: it.unit,
        quantity: it.quantity,
        unit_price: it.unit_price,
        amount: Number((Number(it.quantity || 0) * Number(it.unit_price || 0)).toFixed(2)),
        sort_order: i,
      })),
    }
    await api.createSettlement(payload)
    ElMessage.success('结算单已生成')
    dialog.value = false
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

function addItem() {
  form.items.push(emptyRow())
}

const receivedDialog = ref(false)
const receivedRow = ref<Settlement | null>(null)
const receivedValue = ref(0)
const receivedSaving = ref(false)

function openReceived(row: Settlement) {
  receivedRow.value = row
  receivedValue.value = row.received ?? row.payable ?? 0
  receivedDialog.value = true
}

async function saveReceived() {
  if (!receivedRow.value) return
  receivedSaving.value = true
  try {
    await api.updateSettlementReceived(receivedRow.value.id, Number(receivedValue.value || 0))
    ElMessage.success('实收金额已保存')
    receivedDialog.value = false
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    receivedSaving.value = false
  }
}

async function openView(row: Settlement) {
  try {
    viewRow.value = await api.settlement(row.id)
    viewDialog.value = true
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

async function remove(row: Settlement) {
  try {
    await ElMessageBox.confirm(`确定删除结算单「${row.settle_no || row.id}」？`, '删除确认', { type: 'warning' })
  } catch {
    return
  }
  try {
    await api.deleteSettlement(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

function exportExcel(row: Settlement) {
  window.open(`/api/settlements/${row.id}/excel`, '_blank')
}

onMounted(async () => {
  try {
    ;[projects.value, materialPrices.value, laborAll.value, laborOnly.value] = await Promise.all([
      api.projects(),
      api.priceItems('material'),
      api.priceItems('labor_all'),
      api.priceItems('labor_only'),
    ])
  } catch {
    /* ignore */
  }
  await load()
})
</script>
