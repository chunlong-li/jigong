<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">设置</h2>
        <div class="page-sub">账号安全与系统信息</div>
      </div>
    </div>

    <el-row :gutter="16">
      <el-col :xs="24" :md="12">
        <div class="card">
          <h3 style="margin: 0 0 16px; font-size: 16px">修改密码</h3>
          <el-form :model="form" label-width="90px">
            <el-form-item label="原密码">
              <el-input v-model="form.old_password" type="password" show-password />
            </el-form-item>
            <el-form-item label="新密码">
              <el-input v-model="form.new_password" type="password" show-password placeholder="至少 6 位" />
            </el-form-item>
            <el-form-item label="确认新密码">
              <el-input v-model="form.confirm" type="password" show-password />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
            </el-form-item>
          </el-form>
        </div>
      </el-col>

      <el-col :xs="24" :md="12">
        <div class="card">
          <h3 style="margin: 0 0 16px; font-size: 16px">数据与部署</h3>
          <el-descriptions :column="1" size="small" border>
            <el-descriptions-item label="数据存储">SQLite，挂载在容器 /data 目录</el-descriptions-item>
            <el-descriptions-item label="备份建议">定期备份整个 data 目录即可</el-descriptions-item>
            <el-descriptions-item label="端口">默认 8910</el-descriptions-item>
            <el-descriptions-item label="时区">Asia/Shanghai</el-descriptions-item>
          </el-descriptions>
          <p class="form-tip" style="margin-top: 12px">
            提示：报价单 / 结算单导出的 Excel 由后端直接生成，浏览器下载即可。
          </p>
        </div>
      </el-col>
    </el-row>

    <div class="card" style="margin-bottom: 16px">
      <h3 style="margin: 0 0 14px; font-size: 16px">版本与更新</h3>
      <div style="display: flex; align-items: center; gap: 16px; flex-wrap: wrap">
        <span>当前版本：<b class="ver-click" title="" @click="tapVersion">{{ version || '—' }}</b></span>
        <span class="muted" v-if="platform">（{{ platform }}）</span>
        <el-button :loading="checking" @click="checkUpdate">检查更新</el-button>
        <el-button v-if="update.has_update" type="primary" :loading="applying" @click="doUpdate">
          更新到 {{ update.latest }}
        </el-button>
      </div>
      <div v-if="checked" style="margin-top: 12px">
        <el-alert
          :type="update.has_update ? 'warning' : 'success'"
          :closable="false"
          show-icon
          :title="update.has_update ? '发现新版本 ' + update.latest : '已是最新版本'"
        />
        <div v-if="update.has_update && update.notes" class="form-tip" style="margin-top: 8px; white-space: pre-wrap">
          {{ update.notes }}
        </div>
        <div v-if="update.source" class="form-tip" style="margin-top: 6px">
          更新源：{{ update.source === 'github' ? 'GitHub Releases' : update.source }}
        </div>
      </div>
      <div v-if="betaMode" class="beta-box">
        <div class="beta-row">
          <el-tag type="warning" effect="plain" size="small">测试版通道</el-tag>
          <span class="muted">当前正式版 v{{ stableVersion || version }}</span>
          <el-button size="small" :loading="betaChecking" @click="checkBeta">检测测试版</el-button>
          <el-button v-if="beta.has_update" size="small" type="warning" :loading="applying" @click="doUpdateBeta">
            更新测试版 {{ beta.latest }}
          </el-button>
          <el-button link size="small" @click="exitBeta">退出测试版通道</el-button>
        </div>
        <div v-if="betaChecked" style="margin-top: 10px">
          <el-alert
            :type="beta.has_update ? 'warning' : 'success'"
            :closable="false"
            show-icon
            :title="beta.has_update ? '发现测试版 ' + beta.latest : '已是最新测试版' + (beta.latest ? '（' + beta.latest + '）' : '')"
          />
          <div v-if="beta.has_update && beta.notes" class="form-tip" style="margin-top: 6px">{{ beta.notes }}</div>
        </div>
        <div class="form-tip" style="margin-top: 8px">
          测试版只从自有网关获取，不发 GitHub。想回正式版：等正式版号更高时用上面的「检查更新」即可。
        </div>
      </div>

      <p class="form-tip" style="margin-top: 10px">
        更新会下载对应平台安装包，替换运行目录里的程序与前端，然后自动重启服务（约 3~5 秒）。数据不受影响。
      </p>

      <el-divider content-position="left">版本更新日志</el-divider>
      <el-collapse v-if="changelog.length" v-model="activeLog" accordion class="log-collapse">
        <el-collapse-item v-for="e in changelog" :key="e.version" :name="'log' + e.version.replace(/\./g, '')">
          <template #title>
            <span class="log-title">v{{ e.version }}</span>
            <span class="log-count">{{ e.date }} 发布</span>
            <el-tag v-if="e.version === version" size="small" effect="dark" style="margin-left: 8px">当前</el-tag>
          </template>
          <div class="log-entry">
            <div v-for="(g, gi) in e.groups" :key="gi" class="log-group">
              <p class="log-group-name" v-if="g.name"><b>{{ g.name }}</b></p>
              <p v-for="(it, ii) in g.items" :key="ii" class="log-line">· {{ it }}</p>
            </div>
          </div>
        </el-collapse-item>
      </el-collapse>
      <div v-else class="form-tip">暂无更新日志</div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'

const saving = ref(false)
const form = reactive({ old_password: '', new_password: '', confirm: '' })

const version = ref('')
const platform = ref('')
const checking = ref(false)
const applying = ref(false)
const checked = ref(false)
const update = reactive({ latest: '', has_update: false, notes: '', source: '', current: '' })

