<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">工地项目</h2>
        <div class="page-sub">管理工地、承包方式与合同金额；状态在列表里直接操作</div>
      </div>
      <el-button type="primary" @click="openCreate">
        <el-icon><Plus /></el-icon>新建项目
      </el-button>
    </div>

    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索项目 / 客户 / 地址" clearable style="width: 260px" @change="load">
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
      <el-select v-model="status" placeholder="全部状态" clearable style="width: 140px" @change="load">
        <el-option label="在建" value="active" />
        <el-option label="已完工" value="done" />
        <el-option label="暂停" value="paused" />
      </el-select>
    </div>

    <div class="card">
      <el-table :data="list" v-loading="loading" empty-text="还没有项目，先新建一个">
        <el-table-column prop="name" label="项目名称" min-width="150" show-overflow-tooltip />
        <el-table-column prop="site_address" label="工地地址" min-width="160" show-overflow-tooltip />
        <el-table-column prop="client" label="客户" width="110" show-overflow-tooltip />
        <el-table-column label="承包方式" width="110" align="center">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ row.contract_type || '包工包料' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="start_date" label="开工日期" width="105" />
        <el-table-column label="合同金额" width="120" align="right">
          <template #default="{ row }"><span class="money">{{ money(row.contract_amount) }}</span></template>
        </el-table-column>
        <el-table-column label="状态" width="85" align="center">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="230" align="right">
          <template #default="{ row }">
            <el-button v-if="row.status !== 'active'" link type="success" @click="setStatus(row, 'active')">
              恢复
            </el-button>
            <el-button v-if="row.status === 'active'" link type="warning" @click="setStatus(row, 'paused')">
              暂停
            </el-button>
            <el-button v-if="row.status !== 'done'" link type="info" @click="setStatus(row, 'done')">完工</el-button>
            <el-button link type="success" @click="openMaterial(row)">进料</el-button>
            <el-button link type="primary" @click="openMaterialList(row)">查看</el-button>
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialog" :title="editing ? '编辑项目' : '新建项目'" width="560px">
      <el-form :model="form" label-width="92px">
        <el-form-item label="项目名称" required>
          <el-input v-model="form.name" placeholder="如：滨江花园 3 号楼" />
        </el-form-item>
        <el-form-item label="承包方式">
          <el-radio-group v-model="form.contract_type">
            <el-radio-button value="包工包料">包工包料</el-radio-button>
            <el-radio-button value="包工">包工</el-radio-button>
            <el-radio-button value="点工">点工</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="工地地址">
          <el-input v-model="form.site_address" placeholder="选填" />
        </el-form-item>
        <el-form-item label="客户">
          <el-input v-model="form.client" placeholder="甲方 / 业主名称" />
        </el-form-item>
        <el-form-item label="联系人">
          <el-input v-model="form.contact" placeholder="联系人及电话" />
        </el-form-item>
        <el-form-item label="合同金额">
          <el-input-number v-model="form.contract_amount" :min="0" :precision="2" :step="1000" style="width: 100%" />
        </el-form-item>
        <el-form-item label="开工日期">
          <el-date-picker v-model="form.start_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
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

    <!-- 登记进料 -->
    <el-dialog v-model="materialDialog" :title="'登记进料 · ' + (materialProject?.name || '')" width="560px">
      <el-form :model="material" label-width="92px">
        <el-form-item label="进料日期" required>
          <el-date-picker v-model="material.in_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select
            v-model="materialCategory"
            filterable
            allow-create
            default-first-option
            clearable
            style="width: 100%"
            placeholder="选择或输入分类（新材料会自动加入价格明细）"
            @change="onCategoryChange"
          >
            <el-option v-for="c in materialCategories" :key="c" :label="c" :value="c" />
          </el-select>
        </el-form-item>
        <el-form-item label="材料名称" required>
          <el-select
            v-model="material.name"
            filterable
            allow-create
            default-first-option
            style="width: 100%"
            placeholder="从价格明细选择，或直接输入新材料"
            @change="onPickMaterial"
          >
            <el-option
              v-for="p in materialOptions"
              :key="p.id"
              :label="p.name + (p.spec ? '（' + p.spec + '）' : '') + '  ¥' + money(p.price) + '/' + (p.unit || '单位')"
              :value="p.name"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="规格">
          <el-input v-model="material.spec" placeholder="如：P.O 42.5、HRB400 Φ12" />
        </el-form-item>
        <el-form-item label="单位">
          <el-input v-model="material.unit" placeholder="吨 / 方 / 米 / 件" style="width: 160px" />
        </el-form-item>
        <el-form-item label="数量" required>
          <el-input-number v-model="material.quantity" :min="0" :precision="3" :step="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="单价(元)">
          <el-input-number v-model="material.unit_price" :min="0" :precision="2" :step="10" style="width: 100%" />
        </el-form-item>
        <el-form-item label="金额">
          <div class="money" style="font-size: 18px; color: #2f6bff">
            ¥ {{ money(Number(material.quantity || 0) * Number(material.unit_price || 0)) }}
          </div>
        </el-form-item>
        <el-form-item label="供应商">
          <el-input v-model="material.supplier" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="material.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <div v-if="recentMaterials.length" style="margin-top: 4px">
        <div class="muted" style="font-size: 12px; margin-bottom: 6px">该工地最近进料</div>
        <el-table :data="recentMaterials" size="small" border max-height="190">
          <el-table-column prop="in_date" label="日期" width="100" />
          <el-table-column prop="name" label="材料" min-width="120" show-overflow-tooltip />
          <el-table-column label="数量" width="90" align="right">
            <template #default="{ row }">{{ qty(row.quantity) }} {{ row.unit }}</template>
          </el-table-column>
          <el-table-column label="金额" width="100" align="right">
            <template #default="{ row }"><span class="money">{{ money(row.amount) }}</span></template>
          </el-table-column>
        </el-table>
      </div>

      <template #footer>
        <el-button @click="materialDialog = false">取消</el-button>
        <el-button type="primary" :loading="materialSaving" @click="saveMaterial">保存</el-button>
      </template>
    </el-dialog>

    <!-- 进料明细（全部） -->
    <el-dialog
      v-model="listDialog"
      :title="'进料明细 · ' + (listProject?.name || '')"
      width="900px"
      top="6vh"
    >
      <el-table :data="materialList" v-loading="listLoading" size="small" border max-height="480" empty-text="该工地还没有进料记录">
        <el-table-column prop="in_date" label="日期" width="105" />
        <el-table-column prop="name" label="材料" min-width="130" show-overflow-tooltip />
        <el-table-column prop="spec" label="规格" width="140" show-overflow-tooltip />
        <el-table-column prop="unit" label="单位" width="70" align="center" />
        <el-table-column label="数量" width="95" align="right">
          <template #default="{ row }">{{ qty(row.quantity) }}</template>
        </el-table-column>
        <el-table-column label="单价" width="95" align="right">
          <template #default="{ row }">{{ money(row.unit_price) }}</template>
        </el-table-column>
        <el-table-column label="金额" width="110" align="right">
          <template #default="{ row }"><span class="money">{{ money(row.amount) }}</span></template>
        </el-table-column>
        <el-table-column prop="supplier" label="供应商" min-width="110" show-overflow-tooltip />
        <el-table-column label="" width="70" align="center">
          <template #default="{ row }">
            <el-button link type="danger" @click="removeMaterial(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-footer">
        <span>共 {{ materialList.length }} 条</span>
        <span>合计金额：<span class="money pos">¥ {{ money(materialTotal) }}</span></span>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type Material, type PriceItem, type Project } from '../api'
