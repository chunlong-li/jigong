<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">报价单</h2>
        <div class="page-sub">项目未定时也能先报价；明细从施工价格两级选择</div>
      </div>
      <el-button type="primary" @click="openCreate">
        <el-icon><Plus /></el-icon>新建报价单
      </el-button>
    </div>

    <div class="card">
      <el-table :data="list" v-loading="loading" empty-text="还没有报价单">
        <el-table-column prop="quote_no" label="报价编号" width="150" />
        <el-table-column prop="project_name" label="项目名称" min-width="170" show-overflow-tooltip />
        <el-table-column prop="client" label="客户" width="130" show-overflow-tooltip />
        <el-table-column label="报价类型" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.quote_type === '同行报价' ? 'warning' : 'success'" effect="plain">
              {{ row.quote_type || '客户报价' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="quote_date" label="日期" width="110" />
        <el-table-column label="金额" width="130" align="right">
          <template #default="{ row }"><span class="money">{{ money(row.total) }}</span></template>
        </el-table-column>
        <el-table-column label="操作" width="200" align="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="success" @click="exportExcel(row)">导出 Excel</el-button>
            <el-button link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialog" :title="editing ? '编辑报价单' : '新建报价单'" width="980px" top="5vh">
      <el-form :model="form" label-width="86px">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="项目名称" required>
              <el-input v-model="form.project_name" placeholder="还没定也能填，如：御景园 12 号楼吊顶" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="报价类型">
              <el-radio-group v-model="form.quote_type" @change="onQuoteTypeChange">
                <el-radio-button value="客户报价">客户报价</el-radio-button>
                <el-radio-button value="同行报价">同行报价</el-radio-button>
              </el-radio-group>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="承包方式">
              <el-radio-group v-model="form.contract_type" @change="onContractTypeChange">
                <el-radio-button value="包工包料">包工包料</el-radio-button>
                <el-radio-button value="包工">包工</el-radio-button>
              </el-radio-group>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="报价编号">
              <el-input v-model="form.quote_no" placeholder="自动生成，可修改" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="报价日期">
              <el-date-picker v-model="form.quote_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="客户">
              <el-input v-model="form.client" placeholder="甲方名称" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="税率(%)">
              <el-input-number v-model="form.tax_rate" :min="0" :max="100" :precision="2" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>

        <div class="page-header" style="margin: 6px 0 10px">
          <strong style="font-size: 14px">报价明细</strong>
          <div style="display: flex; gap: 8px; align-items: center">
            <span class="form-tip">可选：{{ form.contract_type }} 的施工价格</span>
            <el-button size="small" @click="addItem">
              <el-icon><Plus /></el-icon>添加一行
            </el-button>
          </div>
        </div>

        <el-table :data="form.items" size="small" border class="items-table" empty-text="请添加报价明细">
          <el-table-column label="序号" width="52" align="center">
            <template #default="{ $index }">{{ $index + 1 }}</template>
          </el-table-column>
          <el-table-column label="房间" width="95">
            <template #default="{ row }">
              <el-input v-model="row.room" size="small" placeholder="客厅" />
            </template>
          </el-table-column>
          <el-table-column label="项目名称" min-width="220">
            <template #default="{ row }">
              <el-cascader
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
            </template>
          </el-table-column>
          <el-table-column label="尺寸" width="150">
            <template #default="{ row }">
              <el-input
                v-model="row.spec"
                size="small"
                placeholder="5*6 或 5+6"
                @change="onSizeChange(row)"
              />
            </template>
          </el-table-column>
          <el-table-column label="单位" width="70">
            <template #default="{ row }">
              <el-input v-model="row.unit" size="small" placeholder="㎡" />
            </template>
          </el-table-column>
          <el-table-column label="数量" width="95">
            <template #default="{ row }">
              <el-input-number v-model="row.quantity" size="small" :min="0" :precision="3" :controls="false" style="width: 100%" />
            </template>
          </el-table-column>
          <el-table-column label="单价(元)" width="105">
            <template #default="{ row }">
              <el-input-number v-model="row.unit_price" size="small" :min="0" :precision="2" :controls="false" style="width: 100%" />
            </template>
          </el-table-column>
          <el-table-column label="金额(元)" width="105" align="right">
            <template #default="{ row }">
              <span class="money">{{ money(row.quantity * row.unit_price) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="" width="48" align="center">
            <template #default="{ $index }">
              <el-button link type="danger" @click="removeItem($index)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="table-footer">
          <span>小计：<span class="money">¥ {{ money(subtotal) }}</span></span>
          <span v-if="form.tax_rate">税金：<span class="money">¥ {{ money(taxAmount) }}</span></span>
          <span>合计：<span class="money pos" style="font-size: 16px">¥ {{ money(total) }}</span></span>
        </div>

        <el-form-item label="备注" style="margin-top: 8px">
          <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="工期、付款方式、有效期等" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type PriceItem, type Quotation } from '../api'
import { money, today } from '../utils/format'

interface ItemRow {
  room: string
  name: string
  spec: string
  unit: string
  quantity: number
  unit_price: number
  path: string[]
}

const list = ref<Quotation[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = ref(false)
const editing = ref(false)
const materialPrices = ref<PriceItem[]>([])
const laborAll = ref<PriceItem[]>([])
const laborOnly = ref<PriceItem[]>([])

const form = reactive<Partial<Quotation> & { items: ItemRow[] }>({
  project_id: 0,
  project_name: '',
  contract_type: '包工包料',
  quote_no: '',
  quote_date: today(),
  client: '',
  title: '报价单',
  tax_rate: 0,
  remark: '',
  items: [],
})

// 三套价目：材料 / 包工包料 / 包工；第一级在级联里显示，但名称只记「分类/条目」
const priceSources = computed(() => [
  { key: 'material', label: '材料', items: materialPrices.value },
  { key: 'labor_all', label: '包工包料', items: laborAll.value },
  { key: 'labor_only', label: '包工', items: laborOnly.value },
])

function priceOf(p: PriceItem, kind: string): number {
  if (kind === 'material') return p.price
  // 同行报价取 price，客户报价取 client_price（没填就退回 price）
  if (form.quote_type === '同行报价') return p.price
  return p.client_price || p.price
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

// 由「分类/条目」反查它属于哪套价目（用于编辑回填级联路径）
function pathOf(name: string): string[] {
  if (!name.includes('/')) return []
  const [cat, nm] = name.split('/')
  for (const src of priceSources.value) {
    const hit = src.items.find((p) => (p.category || '未分类') === cat && p.name === nm)
    if (hit) return [src.key, cat, nm]
  }
  return []
}

const subtotal = computed(() =>
  Number(form.items.reduce((s, it) => s + Number(it.quantity || 0) * Number(it.unit_price || 0), 0).toFixed(2)),
)
const taxAmount = computed(() => Number(((subtotal.value * (form.tax_rate || 0)) / 100).toFixed(2)))
const total = computed(() => Number((subtotal.value + taxAmount.value).toFixed(2)))

function emptyRow(): ItemRow {
  return { room: '', name: '', spec: '', unit: '', quantity: 1, unit_price: 0, path: [] }
}

// 尺寸支持简单算式：5*6=30、5+6=11，算出的数直接填到数量
function parseSize(input: string): number | null {
  if (!input) return null
  const expr = input.replace(/×/g, '*').replace(/÷/g, '/').replace(/\s+/g, '')
  if (!/^[0-9.+\-*/()]+$/.test(expr)) return null
  if (/[+\-*/.]$/.test(expr)) return null
  try {
    // eslint-disable-next-line no-new-func
    const v = Function('"use strict";return (' + expr + ')')()
    if (typeof v === 'number' && isFinite(v) && v > 0) return Math.round(v * 1000) / 1000
  } catch {
    /* ignore */
  }
  return null
}

function onSizeChange(row: ItemRow) {
  const v = parseSize(row.spec)
  if (v !== null) {
    row.quantity = v
    ElMessage.success(`尺寸计算：${row.spec} = ${v}`)
  }
}

function onPickItem(row: ItemRow, val: string[] | null) {
  if (!val || val.length < 3) {
    row.name = ''
    return
  }
  const [kind, cat, name] = val
  const src = priceSources.value.find((s2) => s2.key === kind)
  const p = src?.items.find((x) => (x.category || '未分类') === cat && x.name === name)
  if (!p) return
  row.name = `${cat}/${p.name}`
  row.spec = p.spec
  row.unit = p.unit
  row.unit_price = priceOf(p, kind)
}

function onQuoteTypeChange() {
  // 切换报价类型后，已选条目按新口径重算单价（材料价不受影响）
  for (const it of form.items) {
    if (!it.path || it.path.length < 3) continue
    const [kind, cat, name] = it.path
    const src = priceSources.value.find((s2) => s2.key === kind)
    const p = src?.items.find((x) => (x.category || '未分类') === cat && x.name === name)
    if (p) it.unit_price = priceOf(p, kind)
  }
}

function onContractTypeChange() {
  // 切换承包方式后，已选条目按新价目重匹配；匹配不到就保留原名称
  for (const it of form.items) {
    if (!it.name) continue
    const [cat, name] = it.name.split('/')
    const p = currentPrices.value.find((x) => (x.category || '未分类') === cat && x.name === name)
    if (p) it.path = [cat, p.name]
  }
}

async function loadPrices() {
  try {
    ;[materialPrices.value, laborAll.value, laborOnly.value] = await Promise.all([
      api.priceItems('material'),
      api.priceItems('labor_all'),
      api.priceItems('labor_only'),
    ])
  } catch {
    /* ignore */
  }
}

async function load() {
  loading.value = true
  try {
    list.value = await api.quotations()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function genNo(): string {
  const d = new Date()
  const p = (x: number) => String(x).padStart(2, '0')
  return `BJ${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}`
}

function openCreate() {
  editing.value = false
  Object.assign(form, {
    id: undefined,
    project_id: 0,
    project_name: '',
    contract_type: '包工包料',
    quote_type: '客户报价',
    quote_no: genNo(),
    quote_date: today(),
    client: '',
    title: '报价单',
    tax_rate: 0,
    remark: '',
    items: [emptyRow()],
  })
  dialog.value = true
}

async function openEdit(row: Quotation) {
  try {
    const full = await api.quotation(row.id)
    editing.value = true
    Object.assign(form, {
      ...full,
      contract_type: full.contract_type || '包工包料',
      quote_type: full.quote_type || '客户报价',
      items: full.items.map((it) => ({
        room: it.room || '',
        name: it.name,
        spec: it.spec,
        unit: it.unit,
        quantity: it.quantity,
        unit_price: it.unit_price,
        path: pathOf(it.name),
      })),
    })
    dialog.value = true
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

function addItem() {
  form.items.push(emptyRow())
}

function removeItem(i: number) {
  form.items.splice(i, 1)
}

async function save() {
  if (!form.project_name?.trim()) {
    ElMessage.warning('请填写项目名称')
    return
  }
  const items = form.items.filter((it) => it.name?.trim())
  if (items.length === 0) {
    ElMessage.warning('请至少选择一条报价明细')
    return
  }
  saving.value = true
  try {
    const payload = {
      ...form,
      project_id: 0,
      subtotal: subtotal.value,
      tax_amount: taxAmount.value,
      total: total.value,
      items: items.map((it, i) => ({
        room: it.room,
        name: it.name,
        spec: it.spec,
        unit: it.unit,
        quantity: it.quantity,
        unit_price: it.unit_price,
        amount: Number((Number(it.quantity || 0) * Number(it.unit_price || 0)).toFixed(2)),
        sort_order: i,
      })),
    }
    if (editing.value && form.id) {
      await api.updateQuotation(form.id, payload)
    } else {
      await api.createQuotation(payload)
    }
    ElMessage.success('已保存')
    dialog.value = false
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function remove(row: Quotation) {
  try {
    await ElMessageBox.confirm(`确定删除报价单「${row.quote_no || row.id}」？`, '删除确认', { type: 'warning' })
  } catch {
    return
  }
  try {
    await api.deleteQuotation(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

function exportExcel(row: Quotation) {
  window.open(`/api/quotations/${row.id}/excel`, '_blank')
}

onMounted(async () => {
  await loadPrices()
  await load()
})
</script>