// 连续点「当前版本」3 次解锁测试版通道
const betaMode = ref(false)
const betaChecking = ref(false)
const betaChecked = ref(false)
const stableVersion = ref('')
const beta = reactive({ latest: '', has_update: false, notes: '' })

interface ClGroup { name: string; items: string[] }
interface ClEntry { version: string; date: string; groups: ClGroup[] }
const changelog = ref<ClEntry[]>([])
const activeLog = ref<string[]>([])

// 解析：`## v版本 | 日期` → `### 分组` → `· 条目`
function parseChangelog(md: string): ClEntry[] {
  const out: ClEntry[] = []
  let cur: ClEntry | null = null
  let grp: ClGroup | null = null
  for (const raw of md.split('\n')) {
    const line = raw.trim()
    const h2 = line.match(/^##\s+(.+)$/)
    if (h2) {
      const [ver, date = ''] = h2[1].split('|').map((x) => x.trim())
      cur = { version: ver.replace(/^v/, ''), date, groups: [] }
      grp = null
      out.push(cur)
      continue
    }
    const h3 = line.match(/^###\s+(.+)$/)
    if (h3 && cur) {
      grp = { name: h3[1].trim(), items: [] }
      cur.groups.push(grp)
      continue
    }
    const li = line.match(/^[·\-*]\s*(.+)$/)
    if (li && cur) {
      if (!grp) {
        grp = { name: '', items: [] }
        cur.groups.push(grp)
      }
      grp.items.push(li[1].trim())
    }
  }
  return out
}
let taps = 0
let tapTimer: any = null

function tapVersion() {
  taps++
  clearTimeout(tapTimer)
  tapTimer = setTimeout(() => (taps = 0), 1500)
  if (taps >= 3) {
    taps = 0
    if (!betaMode.value) {
      betaMode.value = true
      try {
        localStorage.setItem('jigong_beta_channel', '1')
      } catch {
        /* ignore */
      }
      ElMessage.success('已解锁测试版通道')
    }
  }
}

function exitBeta() {
  betaMode.value = false
  betaChecked.value = false
  beta.has_update = false
  try {
    localStorage.removeItem('jigong_beta_channel')
  } catch {
    /* ignore */
  }
}

async function checkBeta() {
  betaChecking.value = true
  try {
    const r = await api.updateCheck('beta')
    beta.latest = r.latest
    beta.has_update = r.has_update
    beta.notes = r.notes || ''
    betaChecked.value = true
    if (!r.has_update) ElMessage.success('已是最新测试版')
  } catch (e: any) {
    ElMessage.error(e.message || '测试版检测失败')
  } finally {
    betaChecking.value = false
  }
}

async function doUpdateBeta() {
  try {
    await ElMessageBox.confirm(
      `将更新到测试版 ${beta.latest}，服务会短暂重启（约 3~5 秒），数据不受影响。是否继续？`,
      '更新测试版',
      { type: 'warning', confirmButtonText: '立即更新', cancelButtonText: '取消' },
    )
  } catch {
    return
  }
  applying.value = true
  try {
    const r = await api.updateApply('beta')
    ElMessage.success(r.message || '已更新，正在重启…')
    setTimeout(() => location.reload(), 5000)
  } catch (e: any) {
    ElMessage.error(e.message || '更新失败')
    applying.value = false
  }
}

onMounted(async () => {
  try {
    const v = await api.version()
    version.value = v.version
    // 记录正式版号（去掉测试版尾巴 x.y.z.mmdd.n → x.y.z）
    const m = v.version.match(/^(\d+\.\d+\.\d+)/)
    stableVersion.value = m ? m[1] : v.version
    try {
      betaMode.value = localStorage.getItem('jigong_beta_channel') === '1'
    } catch {
      /* ignore */
    }
    try {
      const cl = await api.changelog()
      changelog.value = parseChangelog(cl.content || '')
      if (changelog.value.length) activeLog.value = ['log' + changelog.value[0].version.replace(/\./g, '')]
    } catch {
      /* ignore */
    }
    platform.value = `${v.os}/${v.arch}`
  } catch {
    /* ignore */
  }
})

async function checkUpdate() {
  checking.value = true
  try {
    const r = await api.updateCheck()
    Object.assign(update, r)
    checked.value = true
    if (!r.has_update) ElMessage.success('已是最新版本')
  } catch (e: any) {
    ElMessage.error(e.message || '检查更新失败')
  } finally {
    checking.value = false
  }
}

async function doUpdate() {
  try {
    await ElMessageBox.confirm(
      `将更新到 ${update.latest}，服务会短暂重启（约 3~5 秒），数据不受影响。是否继续？`,
      '在线更新',
      { type: 'warning', confirmButtonText: '立即更新', cancelButtonText: '取消' },
    )
  } catch {
    return
  }
  applying.value = true
  try {
    const r = await api.updateApply()
    ElMessage.success(r.message || '已更新，正在重启…')
    setTimeout(() => location.reload(), 5000)
  } catch (e: any) {
    ElMessage.error(e.message || '更新失败')
    applying.value = false
  }
}

async function submit() {
  if (!form.old_password || !form.new_password) {
    ElMessage.warning('请填写完整')
    return
  }
  if (form.new_password.length < 6) {
    ElMessage.warning('新密码至少 6 位')
    return
  }
  if (form.new_password !== form.confirm) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  saving.value = true
  try {
    await api.changePassword(form.old_password, form.new_password)
    ElMessage.success('密码已修改，请重新登录')
    form.old_password = form.new_password = form.confirm = ''
    setTimeout(() => (location.href = '/login'), 800)
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}
</script>
