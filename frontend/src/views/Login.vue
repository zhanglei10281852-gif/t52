<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <ticket-outlined class="logo-icon" />
        <h1>景区票务客流管控平台</h1>
        <p>Scenic Ticket Management System</p>
      </div>
      <a-form
        ref="formRef"
        :model="formState"
        :rules="rules"
        @finish="handleLogin"
        layout="vertical"
      >
        <a-form-item label="用户名" name="username">
          <a-input
            v-model:value="formState.username"
            placeholder="请输入用户名"
            size="large"
          >
            <template #prefix><user-outlined /></template>
          </a-input>
        </a-form-item>
        <a-form-item label="密码" name="password">
          <a-input-password
            v-model:value="formState.password"
            placeholder="请输入密码"
            size="large"
            @pressEnter="handleLogin"
          >
            <template #prefix><lock-outlined /></template>
          </a-input-password>
        </a-form-item>
        <a-form-item>
          <a-button
            type="primary"
            html-type="submit"
            size="large"
            block
            :loading="loading"
          >
            登 录
          </a-button>
        </a-form-item>
      </a-form>
      <div class="login-tips">
        <p>管理员账号: admin / scenic2024</p>
        <p>售票员账号: seller1 / sell123</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { login as loginApi } from '@/api'
import { useUserStore } from '@/store/user'

const router = useRouter()
const userStore = useUserStore()
const formRef = ref()
const loading = ref(false)

const formState = reactive({
  username: '',
  password: ''
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

async function handleLogin() {
  try {
    loading.value = true
    const res = await loginApi(formState.username, formState.password)
    userStore.setToken(res.token)
    userStore.setUser(res.user)
    message.success('登录成功')
    router.push('/')
  } catch (e) {
    // error handled in interceptor
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  width: 420px;
  padding: 40px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.15);
}

.login-header {
  text-align: center;
  margin-bottom: 30px;
}

.logo-icon {
  font-size: 48px;
  color: #1890ff;
  margin-bottom: 10px;
}

.login-header h1 {
  font-size: 22px;
  color: rgba(0, 0, 0, 0.85);
  margin: 8px 0;
}

.login-header p {
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
  margin: 0;
}

.login-tips {
  margin-top: 20px;
  padding-top: 15px;
  border-top: 1px solid #f0f0f0;
  font-size: 12px;
  color: #999;
  text-align: center;
}

.login-tips p {
  margin: 4px 0;
}
</style>
