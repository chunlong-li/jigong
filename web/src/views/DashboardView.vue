<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">工作台</h2>
        <div class="page-sub">今天 {{ todayText }}，快速掌握工地用工与进料情况</div>
      </div>
      <div>
        <el-button type="primary" @click="$router.push('/worklogs')">
          <el-icon><Plus /></el-icon>记一笔工
        </el-button>
        <el-button @click="$router.push('/projects')">登记进料</el-button>
      </div>
    </div>

    <div class="stat-grid">
      <div class="stat-card">
        <div class="stat-label">在建项目</div>
        <div class="stat-value">{{ summary.project_count }}<small>个</small></div>
      </div>
      <div class="stat-card">
        <div class="stat-label">在册工人</div>
        <div class="stat-value">{{ summary.worker_count }}<small>人</small></div>
      </div>
      <div class="stat-card">
        <div class="stat-label">本月记工金额</div>
        <div class="stat-value">¥ {{ money(summary.month_work) }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">本月进料金额</div>
        <div class="stat-value">¥ {{ money(summary.month_material) }}</div>
      </div>
    </div>

    <el-row :gutter="16">
      <el-col :xs="24" :lg="14">
        <div class="card">
          <div class="page-header" style="margin-bottom: 8px">
            <h3 style="margin: 0; font-size: 16px">最近记工</h3>
            <el-button link type="primary" @click="$router.push('/worklogs')">查看全部</el-button>
          </div>
          <el-table :data="recentLogs" size="small" empty-text="还没有记工记录">
            <el-table-column prop="work_date" label="日期" width="110" />
            <el-table-column prop="worker_name" label="工人" width="100" />
            <el-table-column prop="project_name" label="项目" show-overflow-tooltip />
            <el-table-column label="计价" width="120">
              <template #default="{ row }">
                {{ modeText[row.mode] }} {{ qty(row.quantity) }}{{ modeUnit[row.mode] }} × {{ money(row.unit_price) }}
              </template>
            </el-table-column>
            <el-table-column label="金额" width="110" align="right">
              <template #default="{ row }">
                <span class="money">{{ money(row.amount) }}</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>

      <el-col :xs="24" :lg="10">
        <div class="card">
          <div class="page-header" style="margin-bottom: 8px">
            <h3 style="margin: 0; font-size: 16px">累计</h3>
          </div>
          <div style="display: flex; flex-direction: column; gap: 14px">
            <div style="display: flex; justify-content: space-between">
              <span class="muted">累计记工</span>
              <span class="money">¥ {{ money(summary.work_amount) }}</span>
            </div>
            <div style="display: flex; justify-content: space-between">
              <span class="muted">累计进料</span>
              <span class="money">¥ {{ money(summary.material_amount) }}</span>
            </div>
            <el-divider style="margin: 4px 0" />
            <div style="display: flex; justify-content: space-between">
              <span class="muted">本月合计</span>
              <span class="money pos">¥ {{ money(summary.month_work + summary.month_material) }}</span>
            </div>
          </div>
          <div style="margin-top: 20px; display: grid; gap: 10px">
            <el-button @click="$router.push('/quotations')">新建报价单</el-button>
            <el-button @click="$router.push('/settlements')">生成结算单</el-button>
          </div>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type Summary, type WorkLog } from '../api'
import { money, qty, modeText, modeUnit } from '../utils/format'

const summary = ref<Summary>({
  project_count: 0,
  worker_count: 0,
  work_amount: 0,
  material_amount: 0,
  month_work: 0,
  month_material: 0,
})
const recentLogs = ref<WorkLog[]>([])
const todayText = new Date().toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' })

onMounted(async () => {
  try {
    summary.value = await api.summary()
    const logs = await api.worklogs()
    recentLogs.value = logs.slice(0, 8)
  } catch {
    /* 忽略首次加载错误 */
  }
})
</script>
