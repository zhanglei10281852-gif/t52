<template>
  <div class="sell-ticket">
    <a-card title="售票管理">
      <a-form
        ref="formRef"
        :model="formState"
        :rules="rules"
        layout="vertical"
        @finish="handleSell"
        style="max-width: 500px; margin: 0 auto"
      >
        <a-form-item label="票型" name="ticket_type_id">
          <a-select v-model:value="formState.ticket_type_id" placeholder="请选择票型" size="large">
            <a-select-option
              v-for="tt in ticketTypes"
              :key="tt.id"
              :value="tt.id"
            >
              {{ tt.name }} - ¥{{ tt.price }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="购票人姓名" name="buyer_name">
          <a-input v-model:value="formState.buyer_name" placeholder="请输入购票人姓名" size="large" />
        </a-form-item>

        <a-form-item label="手机号" name="phone">
          <a-input v-model:value="formState.phone" placeholder="请输入手机号（电子票凭证）" size="large" />
        </a-form-item>

        <a-form-item label="游览日期" name="visit_date">
          <a-date-picker
            v-model:value="formState.visit_date"
            :disabled-date="disabledDate"
            style="width: 100%"
            size="large"
            valueFormat="YYYY-MM-DD"
          />
        </a-form-item>

        <a-form-item label="预约时段" name="time_slot_id">
          <a-radio-group v-model:value="formState.time_slot_id" size="large">
            <a-radio-button :value="1">08:00-10:00</a-radio-button>
            <a-radio-button :value="2">10:00-12:00</a-radio-button>
            <a-radio-button :value="3">12:00-14:00</a-radio-button>
            <a-radio-button :value="4">14:00-16:00</a-radio-button>
          </a-radio-group>
        </a-form-item>

        <a-form-item>
          <a-space>
            <a-button type="primary" size="large" html-type="submit" :loading="loading">
              确认售票
            </a-button>
            <a-button size="large" @click="resetForm">重置</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-card>

    <a-card title="退票" style="margin-top: 16px">
      <a-space style="max-width: 500px; margin: 0 auto; display: flex">
        <a-input
          v-model:value="refundTicketNo"
          placeholder="请输入票号"
          size="large"
          style="flex: 1"
        />
        <a-button type="danger" size="large" @click="handleRefund" :loading="refundLoading">
          退票
        </a-button>
      </a-space>
    </a-card>

    <a-modal
      v-model:open="resultVisible"
      title="售票成功"
      :footer="null"
      @ok="resultVisible = false"
    >
      <div class="result-content">
        <check-circle-outlined class="success-icon" />
        <p class="ticket-no">{{ resultTicketNo }}</p>
        <p class="result-desc">请妥善保管您的票号</p>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { getTicketTypes, sellTicket, refundTicket } from '@/api'
import dayjs from 'dayjs'

const formRef = ref()
const ticketTypes = ref([])
const loading = ref(false)
const refundLoading = ref(false)
const refundTicketNo = ref('')
const resultVisible = ref(false)
const resultTicketNo = ref('')

const formState = reactive({
  ticket_type_id: undefined,
  buyer_name: '',
  phone: '',
  visit_date: dayjs().format('YYYY-MM-DD'),
  time_slot_id: 1
})

const rules = {
  ticket_type_id: [{ required: true, message: '请选择票型', trigger: 'change' }],
  buyer_name: [{ required: true, message: '请输入购票人姓名', trigger: 'blur' }],
  phone: [
    { required: true, message: '请输入手机号', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号', trigger: 'blur' }
  ],
  visit_date: [{ required: true, message: '请选择游览日期', trigger: 'change' }],
  time_slot_id: [{ required: true, message: '请选择预约时段', trigger: 'change' }]
}

function disabledDate(current) {
  return current && current < dayjs().startOf('day')
}

async function loadTicketTypes() {
  try {
    const res = await getTicketTypes()
    ticketTypes.value = res
    if (res.length > 0) {
      formState.ticket_type_id = res[0].id
    }
  } catch (e) {}
}

async function handleSell() {
  try {
    loading.value = true
    const res = await sellTicket(formState)
    resultTicketNo.value = res.ticket_no
    resultVisible.value = true
    message.success('售票成功')
    formRef.value?.resetFields()
    formState.visit_date = dayjs().format('YYYY-MM-DD')
    formState.time_slot_id = 1
    formState.ticket_type_id = ticketTypes.value[0]?.id
  } catch (e) {
    // error handled in interceptor
  } finally {
    loading.value = false
  }
}

function resetForm() {
  formRef.value?.resetFields()
  formState.visit_date = dayjs().format('YYYY-MM-DD')
  formState.time_slot_id = 1
  formState.ticket_type_id = ticketTypes.value[0]?.id
}

async function handleRefund() {
  if (!refundTicketNo.value) {
    message.warning('请输入票号')
    return
  }
  try {
    refundLoading.value = true
    await refundTicket(refundTicketNo.value)
    message.success('退票成功')
    refundTicketNo.value = ''
  } catch (e) {
  } finally {
    refundLoading.value = false
  }
}

onMounted(() => {
  loadTicketTypes()
})
</script>

<style scoped>
.sell-ticket {
  max-width: 600px;
  margin: 0 auto;
}

.result-content {
  text-align: center;
  padding: 20px;
}

.success-icon {
  font-size: 64px;
  color: #52c41a;
}

.ticket-no {
  font-size: 24px;
  font-weight: bold;
  color: #1890ff;
  margin: 16px 0 8px;
  letter-spacing: 2px;
}

.result-desc {
  color: #666;
  font-size: 14px;
}
</style>
