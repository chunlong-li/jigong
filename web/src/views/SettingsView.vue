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
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'

const saving = ref(false)
const form = reactive({ old_password: '', new_password: '', confirm: '' })

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
