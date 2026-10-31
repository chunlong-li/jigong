<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">记工</h2>
        <div class="page-sub">同一工人在同一工地合并为一条，按总天数统计；加班按小时单独记</div>
      </div>
      <el-button type="primary" @click="openCreate">
        <el-icon><Plus /></el-icon>记工
      </el-button>
    </div>

    <div class="toolbar">
      <el-select v-model="filter.project_id" placeholder="全部工地" clearable style="width: 190px" @change="load">
        <el-option v-for="p in projects" :key="p.id" :label="p.name" :value="String(p.id)" />
      </el-select>
      <el-select v-model="filter.worker_id" placeholder="全部工人" clearable filterable style="width: 150px" @change="load">
        <el-option v-for="w in workers" :key="w.id" :label="w.name" :value="String(w.id)" />
      </el-select>
      <el-date-picker
        v-model="range"
        type="daterange"
        value-format="YYYY-MM-DD"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        style="width: 250px"
        @change="load"
      />
      <el-button @click="load">刷新</el-button>
      <el-button style="margin-left: auto" @click="openHistory">结算记录</el-button>
      <el-radio-group v-model="groupBy" @change="load">
        <el-radio-button value="project">按工地查看</el-radio-button>
        <el-radio-button value="month">按月查看</el-radio-button>
      </el-radio-group>
    </div>

    <div class="card">
      <el-table :data="groups" v-loading="loading" empty-text="还没有记工记录">
        <el-table-column v-if="groupBy === 'month'" prop="month" label="月份" width="95" />
        <el-table-column prop="project_name" label="工地" min-width="170" show-overflow-tooltip />
        <el-table-column prop="worker_name" label="工人" width="100" />
        <el-table-column label="总天数" width="100" align="right">
          <template #default="{ row }"><span class="money">{{ qty(row.days) }} 天</span></template>
        </el-table-column>
        <el-table-column label="加班" width="150" align="right">
          <template #default="{ row }">
            <span v-if="row.overtime_count">{{ row.overtime_count }} 次 / {{ qty(row.overtime_hours) }} 时</span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="按时 / 计件" width="140" align="right">
          <template #default="{ row }">
            <template v-if="row.hour_qty || row.piece_qty">
              <span v-if="row.hour_qty">{{ qty(row.hour_qty) }} 时 </span>
              <span v-if="row.piece_qty">{{ qty(row.piece_qty) }} 件</span>
            </template>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="金额" width="120" align="right">
          <template #default="{ row }"><span class="money">{{ money(row.amount) }}</span></template>
        </el-table-column>
        <el-table-column prop="last_date" label="最近" width="105" />
        <el-table-column label="操作" width="130" align="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openView(row)">查看</el-button>
            <el-button link type="warning" @click="settle(row)">结算</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-footer">
        <span>人数：{{ groups.length }}</span>
        <span>合计天数：{{ qty(totalDays) }} 天</span>
        <span>合计金额：<span class="money pos">¥ {{ money(totalAmount) }}</span></span>
      </div>
    </div>

    <!-- 记工 -->
    <el-dialog v-model="dialog" title="记工" width="600px">
      <el-form :model="form" label-width="96px">
        <el-form-item label="工地" required>
          <el-select v-model="form.project_id" style="width: 100%" placeholder="选择工地">
            <el-option v-for="p in projects" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="工人" required>
          <el-select
            v-model="form.worker_ids"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            style="width: 100%"
            placeholder="可多选"
          >
            <el-option
              v-for="w in workers"
              :key="w.id"
              :label="w.name + (w.internal_rate ? '（日薪 ' + money(w.internal_rate) + '）' : '')"
              :value="w.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="计价方式">
          <el-radio-group v-model="form.mode">
            <el-radio-button value="day">按天</el-radio-button>
            <el-radio-button value="hour">按时</el-radio-button>
            <el-radio-button value="piece">计件</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <template v-if="form.mode === 'day'">
          <el-form-item label="记工类型">
            <el-radio-group v-model="form.day_type">
              <el-radio-button value="full">一天</el-radio-button>
              <el-radio-button value="half">半天</el-radio-button>
              <el-radio-button value="overtime">加班</el-radio-button>
            </el-radio-group>
            <div class="form-tip">一天 / 半天计入总天数；加班按小时单独统计，不计天数</div>
          </el-form-item>
          <template v-if="form.day_type === 'overtime'">
            <el-form-item label="加班计算">
              <el-radio-group v-model="form.overtime_mode">
                <el-radio-button value="hours">计时</el-radio-button>
                <el-radio-button value="full">一天</el-radio-button>
                <el-radio-button value="half">半天</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="form.overtime_mode === 'hours'" label="加班时长">
              <el-input-number v-model="form.hours" :min="0.5" :step="0.5" :precision="1" style="width: 160px" />
              <span class="muted" style="margin-left: 8px">小时 × 加班单价</span>
            </el-form-item>
            <el-form-item v-else>
              <span class="form-tip">
                按{{ form.overtime_mode === 'full' ? '一天（1 天）' : '半天（0.5 天）' }}计入总天数，用日薪计算
              </span>
            </el-form-item>
          </template>
        </template>

        <el-form-item v-else :label="'数量(' + (form.mode === 'hour' ? '小时' : '件') + ')'">
          <el-input-number v-model="form.quantity" :min="0" :precision="3" :step="1" style="width: 100%" />
        </el-form-item>

        <el-form-item label="日期" required>
          <el-date-picker
            v-model="form.dates"
            type="dates"
            value-format="YYYY-MM-DD"
            placeholder="可多选日期，点一次加一天"
            style="width: 100%"
          />
          <div class="form-tip">一天 / 半天互斥（同一天再记会覆盖）；加班可与一天 / 半天叠加</div>
        </el-form-item>

        <el-form-item label="单价(元)">
          <el-input-number v-model="form.unit_price" :min="0" :precision="2" :step="10" style="width: 100%" />
          <div class="form-tip">{{ priceHint }}</div>
        </el-form-item>

        <el-form-item label="工作内容">
          <el-input v-model="form.content" placeholder="如：砌墙、扎钢筋、浇混凝土" />
        </el-form-item>
      </el-form>

      <el-alert type="info" :closable="false" show-icon>
        将写入 <b>{{ form.worker_ids.length }}</b> 名工人 × <b>{{ form.dates.length }}</b> 天 =
        <b>{{ form.worker_ids.length * form.dates.length }}</b> 条记录
      </el-alert>

      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <!-- 查看 -->
    <el-dialog v-model="viewDialog" :title="viewTitle" width="780px" top="5vh">
      <el-calendar v-model="calDate">
        <template #date-cell="{ data }">
          <div class="cal-cell">
            <span>{{ data.day.split('-')[2] }}</span>
            <span class="cal-marks">
              <i v-for="t in marksOn(data.day)" :key="t" class="mark" :class="'mark-' + t"></i>
            </span>
          </div>
        </template>
      </el-calendar>

      <div class="legend">
        <span><i class="mark mark-full"></i>一天</span>
        <span><i class="mark mark-half"></i>半天</span>
        <span><i class="mark mark-overtime"></i>加班</span>
        <span><i class="mark mark-other"></i>按时/计件</span>
      </div>

      <el-table :data="viewRecords" size="small" border max-height="280" style="margin-top: 10px">
        <el-table-column prop="work_date" label="日期" width="105" />
        <el-table-column label="类型" width="80">
          <template #default="{ row }">{{ typeText(row) }}</template>
        </el-table-column>
        <el-table-column label="数量" width="100" align="right">
          <template #default="{ row }">{{ qty(row.quantity) }} {{ unitText(row) }}</template>
        </el-table-column>
        <el-table-column label="单价" width="95" align="right">
          <template #default="{ row }">{{ money(row.unit_price) }}</template>
        </el-table-column>
        <el-table-column label="金额" width="105" align="right">
          <template #default="{ row }"><span class="money">{{ money(row.amount) }}</span></template>
        </el-table-column>
        <el-table-column prop="content" label="内容" min-width="110" show-overflow-tooltip />
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.settled_at ? 'info' : 'success'" size="small">
              {{ row.settled_at ? '已结算' : '未结算' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="" width="70" align="center">
          <template #default="{ row }">
            <el-button link type="danger" @click="removeRecord(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="table-footer">
        <span>记录：{{ viewRecords.length }} 条</span>
        <span>金额合计：<span class="money">¥ {{ money(viewTotal) }}</span></span>
      </div>
    </el-dialog>

    <!-- 结算记录 -->
    <el-dialog v-model="historyDialog" title="结算记录（已归档）" width="880px" top="6vh">
      <el-table :data="historyRows" size="small" border max-height="460" empty-text="还没有结算记录">
        <el-table-column prop="created_at" label="结算时间" width="150" />
        <el-table-column prop="project_name" label="工地" min-width="140" show-overflow-tooltip />
        <el-table-column prop="worker_name" label="工人" width="90" />
        <el-table-column label="期间" width="190">
          <template #default="{ row }">{{ row.period_start }} 至 {{ row.period_end }}</template>
        </el-table-column>
        <el-table-column label="天数" width="80" align="right">
          <template #default="{ row }">{{ qty(row.days) }}</template>
        </el-table-column>
        <el-table-column label="加班" width="110" align="right">
          <template #default="{ row }">{{ row.overtime_count }} 次 / {{ qty(row.overtime_hours) }} 时</template>
        </el-table-column>
        <el-table-column label="金额" width="110" align="right">
          <template #default="{ row }"><span class="money">{{ money(row.amount) }}</span></template>
        </el-table-column>
        <el-table-column label="" width="100" align="center">
          <template #default="{ row }">
            <el-button link type="primary" @click="printSlip(row)">工资单</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type Project, type WageSettlement, type WorkLog, type WorkLogGroup, type Worker, type WorkMode } from '../api'
import { money, qty, today } from '../utils/format'

const groups = ref<WorkLogGroup[]>([])
const projects = ref<Project[]>([])
const workers = ref<Worker[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = ref(false)
const range = ref<[string, string] | null>(null)
const filter = reactive({ project_id: '', worker_id: '' })
const groupBy = ref<'project' | 'month'>('project')

const form = reactive({
  project_id: undefined as number | undefined,
  worker_ids: [] as number[],
  mode: 'day' as WorkMode,
  day_type: 'full',
  overtime_mode: 'hours',
  quantity: 1,
  hours: 2,
  unit_price: 0,
  dates: [today()] as string[],
  content: '',
})

const totalAmount = computed(() => groups.value.reduce((s, g) => s + g.amount, 0))
const totalDays = computed(() => groups.value.reduce((s, g) => s + g.days, 0))

const priceHint = computed(() => {
  if (form.mode !== 'day') return '填 0 表示按每名工人自己的内部单价分别计算'
  if (form.day_type === 'overtime' && form.overtime_mode === 'hours')
    return '填 0 表示按每名工人档案里的加班单价（元/小时）计算'
  if (form.day_type === 'overtime') return '加班按一天/半天计入天数，填 0 按工人日薪计算'
  return '填 0 表示按每名工人自己的日薪（内部单价）计算'
})

async function load() {
  loading.value = true
  try {
    const params: Record<string, string> = {}
    if (filter.project_id) params.project_id = filter.project_id
    if (filter.worker_id) params.worker_id = filter.worker_id
    if (range.value?.[0]) params.from = range.value[0]
    if (range.value?.[1]) params.to = range.value[1]
    if (groupBy.value === 'month') params.group_by = 'month'
    groups.value = await api.groupedWorkLogs(params)
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, {
    project_id: filter.project_id ? Number(filter.project_id) : (projects.value[0]?.id ?? undefined),
    worker_ids: filter.worker_id ? [Number(filter.worker_id)] : [],
    mode: 'day' as WorkMode,
    day_type: 'full',
    overtime_mode: 'hours',
    quantity: 1,
    hours: 2,
    unit_price: 0,
    dates: [today()],
    content: '',
  })
  dialog.value = true
}

async function save() {
  if (!form.project_id) {
    ElMessage.warning('请选择工地')
    return
  }
  if (!form.worker_ids.length) {
    ElMessage.warning('请至少选择一名工人')
    return
  }
  if (!form.dates.length) {
    ElMessage.warning('请至少选择一个日期')
    return
  }
  if (
    form.mode === 'day' &&
    form.day_type === 'overtime' &&
    form.overtime_mode === 'hours' &&
    (!form.hours || form.hours <= 0)
  ) {
    ElMessage.warning('加班请填写小时数')
    return
  }
  if (form.mode !== 'day' && (!form.quantity || form.quantity <= 0)) {
    ElMessage.warning('数量必须大于 0')
    return
  }
  saving.value = true
  try {
    let dayType = form.mode === 'day' ? form.day_type : ''
    if (dayType === 'overtime') {
      if (form.overtime_mode === 'full') dayType = 'full'
      else if (form.overtime_mode === 'half') dayType = 'half'
    }
    const res = await api.batchWorkLogs({
      project_id: form.project_id,
      worker_ids: form.worker_ids,
      dates: form.dates,
      mode: form.mode,
      day_type: dayType,
      quantity: form.quantity,
      hours: form.hours,
      unit_price: form.unit_price,
      content: form.content,
    })
    ElMessage.success(`已写入 ${res.created} 条记工`)
    dialog.value = false
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

function lastDayOfMonth(month: string): string {
  const [y, m] = month.split('-').map(Number)
  const d = new Date(y, m, 0).getDate()
  return `${month}-${String(d).padStart(2, '0')}`
}

async function settle(g: WorkLogGroup) {
  const label = g.month ? `${g.month} · ${g.worker_name}` : g.worker_name
  let action: 'print' | 'only' | 'close' = 'close'
  try {
    await ElMessageBox.confirm(
      `结算「${label} · ${g.project_name}」：当前 ${qty(g.days)} 天 / ¥${money(g.amount)} 将归档冻结，` +
        '之后的新记工重新累计。是否打印工资单？',
      '结算确认',
      {
        distinguishCancelAndClose: true,
        confirmButtonText: '打印并结算',
        cancelButtonText: '仅结算',
        type: 'warning',
      },
    )
    action = 'print'
  } catch (a: any) {
    action = a === 'cancel' ? 'only' : 'close'
  }
  if (action === 'close') return

  try {
    const through = groupBy.value === 'month' && g.month ? lastDayOfMonth(g.month) : undefined
    const res = through
      ? await api.settleWorkLogThrough(g.project_id, g.worker_id, through)
      : await api.settleWorkLog(g.project_id, g.worker_id)
    ElMessage.success(`已结算归档：${res.days} 天 / ¥${money(res.amount)}`)
    if (action === 'print') {
      window.open(`/api/wage-settlements/${res.id}/excel`, '_blank')
    }
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

// ---- 结算记录 ----
const historyDialog = ref(false)
const historyRows = ref<WageSettlement[]>([])

async function openHistory() {
  try {
    historyRows.value = await api.wageSettlements()
    historyDialog.value = true
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

function printSlip(row: WageSettlement) {
  window.open(`/api/wage-settlements/${row.id}/excel`, '_blank')
}

// ---- 查看 ----
const viewDialog = ref(false)
const viewGroup = ref<WorkLogGroup | null>(null)
const viewRecords = ref<WorkLog[]>([])
const calDate = ref(new Date())

const viewTitle = computed(() =>
  viewGroup.value ? `${viewGroup.value.worker_name} · ${viewGroup.value.project_name}` : '记工明细',
)
const viewTotal = computed(() => viewRecords.value.reduce((s, r) => s + r.amount, 0))

async function openView(g: WorkLogGroup) {
  viewGroup.value = g
  viewDialog.value = true
  calDate.value = new Date()
  await reloadView()
}

async function reloadView() {
  const g = viewGroup.value
  if (!g) return
  try {
    viewRecords.value = await api.worklogs({ project_id: String(g.project_id), worker_id: String(g.worker_id) })
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

function marksOn(day: string): string[] {
  const set = new Set<string>()
  for (const r of viewRecords.value) {
    if (r.work_date !== day) continue
    if (r.mode === 'day') {
      set.add(r.day_type === 'half' ? 'half' : r.day_type === 'overtime' ? 'overtime' : 'full')
    } else {
      set.add('other')
    }
  }
  return [...set]
}

function typeText(r: WorkLog): string {
  if (r.mode === 'day') {
    if (r.day_type === 'half') return '半天'
    if (r.day_type === 'overtime') return '加班'
    return '一天'
  }
  return r.mode === 'hour' ? '按时' : '计件'
}

function unitText(r: WorkLog): string {
  if (r.mode === 'day') return r.day_type === 'overtime' ? '小时' : '天'
  return r.mode === 'hour' ? '小时' : '件'
}

async function removeRecord(r: WorkLog) {
  try {
    await ElMessageBox.confirm(
      `确定删除 ${r.work_date} 的这条${typeText(r)}记录？`,
      '删除确认',
      { type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await api.deleteWorkLog(r.id)
    ElMessage.success('已删除')
    await reloadView()
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

onMounted(async () => {
  try {
    ;[projects.value, workers.value] = await Promise.all([api.projects(), api.workers()])
  } catch {
    /* ignore */
  }
  await load()
})
</script>

<style scoped>
.cal-cell {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  gap: 3px;
  font-size: 12px;
}

.cal-marks {
  display: flex;
  gap: 3px;
}

.mark {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  display: inline-block;
}

.mark-full {
  background: #2f6bff;
}
.mark-half {
  background: #f0a020;
}
.mark-overtime {
  background: #f56c6c;
}
.mark-other {
  background: #909399;
}

.legend {
  display: flex;
  gap: 18px;
  font-size: 12px;
  color: #6b7280;
  margin-top: 8px;
}

.legend span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
</style>
