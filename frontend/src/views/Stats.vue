<template>
  <div class="stats">
    <a-card>
      <a-tabs v-model:activeKey="activeTab">
        <a-tab-pane key="daily" tab="每日入园趋势">
          <div class="chart-container">
            <div ref="lineChart" class="chart"></div>
          </div>
        </a-tab-pane>
        <a-tab-pane key="ticket-type" tab="票型收入统计">
          <div class="chart-container">
            <div ref="barChart" class="chart"></div>
          </div>
        </a-tab-pane>
        <a-tab-pane key="heatmap" tab="时段热度热力图">
          <div class="chart-container">
            <div ref="heatmapChart" class="chart"></div>
          </div>
        </a-tab-pane>
        <a-tab-pane key="gate" tab="各闸机通过量">
          <a-table
            :columns="gateColumns"
            :data-source="gateStats"
            :pagination="false"
            row-key="gate_id"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'total'">
                <a-tag color="blue">{{ record.total }}</a-tag>
              </template>
            </template>
          </a-table>
        </a-tab-pane>
      </a-tabs>
    </a-card>

    <a-card title="日期范围" style="margin-top: 16px">
      <a-range-picker
        v-model:value="dateRange"
        valueFormat="YYYY-MM-DD"
        @change="handleDateChange"
      />
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick, onUnmounted } from 'vue'
import * as echarts from 'echarts'
import { getDailyStats, getTicketTypeStats, getSlotHeatmap, getGateStats } from '@/api'
import dayjs from 'dayjs'

const activeTab = ref('daily')
const dateRange = ref([
  dayjs().subtract(29, 'day').format('YYYY-MM-DD'),
  dayjs().format('YYYY-MM-DD')
])

const lineChart = ref(null)
const barChart = ref(null)
const heatmapChart = ref(null)

let lineChartInstance = null
let barChartInstance = null
let heatmapChartInstance = null

const dailyData = ref([])
const ticketTypeData = ref([])
const heatmapData = ref([])
const gateStats = ref([])

const gateColumns = [
  { title: '闸机名称', dataIndex: 'gate_name', key: 'gate_name' },
  { title: '入园人次', dataIndex: 'in_count', key: 'in_count' },
  { title: '出园人次', dataIndex: 'out_count', key: 'out_count' },
  { title: '总通过量', key: 'total' }
]

async function loadDailyStats() {
  try {
    const params = {}
    if (dateRange.value && dateRange.value.length === 2) {
      params.start_date = dateRange.value[0]
      params.end_date = dateRange.value[1]
    }
    const res = await getDailyStats(params)
    dailyData.value = res
    updateLineChart()
  } catch (e) {}
}

async function loadTicketTypeStats() {
  try {
    const params = {}
    if (dateRange.value && dateRange.value.length === 2) {
      params.start_date = dateRange.value[0]
      params.end_date = dateRange.value[1]
    }
    const res = await getTicketTypeStats(params)
    ticketTypeData.value = res
    updateBarChart()
  } catch (e) {}
}

async function loadHeatmapStats() {
  try {
    const params = {}
    if (dateRange.value && dateRange.value.length === 2) {
      params.start_date = dateRange.value[0]
      params.end_date = dateRange.value[1]
    }
    const res = await getSlotHeatmap(params)
    heatmapData.value = res
    updateHeatmapChart()
  } catch (e) {}
}

async function loadGateStats() {
  try {
    const params = {}
    if (dateRange.value && dateRange.value.length === 2) {
      params.start_date = dateRange.value[0]
      params.end_date = dateRange.value[1]
    }
    const res = await getGateStats(params)
    gateStats.value = res
  } catch (e) {}
}

function updateLineChart() {
  if (!lineChartInstance) return
  const dates = dailyData.value.map(item => item.date)
  const checkInCounts = dailyData.value.map(item => item.check_in_count)
  const soldCounts = dailyData.value.map(item => item.sold_count)

  lineChartInstance.setOption({
    tooltip: {
      trigger: 'axis'
    },
    legend: {
      data: ['入园人数', '售票数']
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: dates
    },
    yAxis: {
      type: 'value',
      name: '人数'
    },
    series: [
      {
        name: '入园人数',
        type: 'line',
        smooth: true,
        data: checkInCounts,
        itemStyle: { color: '#1890ff' },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(24, 144, 255, 0.3)' },
            { offset: 1, color: 'rgba(24, 144, 255, 0.05)' }
          ])
        }
      },
      {
        name: '售票数',
        type: 'line',
        smooth: true,
        data: soldCounts,
        itemStyle: { color: '#52c41a' }
      }
    ]
  })
}

