<template>
  <el-menu class="gj-menu" :default-active="active" router @select="$emit('select')">
    <el-menu-item index="/">
      <el-icon><HomeFilled /></el-icon>
      <span>工作台</span>
    </el-menu-item>
    <el-menu-item index="/projects">
      <el-icon><OfficeBuilding /></el-icon>
      <span>工地项目</span>
    </el-menu-item>
    <el-menu-item index="/workers">
      <el-icon><User /></el-icon>
      <span>工人管理</span>
    </el-menu-item>
    <el-menu-item index="/worklogs">
      <el-icon><Calendar /></el-icon>
      <span>记工</span>
    </el-menu-item>
    <el-menu-item index="/prices">
      <el-icon><PriceTag /></el-icon>
      <span>价格明细</span>
    </el-menu-item>
    <el-menu-item index="/quotations">
      <el-icon><Document /></el-icon>
      <span>报价单</span>
    </el-menu-item>
    <el-menu-item index="/settlements">
      <el-icon><Money /></el-icon>
      <span>结算单</span>
    </el-menu-item>
    <el-menu-item index="/settings">
      <el-icon><Setting /></el-icon>
      <span>设置</span>
    </el-menu-item>
  </el-menu>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const active = computed(() => (route.path === '/' ? '/' : route.path))

defineEmits<{ (e: 'select'): void }>()
</script>
