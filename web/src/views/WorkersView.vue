<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">工人管理</h2>
        <div class="page-sub">维护工人档案与默认单价，记工时自动带出</div>
      </div>
      <el-button type="primary" @click="openCreate">
        <el-icon><Plus /></el-icon>添加工人
      </el-button>
    </div>

    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索姓名 / 电话" clearable style="width: 240px" @change="load">
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
    </div>

    <div class="card">
      <el-table :data="list" v-loading="loading" empty-text="还没有工人">
        <el-table-column prop="name" label="姓名" width="120" />
        <el-table-column prop="phone" label="电话" width="150" />
        <el-table-column prop="id_card" label="身份证号" min-width="180" show-overflow-tooltip />
        <el-table-column label="内部单价" width="120" align="right">
          <template #default="{ row }"><span class="money">{{ money(row.internal_rate) }}</span></template>
        </el-table-column>
        <el-table-column label="对外单价" width="120" align="right">
          <template #default="{ row }"><span class="money">{{ money(row.external_rate) }}</span></template>
        </el-table-column>
        <el-table-column label="加班单价/时" width="130" align="right">
          <template #default="{ row }"><span class="money">{{ money(row.overtime_rate) }}</span></template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip />
        <el-table-column label="操作" width="150" align="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialog" :title="editing ? '编辑工人' : '添加工人'" width="520px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="姓名" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="电话">
          <el-input v-model="form.phone" />
        </el-form-item>
        <el-form-item label="身份证号">
          <el-input v-model="form.id_card" />
        </el-form-item>
        <el-form-item label="内部单价">
          <el-input-number v-model="form.internal_rate" :min="0" :precision="2" :step="10" style="width: 100%" />
        </el-form-item>
        <el-form-item label="对外单价">
          <el-input-number v-model="form.external_rate" :min="0" :precision="2" :step="10" style="width: 100%" />
        </el-form-item>
        <el-form-item label="加班单价">
          <el-input-number v-model="form.overtime_rate" :min="0" :precision="2" :step="5" style="width: 100%" />
          <div class="form-tip">元/小时，记「加班」时自动带出</div>
        </el-form-item>
        <el-form-item>
          <span class="form-tip">内部单价用于记工与结算成本；对外单价用于报价单，报给客户。</span>
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
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type Worker } from '../api'
import { money } from '../utils/format'

const list = ref<Worker[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = ref(false)
const editing = ref(false)
const keyword = ref('')

const form = reactive<Partial<Worker>>({ name: '', phone: '', id_card: '', internal_rate: 0, external_rate: 0, overtime_rate: 0, remark: '' })

async function load() {
  loading.value = true
  try {
    list.value = await api.workers(keyword.value)
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = false
  Object.assign(form, { id: undefined, name: '', phone: '', id_card: '', internal_rate: 0, external_rate: 0, overtime_rate: 0, remark: '' })
  dialog.value = true
}

function openEdit(row: Worker) {
  editing.value = true
  Object.assign(form, row)
  dialog.value = true
}

async function save() {
  if (!form.name?.trim()) {
    ElMessage.warning('请填写姓名')
    return
  }
  saving.value = true
  try {
    if (editing.value && form.id) {
      await api.updateWorker(form.id, form)
    } else {
      await api.createWorker(form)
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

async function remove(row: Worker) {
  try {
    await ElMessageBox.confirm(`确定删除工人「${row.name}」？`, '删除确认', { type: 'warning' })
  } catch {
    return
  }
  try {
    await api.deleteWorker(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

onMounted(load)
</script>
