<template>
  <div class="check-in">
    <a-row :gutter="16">
      <a-col :span="12">
        <a-card title="入园核销（闸机扫码）">
          <a-form layout="vertical" @finish="handleCheckIn">
            <a-form-item label="选择闸机">
              <a-select v-model:value="checkInForm.gate_id" placeholder="请选择闸机" size="large">
                <a-select-option v-for="gate in gates" :key="gate.id" :value="gate.id">
                  {{ gate.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item label="票号">
              <a-input
                v-model:value="checkInForm.ticket_no"
                placeholder="请输入或扫描票号"
                size="large"
                @pressEnter="handleCheckIn"
              >
                <template #prefix><qrcode-outlined /></template>
              </a-input>
            </a-form-item>
            <a-form-item>
              <a-button
                type="primary"
                size="large"
                block
                html-type="submit"
                :loading="checkInLoading"
              >
                确认入园
              </a-button>
            </a-form-item>
          </a-form>

          <a-alert
            v-if="checkInResult"
            :type="checkInResult.success ? 'success' : 'error'"
            show-icon
            style="margin-top: 16px"
          >
            <template #message>
              {{ checkInResult.success ? '入园成功' : '核销失败' }}
            </template>
            <template #description>
              {{ checkInResult.message }}
            </template>
          </a-alert>
        </a-card>
      </a-col>

      <a-col :span="12">
        <a-card title="出园核销">
          <a-form layout="vertical" @finish="handleCheckOut">
            <a-form-item label="选择闸机">
              <a-select v-model:value="checkOutForm.gate_id" placeholder="请选择闸机" size="large">
                <a-select-option v-for="gate in gates" :key="gate.id" :value="gate.id">
                  {{ gate.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item label="票号">
              <a-input
                v-model:value="checkOutForm.ticket_no"
                placeholder="请输入或扫描票号"
                size="large"
                @pressEnter="handleCheckOut"
              >
                <template #prefix><qrcode-outlined /></template>
              </a-input>
            </a-form-item>
            <a-form-item>
              <a-button
                type="primary"
                size="large"
                block
                danger
                html-type="submit"
                :loading="checkOutLoading"
              >
                确认出园
              </a-button>
            </a-form-item>
          </a-form>

          <a-alert
            v-if="checkOutResult"
            :type="checkOutResult.success ? 'success' : 'error'"
            show-icon
            style="margin-top: 16px"
          >
            <template #message>
              {{ checkOutResult.success ? '出园成功' : '核销失败' }}
            </template>
            <template #description>
              {{ checkOutResult.message }}
            </template>
          </a-alert>
        </a-card>
      </a-col>
    </a-row>

    <a-card title="实时在园人数" style="margin-top: 16px">
      <div class="in-park-display">
        <div class="in-park-number">{{ inParkCount.count }}</div>
        <div class="in-park-label">人</div>
        <a-progress
          :percent="Math.round(inParkCount.percentage)"
          :stroke-color="inParkCount.percentage > 90 ? '#f5222d' : inParkCount.percentage > 70 ? '#faad14' : '#52c41a'"
          style="width: 300px; margin-left: 30px"
        />
        <span class="max-label">最大承载量: {{ inParkCount.max_capacity }} 人</span>
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { message } from 'ant-design-vue'
import { getGates, checkIn, checkOut, getInParkCount } from '@/api'

const gates = ref([])
const checkInLoading = ref(false)
const checkOutLoading = ref(false)
const checkInResult = ref(null)
const checkOutResult = ref(null)

const inParkCount = ref({
  count: 0,
  max_capacity: 8000,
  percentage: 0
})

const checkInForm = reactive({
  ticket_no: '',
  gate_id: undefined
})

const checkOutForm = reactive({
  ticket_no: '',
  gate_id: undefined
})

let timer = null

async function loadGates() {
  try {
    const res = await getGates()
    gates.value = res
    if (res.length > 0) {
      checkInForm.gate_id = res[0].id
      checkOutForm.gate_id = res[0].id
    }
  } catch (e) {}
}

async function loadInParkCount() {
  try {
    const res = await getInParkCount()
    inParkCount.value = res
  } catch (e) {}
}

async function handleCheckIn() {
  if (!checkInForm.ticket_no) {
    message.warning('请输入票号')
    return
  }
  if (!checkInForm.gate_id) {
    message.warning('请选择闸机')
    return
  }
  try {
    checkInLoading.value = true
    const res = await checkIn(checkInForm)
    checkInResult.value = { success: true, message: res.message || '入园成功' }
    checkInForm.ticket_no = ''
    loadInParkCount()
  } catch (e) {
    checkInResult.value = {
      success: false,
      message: e.response?.data?.error || '核销失败'
    }
  } finally {
    checkInLoading.value = false
    setTimeout(() => { checkInResult.value = null }, 3000)
  }
}

async function handleCheckOut() {
  if (!checkOutForm.ticket_no) {
    message.warning('请输入票号')
    return
  }
  if (!checkOutForm.gate_id) {
    message.warning('请选择闸机')
    return
  }
  try {
    checkOutLoading.value = true
    const res = await checkOut(checkOutForm)
    checkOutResult.value = { success: true, message: res.message || '出园成功' }
    checkOutForm.ticket_no = ''
    loadInParkCount()
  } catch (e) {
    checkOutResult.value = {
      success: false,
      message: e.response?.data?.error || '核销失败'
    }
  } finally {
    checkOutLoading.value = false
    setTimeout(() => { checkOutResult.value = null }, 3000)
  }
}

onMounted(() => {
  loadGates()
  loadInParkCount()
  timer = setInterval(loadInParkCount, 10000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.check-in {
  max-width: 900px;
  margin: 0 auto;
}

.in-park-display {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.in-park-number {
  font-size: 48px;
  font-weight: bold;
  color: #1890ff;
}

.in-park-label {
  font-size: 18px;
  color: #666;
  margin-left: 8px;
  margin-top: 20px;
}

.max-label {
  margin-left: 20px;
  color: #999;
  font-size: 14px;
}
</style>