function updateBarChart() {
  if (!barChartInstance) return
  const names = ticketTypeData.value.map(item => item.ticket_type_name)
  const revenues = ticketTypeData.value.map(item => item.revenue)
  const counts = ticketTypeData.value.map(item => item.count)

  barChartInstance.setOption({
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' }
    },
    legend: {
      data: ['收入(元)', '售票数']
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: names
    },
    yAxis: [
      {
        type: 'value',
        name: '收入(元)',
        position: 'left'
      },
      {
        type: 'value',
        name: '售票数',
        position: 'right'
      }
    ],
    series: [
      {
        name: '收入(元)',
        type: 'bar',
        data: revenues,
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: '#1890ff' },
            { offset: 1, color: '#69c0ff' }
          ]),
          borderRadius: [4, 4, 0, 0]
        }
      },
      {
        name: '售票数',
        type: 'bar',
        yAxisIndex: 1,
        data: counts,
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: '#52c41a' },
            { offset: 1, color: '#95de64' }
          ]),
          borderRadius: [4, 4, 0, 0]
        }
      }
    ]
  })
}

function updateHeatmapChart() {
  if (!heatmapChartInstance) return

  const dateSet = new Set()
  const slotSet = new Set()
  heatmapData.value.forEach(item => {
    dateSet.add(item.date)
    slotSet.add(item.slot_name)
  })

  const dates = Array.from(dateSet).sort()
  const slots = ['08:00-10:00', '10:00-12:00', '12:00-14:00', '14:00-16:00']

  const data = []
  let maxVal = 0
  heatmapData.value.forEach(item => {
    const x = dates.indexOf(item.date)
    const y = slots.indexOf(item.slot_name)
    if (x >= 0 && y >= 0) {
      data.push([x, y, item.count])
      if (item.count > maxVal) maxVal = item.count
    }
  })

  heatmapChartInstance.setOption({
    tooltip: {
      position: 'top',
      formatter: function (params) {
        return `${dates[params.data[0]]}<br/>${slots[params.data[1]]}<br/>入园: ${params.data[2]}人`
      }
    },
    grid: {
      left: '10%',
      right: '10%',
      top: '10%',
      bottom: '15%'
    },
    xAxis: {
      type: 'category',
      data: dates,
      splitArea: { show: true },
      axisLabel: { rotate: 45 }
    },
    yAxis: {
      type: 'category',
      data: slots,
      splitArea: { show: true }
    },
    visualMap: {
      min: 0,
      max: maxVal || 100,
      calculable: true,
      orient: 'horizontal',
      left: 'center',
      bottom: '0%',
      inRange: {
        color: ['#f0f9eb', '#67c23a', '#e6a23c', '#f56c6c']
      }
    },
    series: [{
      name: '入园人数',
      type: 'heatmap',
      data: data,
      label: {
        show: true,
        formatter: function(params) {
          return params.data[2]
        }
      },
      emphasis: {
        itemStyle: {
          shadowBlur: 10,
          shadowColor: 'rgba(0, 0, 0, 0.5)'
        }
      }
    }]
  })
}

function initCharts() {
  if (lineChart.value) {
    lineChartInstance = echarts.init(lineChart.value)
  }
  if (barChart.value) {
    barChartInstance = echarts.init(barChart.value)
  }
  if (heatmapChart.value) {
    heatmapChartInstance = echarts.init(heatmapChart.value)
  }
}

function handleDateChange() {
  loadDailyStats()
  loadTicketTypeStats()
  loadHeatmapStats()
  loadGateStats()
}

function handleResize() {
  lineChartInstance?.resize()
  barChartInstance?.resize()
  heatmapChartInstance?.resize()
}

onMounted(async () => {
  await Promise.all([
    loadDailyStats(),
    loadTicketTypeStats(),
    loadHeatmapStats(),
    loadGateStats()
  ])
  await nextTick()
  initCharts()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  lineChartInstance?.dispose()
  barChartInstance?.dispose()
  heatmapChartInstance?.dispose()
})
</script>

<style scoped>
.stats {
  padding: 0;
}

.chart-container {
  padding: 10px 0;
}

.chart {
  height: 400px;
  width: 100%;
}
</style>
