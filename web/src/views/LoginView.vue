<template>
  <div class="login-wrap">
    <div class="login-card">
      <div class="login-logo">
        <div class="logo-badge">工</div>
        <h1>记工</h1>
        <p>工地记工 · 进料 · 报价 · 结算</p>
      </div>
      <el-form :model="form" @submit.prevent="submit">
        <el-form-item>
          <el-input v-model="form.username" size="large" placeholder="账号" :prefix-icon="User" />
        </el-form-item>
        <el-form-item>
          <el-input
            v-model="form.password"
            size="large"
            type="password"
            placeholder="密码"
            show-password
            :prefix-icon="Lock"
            @keyup.enter="submit"
          />
        </el-form-item>
        <el-button type="primary" size="large" style="width: 100%" :loading="loading" @click="submit">
          登 录
        </el-button>
      </el-form>
      <p v-if="showHint" class="form-tip" style="text-align: center; margin-top: 18px">
        初始账号 admin / admin123，登录后请到「设置」修改密码
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import { api } from '../api'

const router = useRouter()
const route = useRoute()
const loading = ref(false)
const showHint = ref(false)
const form = reactive({ username: 'admin', password: '' })

onMounted(async () => {
  try {
    const st = await api.initStatus()
    showHint.value = st.first_time
  } catch {
    showHint.value = true
  }
})

async function submit() {
  if (!form.username || !form.password) {
    ElMessage.warning('请输入账号和密码')
    return
  }
  loading.value = true
  try {
    await api.login(form.username, form.password)
    ElMessage.success('登录成功')
    const redirect = (route.query.redirect as string) || '/'
    location.href = redirect
  } catch (e: any) {
    ElMessage.error(e.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>
