<template>
  <el-container style="height: 100%">
    <el-aside v-if="!isMobile" width="216px" class="gj-aside">
      <div class="gj-brand">
        <span class="dot"></span>
        记工
      </div>
      <NavMenu />
      <div class="gj-aside-foot">记工 v1.0 · 本地部署</div>
    </el-aside>

    <el-container>
      <el-header class="gj-header">
        <div style="display: flex; align-items: center; gap: 8px; min-width: 0">
          <el-button v-if="isMobile" text class="gj-burger" @click="drawer = true">
            <el-icon :size="22"><Fold /></el-icon>
          </el-button>
          <div class="gj-header-title">{{ title }}</div>
        </div>
        <el-dropdown @command="onCommand">
          <span style="cursor: pointer; display: flex; align-items: center; gap: 8px">
            <el-avatar :size="28" style="background: #2f6bff">管</el-avatar>
            <span class="gj-user">{{ username }}</span>
            <el-icon><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="settings">修改密码</el-dropdown-item>
              <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>
      <el-main class="gj-main">
        <router-view />
      </el-main>
    </el-container>

    <el-drawer v-model="drawer" direction="ltr" size="216px" :with-header="false" class="gj-drawer">
      <div class="gj-brand">
        <span class="dot"></span>
        记工
      </div>
      <NavMenu @select="drawer = false" />
      <div class="gj-aside-foot">记工 v1.0 · 本地部署</div>
    </el-drawer>
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import NavMenu from '../components/NavMenu.vue'

const route = useRoute()
const router = useRouter()
const username = ref('admin')
const drawer = ref(false)
const isMobile = ref(false)

const titles: Record<string, string> = {
  dashboard: '工作台',
  projects: '工地项目',
  workers: '工人管理',
  worklogs: '记工',
  prices: '价格明细',
  quotations: '报价单',
  settlements: '结算单',
  settings: '设置',
}

const title = computed(() => titles[String(route.name)] ?? '记工')

function syncViewport() {
  isMobile.value = window.innerWidth < 768
  if (!isMobile.value) drawer.value = false
}

onMounted(async () => {
  syncViewport()
  window.addEventListener('resize', syncViewport)
  try {
    const me = await api.me()
    username.value = me.username || 'admin'
  } catch {
    /* ignore */
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', syncViewport)
})

async function onCommand(cmd: string) {
  if (cmd === 'settings') {
    router.push('/settings')
    return
  }
  if (cmd === 'logout') {
    try {
      await api.logout()
    } catch {
      /* ignore */
    }
    ElMessage.success('已退出登录')
    location.href = '/login'
  }
}
</script>
