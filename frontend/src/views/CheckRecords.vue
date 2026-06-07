<template>
  <div class="check-records">
    <a-card>
      <a-form layout="inline" @finish="handleSearch">
        <a-form-item label="类型">
          <a-select v-model:value="searchForm.check_type" placeholder="全部" allow-clear style="width: 120px">
            <a-select-option value="in">入园</a-select-option>
            <a-select-option value="out">出园</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="日期">
          <a-date-picker v-model:value="searchForm.date" valueFormat="YYYY-MM-DD" allow-clear />
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
          <template v-if="column.key === 'check_type'">
            <a-tag :color="record.check_type === 'in' ? 'green' : 'orange'">
              {{ record.check_type === 'in' ? '入园' : '出园' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'check_time'">
            {{ formatTime(record.check_time) }}
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { getCheckRecords } from '@/api'
import dayjs from 'dayjs'

const loading = ref(false)
const tableData = ref([])

const searchForm = reactive({
  check_type: undefined,
  date: undefined
})

const pagination = reactive({
  page: 1,
  size: 10,
  total: 0
})

const columns = [
  { title: '票号', dataIndex: 'ticket_no', key: 'ticket_no' },
  { title: '类型', key: 'check_type', width: 80 },
  { title: '闸机', dataIndex: 'gate_name', key: 'gate_name' },
  { title: '时间', key: 'check_time' },
  { title: '时段ID', dataIndex: 'time_slot_id', key: 'time_slot_id', width: 80 }
]

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
    if (searchForm.check_type) params.check_type = searchForm.check_type
    if (searchForm.date) params.date = searchForm.date

    const res = await getCheckRecords(params)
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
  searchForm.check_type = undefined
  searchForm.date = undefined
  pagination.page = 1
  loadData()
}

function handlePageChange(page, pageSize) {
  pagination.page = page
  pagination.size = pageSize
  loadData()
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.check-records {
  padding: 0;
}
</style>