import { money, qty, today } from '../utils/format'

const list = ref<Project[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = ref(false)
const editing = ref(false)
const keyword = ref('')
const status = ref('')

const empty = (): Partial<Project> => ({
  name: '',
  site_address: '',
  client: '',
  contact: '',
  contract_amount: 0,
  start_date: today(),
  status: 'active',
  contract_type: '包工包料',
  remark: '',
})

const form = reactive<Partial<Project>>(empty())

const statusText = (s: string) => ({ active: '在建', done: '已完工', paused: '暂停' }[s] ?? s)
const statusType = (s: string) => ({ active: 'success', done: 'info', paused: 'warning' }[s] ?? 'info')

async function load() {
  loading.value = true
  try {
    const params: Record<string, string> = {}
    if (keyword.value) params.q = keyword.value
    if (status.value) params.status = status.value
    list.value = await api.projects(params)
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = false
  Object.assign(form, empty())
  dialog.value = true
}

function openEdit(row: Project) {
  editing.value = true
  Object.assign(form, row)
  dialog.value = true
}

async function save() {
  if (!form.name?.trim()) {
    ElMessage.warning('请填写项目名称')
    return
  }
  saving.value = true
  try {
    if (editing.value && form.id) {
      await api.updateProject(form.id, form)
      ElMessage.success('已保存')
    } else {
      await api.createProject(form)
      ElMessage.success('已创建')
    }
    dialog.value = false
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function setStatus(row: Project, next: string) {
  try {
    await api.updateProject(row.id, { ...row, status: next })
    ElMessage.success(next === 'active' ? '已恢复在建' : next === 'paused' ? '已暂停' : '已标记完工')
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

// ---- 登记进料（从项目行进入）----
const materialPrices = ref<PriceItem[]>([])
const materialCategory = ref('')
const materialCategories = computed(() => [
  ...new Set(materialPrices.value.map((p) => p.category).filter(Boolean)),
])

// 选了分类，材料名称只列该分类下的价目
const materialOptions = computed(() =>
  materialCategory.value
    ? materialPrices.value.filter((p) => p.category === materialCategory.value)
    : materialPrices.value,
)
const listDialog = ref(false)
const listLoading = ref(false)
const listProject = ref<Project | null>(null)
const materialList = ref<Material[]>([])
const materialTotal = ref(0)
const recentMaterials = ref<Material[]>([])
const materialDialog = ref(false)
const materialSaving = ref(false)
const materialProject = ref<Project | null>(null)
const material = reactive<Partial<Material>>({
  project_id: undefined as unknown as number,
  in_date: today(),
  name: '',
  spec: '',
  unit: '',
  quantity: 1,
  unit_price: 0,
  supplier: '',
  remark: '',
})

function onCategoryChange() {
  if (!material.name) return
  const stillValid = materialOptions.value.some((p) => p.name === material.name)
  if (!stillValid) {
    material.name = ''
    material.spec = ''
    material.unit = ''
    material.unit_price = 0
  }
}

function onPickMaterial(name: string) {
  const it = materialPrices.value.find((p) => p.name === name)
  if (!it) return
  material.spec = it.spec
  material.unit = it.unit
  material.unit_price = it.price
  materialCategory.value = it.category
}

async function openMaterial(row: Project) {
  materialProject.value = row
  materialCategory.value = ''
  Object.assign(material, {
    project_id: row.id,
    in_date: today(),
    name: '',
    spec: '',
    unit: '',
    quantity: 1,
    unit_price: 0,
    supplier: '',
    remark: '',
  })
  materialDialog.value = true
  try {
    const rows = await api.materials({ project_id: String(row.id) })
    recentMaterials.value = rows.slice(0, 5)
  } catch {
    recentMaterials.value = []
  }
}

async function openMaterialList(row: Project) {
  listProject.value = row
  listDialog.value = true
  listLoading.value = true
  try {
    const rows = await api.materials({ project_id: String(row.id) })
    materialList.value = rows
    materialTotal.value = rows.reduce((s, r) => s + r.amount, 0)
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    listLoading.value = false
  }
}

async function removeMaterial(row: Material) {
  try {
    await ElMessageBox.confirm(`确定删除 ${row.in_date} 的「${row.name}」？`, '删除确认', { type: 'warning' })
  } catch {
    return
  }
  try {
    await api.deleteMaterial(row.id)
    ElMessage.success('已删除')
    if (listProject.value) await openMaterialList(listProject.value)
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

async function saveMaterial() {
  if (!material.name?.trim()) {
    ElMessage.warning('请填写材料名称')
    return
  }
  if (!material.quantity || material.quantity <= 0) {
    ElMessage.warning('数量必须大于 0')
    return
  }
  materialSaving.value = true
  try {
    await api.createMaterial(material)

    // 新材料：自动补进「材料价格」价目表
    const name = (material.name || '').trim()
    const exists = materialPrices.value.some((p) => p.name === name)
    if (name && !exists) {
      try {
        await api.createPriceItem({
          kind: 'material',
          category: materialCategory.value,
          name,
          spec: material.spec || '',
          unit: material.unit || '',
          price: material.unit_price || 0,
        })
        materialPrices.value = await api.priceItems('material')
        ElMessage.success('已登记进料，并把新材料加入价格明细')
      } catch (e: any) {
        ElMessage.warning('已登记进料，但新材料写入价格明细失败：' + e.message)
      }
    } else {
      ElMessage.success('已登记进料')
    }
    material.name = ''
    material.spec = ''
    material.unit = ''
    material.quantity = 1
    material.unit_price = 0
    if (materialProject.value) {
      const rows = await api.materials({ project_id: String(materialProject.value.id) })
      recentMaterials.value = rows.slice(0, 5)
    }
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    materialSaving.value = false
  }
}

async function remove(row: Project) {
  try {
    await ElMessageBox.confirm(`确定删除项目「${row.name}」？`, '删除确认', { type: 'warning' })
  } catch {
    return
  }
  try {
    await api.deleteProject(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

onMounted(async () => {
  await load()
  try {
    materialPrices.value = await api.priceItems('material')
  } catch {
    /* ignore */
  }
})
</script>
