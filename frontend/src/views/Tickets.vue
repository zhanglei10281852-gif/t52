<template>
  <div class="tickets">
    <a-card>
      <a-form layout="inline" @finish="handleSearch">
        <a-form-item label="手机号">
          <a-input v-model:value="searchForm.phone" placeholder="请输入手机号" allow-clear />
        </a-form-item>
        <a-form-item label="票号">
          <a-input v-model:value="searchForm.ticket_no" placeholder="请输入票号" allow-clear />
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" html-type="submit">查询</a-button>
            <a-button @click="handleReset">重置</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-card>

    <a-card style="margin-top: 16px">
      <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        :pagination="{
          current: pagination.page,
          pageSize: pagination.size,
          total: pagination.total,
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条记录`,
          onChange: handlePageChange,
          onShowSizeChange: handlePageChange
        }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'price'">
            ¥{{ record.price }}
          </template>
          <template v-else-if="column.key === 'check_in_time'">
            {{ record.check_in_time ? formatTime(record.check_in_time) : '-' }}
          </template>
          <template v-else-if="column.key === 'check_out_time'">
            {{ record.check_out_time ? formatTime(record.check_out_time) : '-' }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="viewDetail(record)">详情</a-button>
              <a-button
                v-if="record.status === 'unused'"
                type="link"
                size="small"
                danger
                @click="handleRefund(record)"
              >
                退票
              </a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:open="detailVisible"
      title="票务详情"
      :footer="null"
      width="600px"
    >
      <a-descriptions :column="2" bordered v-if="currentTicket">
        <a-descriptions-item label="票号">{{ currentTicket.ticket_no }}</a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-tag :color="getStatusColor(currentTicket.status)">
            {{ getStatusText(currentTicket.status) }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="票型">{{ currentTicket.ticket_type?.name }}</a-descriptions-item>
        <a-descriptions-item label="票价">¥{{ currentTicket.price }}</a-descriptions-item>
        <a-descriptions-item label="购票人">{{ currentTicket.buyer_name }}</a-descriptions-item>
        <a-descriptions-item label="手机号">{{ currentTicket.phone }}</a-descriptions-item>
        <a-descriptions-item label="游览日期">{{ formatDate(currentTicket.visit_date) }}</a-descriptions-item>
        <a-descriptions-item label="预约时段">{{ currentTicket.time_slot_name }}</a-descriptions-item>
        <a-descriptions-item label="售票员">{{ currentTicket.seller_name }}</a-descriptions-item>
        <a-descriptions-item label="售票时间">{{ formatTime(currentTicket.created_at) }}</a-descriptions-item>
        <a-descriptions-item label="入园闸机">{{ currentTicket.check_in_gate?.name || '-' }}</a-descriptions-item>
        <a-descriptions-item label="入园时间">{{ currentTicket.check_in_time ? formatTime(currentTicket.check_in_time) : '-' }}</a-descriptions-item>
        <a-descriptions-item label="出园闸机">{{ currentTicket.check_out_gate?.name || '-' }}</a-descriptions-item>
        <a-descriptions-item label="出园时间">{{ currentTicket.check_out_time ? formatTime(currentTicket.check_out_time) : '-' }}</a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { searchTickets, refundTicket } from '@/api'
import dayjs from 'dayjs'

const loading = ref(false)
const tableData = ref([])
const detailVisible = ref(false)
const currentTicket = ref(null)

const searchForm = reactive({
  phone: '',
  ticket_no: ''
})

const pagination = reactive({
  page: 1,
  size: 10,
  total: 0
})

const columns = [
  { title: '票号', dataIndex: 'ticket_no', key: 'ticket_no' },
  { title: '票型', dataIndex: ['ticket_type', 'name'], key: 'ticket_type' },
  { title: '购票人', dataIndex: 'buyer_name', key: 'buyer_name' },
  { title: '手机号', dataIndex: 'phone', key: 'phone' },
  { title: '游览日期', dataIndex: 'visit_date', key: 'visit_date', customRender: ({ text }) => formatDate(text) },
  { title: '时段', dataIndex: 'time_slot_name', key: 'time_slot_name' },
  { title: '票价', dataIndex: 'price', key: 'price' },
  { title: '状态', key: 'status' },
  { title: '入园时间', key: 'check_in_time' },
  { title: '操作', key: 'action', width: 150, fixed: 'right' }
]

function getStatusColor(status) {
  const colors = {
    unused: 'blue',
    used: 'green',
    refunded: 'default'
  }
  return colors[status] || 'default'
}

function getStatusText(status) {
  const texts = {
    unused: '未使用',
    used: '已使用',
    refunded: '已退票'
  }
  return texts[status] || status
}

function formatDate(date) {
  return dayjs(date).format('YYYY-MM-DD')
}

function formatTime(date) {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

async function loadData() {
  try {
    loading.value = true
    const params = {
      page: pagination.page,
      page_size: pagination.size
    }
    if (searchForm.phone) params.phone = searchForm.phone
    if (searchForm.ticket_no) params.ticket_no = searchForm.ticket_no

    const res = await searchTickets(params)
    tableData.value = res.list
    pagination.total = res.total
  } catch (e) {
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  pagination.page = 1
  loadData()
}

function handleReset() {
  searchForm.phone = ''
  searchForm.ticket_no = ''
  pagination.page = 1
  loadData()
}

function handlePageChange(page, pageSize) {
  pagination.page = page
  pagination.size = pageSize
  loadData()
}

function viewDetail(record) {
  currentTicket.value = record
  detailVisible.value = true
}

function handleRefund(record) {
  Modal.confirm({
    title: '确认退票',
    content: `确定要退票吗？票号: ${record.ticket_no}`,
    okText: '确认退票',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      try {
        await refundTicket(record.ticket_no)
        message.success('退票成功')
        loadData()
      } catch (e) {}
    }
  })
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.tickets {
  padding: 0;
}
</style>
