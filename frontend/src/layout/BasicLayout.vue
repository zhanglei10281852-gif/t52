<template>
  <a-layout style="min-height: 100vh">
    <a-layout-sider
      v-model:collapsed="collapsed"
      collapsible
      theme="dark"
      width="220"
    >
      <div class="logo">
        <component :is="'TicketOutlined'" style="font-size: 24px" />
        <span v-if="!collapsed" class="logo-text">景区票务平台</span>
      </div>
      <a-menu
        theme="dark"
        mode="inline"
        :selected-keys="selectedKeys"
        :inline-collapsed="collapsed"
        @click="handleMenuClick"
      >
        <a-menu-item key="/dashboard">
          <template #icon>
            <component :is="'DashboardOutlined'" />
          </template>
          <span>实时看板</span>
        </a-menu-item>
        <a-menu-item v-if="hasSellerRole" key="/sell">
          <template #icon>
            <component :is="'TicketOutlined'" />
          </template>
          <span>售票管理</span>
        </a-menu-item>
        <a-menu-item key="/tickets">
          <template #icon>
            <component :is="'SearchOutlined'" />
          </template>
          <span>票务查询</span>
        </a-menu-item>
        <a-menu-item key="/check-records">
          <template #icon>
            <component :is="'CheckCircleOutlined'" />
          </template>
          <span>核销记录</span>
        </a-menu-item>
        <a-menu-item v-if="isAdmin" key="/check-in">
          <template #icon>
            <component :is="'LoginOutlined'" />
          </template>
          <span>入园核销</span>
        </a-menu-item>
        <a-menu-item key="/stats">
          <template #icon>
            <component :is="'BarChartOutlined'" />
          </template>
          <span>统计分析</span>
        </a-menu-item>
      </a-menu>
    </a-layout-sider>

    <a-layout>
      <a-layout-header class="header">
        <div class="header-left">
          <a-button type="text" @click="collapsed = !collapsed">
            <template #icon>
              <component :is="collapsed ? 'MenuUnfoldOutlined' : 'MenuFoldOutlined'" />
            </template>
          </a-button>
          <span class="page-title">{{ pageTitle }}</span>
        </div>
        <div class="header-right">
          <a-dropdown>
            <span class="user-info">
              <component :is="'UserOutlined'" />
              {{ userName }}
              <component :is="'DownOutlined'" />
            </span>
            <template #overlay>
              <a-menu @click="handleUserMenuClick">
                <a-menu-item key="logout">
                  <component :is="'LogoutOutlined'" />
                  退出登录
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </a-layout-header>

      <a-layout-content class="content">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </a-layout-content>

      <a-layout-footer class="footer">
        景区票务客流管控平台 ©2024
      </a-layout-footer>
    </a-layout>
  </a-layout>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/store/user'
import { getCurrentUser } from '@/api'
import {
  DashboardOutlined,
  TicketOutlined,
  SearchOutlined,
  CheckCircleOutlined,
  LoginOutlined,
  BarChartOutlined,
  UserOutlined,
  DownOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined
} from '@ant-design/icons-vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const collapsed = ref(false)
const selectedKeys = ref([route.path])

const user = computed(() => userStore.user)
const userName = computed(() => user.value?.name || user.value?.username || '')
const isAdmin = computed(() => user.value?.role === 'admin')
const hasSellerRole = computed(() => user.value?.role === 'admin' || user.value?.role === 'seller')
const pageTitle = computed(() => route.meta?.title || '')

watch(() => route.path, (path) => {
  selectedKeys.value = [path]
})

onMounted(() => {
  if (!userStore.user) {
    loadUserInfo()
  }
  selectedKeys.value = [route.path]
})

function loadUserInfo() {
  getCurrentUser().then(res => {
    userStore.setUser(res)
  }).catch(() => {
    router.push('/login')
  })
}

function handleMenuClick({ key }) {
  router.push(key)
}

function handleUserMenuClick({ key }) {
  if (key === 'logout') {
    userStore.logout()
    router.push('/login')
  }
}
</script>

<style scoped>
.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: white;
  font-size: 16px;
  font-weight: bold;
  background: rgba(255, 255, 255, 0.1);
}

.logo-text {
  font-size: 16px;
}

.header {
  background: white;
  padding: 0 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 64px;
  line-height: 64px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
}

.header-right {
  display: flex;
  align-items: center;
}

.user-info {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
}

.content {
  margin: 16px;
  min-height: 280px;
}

.footer {
  text-align: center;
  background: white;
  color: rgba(0, 0, 0, 0.45);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
