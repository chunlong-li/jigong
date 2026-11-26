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
        <span>累计应付：<span class="money">¥ {{ money(totalPayable) }}</span></span>
        <span>累计实收：<span class="money pos">¥ {{ money(totalReceived) }}</span></span>
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
        <el-table-column label="结算单" width="90" align="center">
          <template #default="{ row }">{{ row.settle_count }}</template>
        </el-table-column>
        <el-table-column label="应付" width="110" align="right">
          <template #default="{ row }">{{ money(row.payable) }}</template>
        </el-table-column>
        <el-table-column label="实收" width="110" align="right">
          <template #default="{ row }">{{ money(row.received) }}</template>
        </el-table-column>
        <el-table-column label="未结款" width="120" align="right">
          <template #default="{ row }">
            <span class="money" :class="row.outstanding > 0 ? 'warn' : ''">{{ money(row.outstanding) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" align="right">
          <template #default="{ row }">
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
const totalPayable = computed(() => Number(list.value.reduce((s, c) => s + (c.payable || 0), 0).toFixed(2)))
const totalReceived = computed(() => Number(list.value.reduce((s, c) => s + (c.received || 0), 0).toFixed(2)))

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
