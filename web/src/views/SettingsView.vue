<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">设置</h2>
        <div class="page-sub">账号安全与系统信息</div>
      </div>
    </div>

    <div class="card" style="margin-bottom: 16px">
      <h3 style="margin: 0 0 14px; font-size: 16px">版本与更新</h3>
      <div style="display: flex; align-items: center; gap: 16px; flex-wrap: wrap">
        <span>当前版本：<b>{{ version || '—' }}</b></span>
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
      <p class="form-tip" style="margin-top: 10px">
        更新会下载对应平台安装包，替换运行目录里的程序与前端，然后自动重启服务（约 3~5 秒）。数据不受影响。
      </p>
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

onMounted(async () => {
  try {
    const v = await api.version()
    version.value = v.version
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
