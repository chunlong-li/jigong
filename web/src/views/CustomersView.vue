<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">客户</h2>
        <div class="page-sub">客户信息与未结款统计</div>
      </div>
      <el-button type="primary" @click="openCreate">
        <el-icon><Plus /></el-icon>
        <span>新增客户</span>
      </el-button>
    </div>

    <div class="card">
      <div class="stat-row">
        <span>客户数：<b>{{ list.length }}</b></span>
        <span>未结款合计：<b class="money warn">¥ {{ money(totalOutstanding) }}</b></span>
        <span>未结单数：<b>{{ totalOpen }}</b></span>
        <span>未结应付：<span class="money">¥ {{ money(totalOpenPayable) }}</span></span>
        <span>未结已收：<span class="money pos">¥ {{ money(totalOpenReceived) }}</span></span>
      </div>

      <el-table :data="list" v-loading="loading" empty-text="还没有客户，点右上角新增">
        <el-table-column prop="name" label="客户名称" min-width="150" show-overflow-tooltip />
        <el-table-column label="分类" width="110" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.category" size="small" effect="plain">{{ row.category }}</el-tag>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column prop="contact" label="联系人" width="110" show-overflow-tooltip />
        <el-table-column prop="phone" label="电话" width="130" show-overflow-tooltip />
        <el-table-column prop="address" label="地址" min-width="180" show-overflow-tooltip />
        <el-table-column label="结算单" width="80" align="center">
          <template #default="{ row }">{{ row.settle_count }}</template>
        </el-table-column>
        <el-table-column label="未结单" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.open_count > 0" size="small" type="warning" effect="plain">{{ row.open_count }}</el-tag>
            <span v-else class="muted">0</span>
          </template>
        </el-table-column>
        <el-table-column label="未结应付" width="115" align="right">
          <template #default="{ row }">{{ money(row.open_payable) }}</template>
        </el-table-column>
        <el-table-column label="未结已收" width="115" align="right">
          <template #default="{ row }">{{ money(row.open_received) }}</template>
        </el-table-column>
        <el-table-column label="未结款" width="120" align="right">
          <template #default="{ row }">
            <span class="money" :class="row.outstanding > 0 ? 'warn' : ''">{{ money(row.outstanding) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" align="right">
          <template #default="{ row }">
            <el-button link type="success" :disabled="!(row.outstanding > 0)" @click="openReceive(row)">已收</el-button>
            <el-button link type="primary" @click="openPayments(row)">记录</el-button>
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialog" :title="editing ? '编辑客户' : '新增客户'" width="560px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="客户名称" required>
          <el-input v-model="form.name" placeholder="如：宏远地产 / 张先生" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select
            v-model="form.category"
            filterable
            allow-create
            clearable
            default-first-option
            placeholder="选择或直接输入新分类"
            style="width: 100%"
          >
            <el-option v-for="c in categories" :key="c" :label="c" :value="c" />
          </el-select>
        </el-form-item>
        <el-form-item label="联系人">
          <el-input v-model="form.contact" />
        </el-form-item>
        <el-form-item label="电话">
          <el-input v-model="form.phone" />
        </el-form-item>
        <el-form-item label="地址">
          <el-input v-model="form.address" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
    <el-dialog v-model="receiveDialog" title="登记已收（收款）" width="460px">
      <el-descriptions :column="1" border size="small" style="margin-bottom: 14px">
        <el-descriptions-item label="客户">{{ receiveRow?.name }}</el-descriptions-item>
        <el-descriptions-item label="未结应付 / 已收">
          <span class="money">¥ {{ money(receiveRow?.open_payable) }}</span>
          <span class="muted"> / </span>
          <span class="money pos">¥ {{ money(receiveRow?.open_received) }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="当前未结款">
          <span class="money warn">¥ {{ money(receiveRow?.outstanding) }}</span>
        </el-descriptions-item>
      </el-descriptions>
      <el-form label-width="90px">
        <el-form-item label="本次收款">
          <el-input-number v-model="receiveAmount" :min="0" :precision="2" style="width: 100%" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="receiveNote" placeholder="如：微信转账 / 现金" />
        </el-form-item>
      </el-form>
      <div class="form-tip">
        收款后按最早未结的结算单依次冲抵已收金额，未结款自动扣减。
      </div>
      <template #footer>
        <el-button @click="receiveDialog = false">取消</el-button>
        <el-button type="primary" :loading="receiving" @click="saveReceive">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="paymentsDialog" :title="'收款记录 · ' + (paymentsRow?.name || '')" width="480px">
      <el-table :data="payments" size="small" border empty-text="还没有收款记录">
        <el-table-column prop="paid_at" label="时间" width="170" />
        <el-table-column label="金额" width="110" align="right">
          <template #default="{ row }"><span class="money pos">{{ money(row.amount) }}</span></template>
        </el-table-column>
        <el-table-column prop="note" label="备注" show-overflow-tooltip />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { api, type Customer } from '../api'
import { money } from '../utils/format'

const list = ref<Customer[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = ref(false)
const editing = ref(false)

const form = reactive<Partial<Customer>>({
  name: '',
  category: '',
  contact: '',
  phone: '',
  address: '',
  remark: '',
})

const categories = computed(() => [...new Set(list.value.map((c) => c.category).filter(Boolean))])
const totalOutstanding = computed(() => Number(list.value.reduce((s, c) => s + (c.outstanding || 0), 0).toFixed(2)))
const totalOpen = computed(() => list.value.reduce((s, c) => s + (c.open_count || 0), 0))
const totalOpenPayable = computed(() => Number(list.value.reduce((s, c) => s + (c.open_payable || 0), 0).toFixed(2)))
const totalOpenReceived = computed(() => Number(list.value.reduce((s, c) => s + (c.open_received || 0), 0).toFixed(2)))

async function load() {
  loading.value = true
  try {
    list.value = await api.customers()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = false
  Object.assign(form, { id: undefined, name: '', category: '', contact: '', phone: '', address: '', remark: '' })
  dialog.value = true
}

function openEdit(row: Customer) {
  editing.value = true
  Object.assign(form, {
    id: row.id,
    name: row.name,
    category: row.category,
    contact: row.contact,
    phone: row.phone,
    address: row.address,
    remark: row.remark,
  })
  dialog.value = true
}

async function save() {
  if (!form.name?.trim()) {
    ElMessage.warning('请填写客户名称')
    return
  }
  saving.value = true
  try {
    if (editing.value && form.id) {
      await api.updateCustomer(form.id, form)
      ElMessage.success('客户已更新')
    } else {
      await api.createCustomer(form)
      ElMessage.success('客户已新增')
    }
    dialog.value = false
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function remove(row: Customer) {
  try {
    await ElMessageBox.confirm(`确定删除客户「${row.name}」？`, '删除确认', { type: 'warning' })
  } catch {
    return
  }
  try {
    await api.deleteCustomer(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

const receiveDialog = ref(false)
const receiveRow = ref<Customer | null>(null)
const receiveAmount = ref(0)
const receiveNote = ref('')
const receiving = ref(false)

const paymentsDialog = ref(false)
const paymentsRow = ref<Customer | null>(null)
const payments = ref<{ id: number; amount: number; note: string; paid_at: string }[]>([])

function openReceive(row: Customer) {
  receiveRow.value = row
  receiveAmount.value = Number(row.outstanding || 0)
  receiveNote.value = ''
  receiveDialog.value = true
}

async function saveReceive() {
  if (!receiveRow.value) return
  if (!receiveAmount.value || receiveAmount.value <= 0) {
    ElMessage.warning('请填写收款金额')
    return
  }
  receiving.value = true
  try {
    const r = await api.receivePayment(receiveRow.value.id, receiveAmount.value, receiveNote.value)
    const detail = (r.allocations || []).map((a) => `${a.settle_no || '结算单'} ¥${money(a.amount)}`).join('，')
    ElMessage.success(`已收 ¥${money(r.amount)}${detail ? '（' + detail + '）' : ''}，剩余未结 ¥${money(r.outstanding)}`)
    receiveDialog.value = false
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    receiving.value = false
  }
}

async function openPayments(row: Customer) {
  paymentsRow.value = row
  payments.value = []
  paymentsDialog.value = true
  try {
    payments.value = await api.customerPayments(row.id)
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

onMounted(load)
</script>

<style scoped>
.stat-row {
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
  margin-bottom: 12px;
  font-size: 13px;
  color: #606266;
}
.stat-row b {
  color: #303133;
}
</style>
