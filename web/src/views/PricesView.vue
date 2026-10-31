<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">价格明细</h2>
        <div class="page-sub">材料价格用于进料登记自动带价；施工价格用于报价参考</div>
      </div>
      <div style="display: flex; gap: 8px">
        <el-button @click="openCatEdit">编辑分类</el-button>
        <el-button type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>{{ addLabel }}
        </el-button>
      </div>
    </div>

    <div class="toolbar">
      <el-radio-group v-model="kind" @change="onKindChange">
        <el-radio-button value="material">材料价格</el-radio-button>
        <el-radio-button value="labor_all">施工价格（包工包料）</el-radio-button>
        <el-radio-button value="labor_only">施工价格（包工）</el-radio-button>
      </el-radio-group>
      <el-select v-model="categoryFilter" placeholder="全部分类" clearable style="width: 160px" @change="load">
        <el-option v-for="c in categoryNames" :key="c" :label="c" :value="c" />
      </el-select>
      <el-input v-model="keyword" placeholder="搜索名称 / 规格" clearable style="width: 220px" @change="load">
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
    </div>

    <div class="card">
      <el-table :data="list" v-loading="loading" :empty-text="'还没有' + kindLabel + '条目'">
        <el-table-column label="分类" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.category" size="small" effect="plain">{{ row.category }}</el-tag>
            <span v-else class="muted">未分类</span>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="170" show-overflow-tooltip />
        <el-table-column prop="spec" label="规格" min-width="150" show-overflow-tooltip />
        <el-table-column prop="unit" label="单位" width="90" align="center" />
        <el-table-column v-if="kind === 'material'" label="单价(元)" width="130" align="right">
          <template #default="{ row }"><span class="money">{{ money(row.price) }}</span></template>
        </el-table-column>
        <template v-else>
          <el-table-column label="同行价格" width="120" align="right">
            <template #default="{ row }"><span class="money">{{ money(row.price) }}</span></template>
          </el-table-column>
          <el-table-column label="客户价格" width="120" align="right">
            <template #default="{ row }"><span class="money pos">{{ money(row.client_price) }}</span></template>
          </el-table-column>
        </template>
        <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip />
        <el-table-column label="操作" width="140" align="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialog" :title="dialogTitle" width="500px">
      <el-form :model="form" label-width="88px">
        <el-form-item label="分类">
          <el-select
            v-model="form.category"
            filterable
            allow-create
            default-first-option
            clearable
            style="width: 100%"
            placeholder="选择或输入自定义分类，如：水泥 / 钢材 / 辅材"
          >
            <el-option v-for="c in categoryNames" :key="c" :label="c" :value="c" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.name" :placeholder="kind === 'material' ? '如：水泥、螺纹钢' : '如：砌墙、贴砖'" />
        </el-form-item>
        <el-form-item label="规格">
          <el-input v-model="form.spec" placeholder="如：P.O 42.5、HRB400 Φ12" />
        </el-form-item>
        <el-form-item label="单位">
          <el-input v-model="form.unit" placeholder="吨 / 方 / ㎡ / 米" style="width: 160px" />
        </el-form-item>
        <template v-if="kind === 'material'">
          <el-form-item label="单价(元)">
            <el-input-number v-model="form.price" :min="0" :precision="2" :step="10" style="width: 100%" />
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item label="同行价格">
            <el-input-number v-model="form.price" :min="0" :precision="2" :step="10" style="width: 100%" />
            <div class="form-tip">报给同行 / 合作方的价格</div>
          </el-form-item>
          <el-form-item label="客户价格">
            <el-input-number v-model="form.client_price" :min="0" :precision="2" :step="10" style="width: 100%" />
            <div class="form-tip">报给客户的价格</div>
          </el-form-item>
        </template>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <!-- 编辑分类（含新建） -->
    <el-dialog v-model="catEditDialog" title="编辑分类" width="480px">
      <el-form label-width="80px">
        <el-form-item label="价目类型">
          <el-radio-group v-model="catKind" @change="onCatKindChange">
            <el-radio-button value="material">材料价格</el-radio-button>
            <el-radio-button value="labor_all">施工价格（包工包料）</el-radio-button>
            <el-radio-button value="labor_only">施工价格（包工）</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="catEditingId" style="width: 100%" @change="onCatPick">
            <el-option :value="0" label="＋ 新建分类" />
            <el-option v-for="c in catList" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="catName" :placeholder="catEditingId ? '改成什么名字' : '新分类名称'" />
        </el-form-item>
      </el-form>
      <p class="form-tip">改名后，该分类下的所有价目条目会一起更新。</p>
      <template #footer>
        <el-button @click="catEditDialog = false">取消</el-button>
        <el-button type="primary" :loading="catSaving" @click="saveCat">{{ catEditingId ? '保存改名' : '新建分类' }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type PriceCategory, type PriceItem } from '../api'
import { money } from '../utils/format'

const kind = ref<'material' | 'labor_all' | 'labor_only'>('material')
const list = ref<PriceItem[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = ref(false)
const editing = ref(false)
const keyword = ref('')
const categoryFilter = ref('')
const categories = ref<PriceCategory[]>([])
const catEditDialog = ref(false)
const catKind = ref<'material' | 'labor_all' | 'labor_only'>('material')
const catList = ref<PriceCategory[]>([])
const catName = ref('')
const catEditingId = ref<number>(0)
const catSaving = ref(false)

const form = reactive<Partial<PriceItem>>({
  kind: 'material',
  category: '',
  name: '',
  spec: '',
  unit: '',
  price: 0,
  client_price: 0,
  remark: '',
})

const kindLabel = computed(
  () =>
    ({ material: '材料价格', labor_all: '施工价格（包工包料）', labor_only: '施工价格（包工）' } as Record<string, string>)[
      kind.value
    ],
)

const addLabel = computed(() => (kind.value === 'material' ? '添加材料' : '添加工序'))

const dialogTitle = computed(() => `${editing.value ? '编辑' : '添加'}${kindLabel.value}`)

const categoryNames = computed(() => categories.value.map((c) => c.name))

async function onKindChange() {
  // 切换价目表：清空筛选，重新拉数据和分类
  categoryFilter.value = ''
  await load()
  await loadCategories()
}

async function loadCategories() {
  try {
    categories.value = await api.priceCategories(kind.value)
  } catch {
    categories.value = []
  }
}

function openCatEdit() {
  catKind.value = kind.value
  catName.value = ''
  catEditingId.value = 0
  catEditDialog.value = true
  loadCatList()
}

async function loadCatList() {
  try {
    catList.value = await api.priceCategories(catKind.value)
  } catch {
    catList.value = []
  }
}

async function onCatKindChange() {
  catEditingId.value = 0
  catName.value = ''
  await loadCatList()
}

function onCatPick(id: number) {
  if (!id) {
    catName.value = ''
    return
  }
  const c = catList.value.find((x) => x.id === id)
  catName.value = c?.name || ''
}

async function saveCat() {
  const name = catName.value.trim()
  if (!name) {
    ElMessage.warning('请填写分类名称')
    return
  }
  catSaving.value = true
  try {
    if (catEditingId.value) {
      await api.renamePriceCategory(catEditingId.value, name)
      ElMessage.success('分类已改名')
    } else {
      await api.createPriceCategory(catKind.value, name)
      ElMessage.success('分类已新建')
    }
    catEditDialog.value = false
    // 当前页签的分类列表要刷新
    if (catKind.value === kind.value) {
      if (categoryFilter.value && catEditingId.value) categoryFilter.value = name
      await loadCategories()
      await load()
    }
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    catSaving.value = false
  }
}

async function load() {
  loading.value = true
  try {
    list.value = await api.priceItems(kind.value, keyword.value, categoryFilter.value)
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = false
  Object.assign(form, {
    id: undefined,
    kind: kind.value,
    category: categoryFilter.value || '',
    name: '',
    spec: '',
    unit: '',
    price: 0,
    client_price: 0,
    remark: '',
  })
  dialog.value = true
}

function openEdit(row: PriceItem) {
  editing.value = true
  Object.assign(form, row)
  dialog.value = true
}

async function save() {
  if (!form.name?.trim()) {
    ElMessage.warning('请填写名称')
    return
  }
  saving.value = true
  try {
    const payload = { ...form, kind: kind.value }
    if (editing.value && form.id) {
      await api.updatePriceItem(form.id, payload)
    } else {
      await api.createPriceItem(payload)
    }
    ElMessage.success('已保存')
    dialog.value = false
    await load()
    await loadCategories()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function remove(row: PriceItem) {
  try {
    await ElMessageBox.confirm(`确定删除「${row.name}」？`, '删除确认', { type: 'warning' })
  } catch {
    return
  }
  try {
    await api.deletePriceItem(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

onMounted(async () => {
  await load()
  await loadCategories()
})
</script>
