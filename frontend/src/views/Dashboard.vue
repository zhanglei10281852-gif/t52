<template>
  <div class="dashboard">
    <a-row :gutter="16">
      <a-col :span="8">
        <a-card class="stat-card">
          <a-statistic
            title="今日入园人数"
            :value="dashboardData.today_check_in"
            :value-style="{ color: '#52c41a' }"
          >
            <template #prefix><login-outlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="8">
        <a-card class="stat-card">
          <a-statistic
            title="今日出园人数"
            :value="dashboardData.today_check_out"
            :value-style="{ color: '#faad14' }"
          >
            <template #prefix><logout-outlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="8">
        <a-card class="stat-card">
          <a-statistic
            title="今日售票数"
            :value="dashboardData.today_sold"
            :value-style="{ color: '#1890ff' }"
          >
            <template #prefix><ticket-outlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="12">
        <a-card title="当前在园人数" class="chart-card">
          <div class="in-park-container">
            <div ref="gaugeChart" class="gauge-chart"></div>
            <div class="in-park-info">
              <div class="in-park-count">{{ dashboardData.in_park_count }}</div>
              <div class="in-park-label">人</div>
              <div class="in-park-max">最大承载量: {{ dashboardData.max_capacity }} 人</div>
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card title="各时段余票" class="chart-card">
          <a-list size="large">
            <a-list-item v-for="slot in dashboardData.slot_info" :key="slot.slot_id">
              <a-list-item-meta>
                <template #title>{{ slot.slot_name }}</template>
                <template #description>
                  剩余 {{ slot.remaining }} 张 / 共 {{ slot.max_capacity }} 张
                </template>
              </a-list-item-meta>
              <a-progress
                :percent="Math.round(slot.sold_count / slot.max_capacity * 100)"
                :show-info="false"
                :stroke-color="getProgressColor(slot)"
                style="width: 150px"
              />
            </a-list-item>
          </a-list>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="24">
        <a-card title="今日每小时入园分布" class="chart-card">
          <div ref="barChart" class="bar-chart"></div>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import * as echarts from 'echarts'
import { getDashboard, getHourlyStats } from '@/api'

const gaugeChart = ref(null)
const barChart = ref(null)
let gaugeChartInstance = null
let barChartInstance = null

const dashboardData = ref({
  in_park_count: 0,
  max_capacity: 8000,
  today_check_in: 0,
  today_check_out: 0,
  today_sold: 0,
  slot_info: []
})

const hourlyData = ref([])

let timer = null

function getProgressColor(slot) {
  const percent = slot.sold_count / slot.max_capacity
  if (percent >= 0.9) return '#f5222d'
  if (percent >= 0.7) return '#faad14'
  return '#52c41a'
}

async function loadDashboard() {
  try {
    const res = await getDashboard()
    dashboardData.value = res
    updateGaugeChart()
  } catch (e) {}
}

async function loadHourlyStats() {
  try {
    const res = await getHourlyStats()
    hourlyData.value = res
    updateBarChart()
  } catch (e) {}
}

function updateGaugeChart() {
  if (!gaugeChartInstance) return
  const percent = dashboardData.value.max_capacity > 0
    ? (dashboardData.value.in_park_count / dashboardData.value.max_capacity) * 100
    : 0
  gaugeChartInstance.setOption({
    series: [{
      type: 'gauge',
      startAngle: 90,
      endAngle: -270,
      pointer: { show: false },
      progress: {
        show: true,
        overlap: false,
        roundCap: true,
        clip: false,
        itemStyle: {
          color: percent > 90 ? '#f5222d' : percent > 70 ? '#faad14' : '#1890ff'
        }
      },
      axisLine: {
        lineStyle: {
          width: 20,
          color: [[1, '#f0f0f0']]
        }
      },
      splitLine: { show: false },
      axisTick: { show: false },
      axisLabel: { show: false },
      data: [{ value: percent }],
      detail: { show: false }
    }]
  })
}

function updateBarChart() {
  if (!barChartInstance) return
  const hours = hourlyData.value.map(item => item.hour + '时')
  const counts = hourlyData.value.map(item => item.count)

  barChartInstance.setOption({
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '10%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: hours,
      axisLabel: { interval: 0 }
    },
    yAxis: {
      type: 'value',
      name: '人数'
    },
    series: [{
      data: counts,
      type: 'bar',
      itemStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: '#1890ff' },
          { offset: 1, color: '#69c0ff' }
        ]),
        borderRadius: [4, 4, 0, 0]
      }
    }]
  })
}

function initCharts() {
  if (gaugeChart.value) {
    gaugeChartInstance = echarts.init(gaugeChart.value)
    updateGaugeChart()
  }
  if (barChart.value) {
    barChartInstance = echarts.init(barChart.value)
    updateBarChart()
  }
}

function handleResize() {
  gaugeChartInstance?.resize()
  barChartInstance?.resize()
}

onMounted(async () => {
  await loadDashboard()
  await loadHourlyStats()
  await nextTick()
  initCharts()
  window.addEventListener('resize', handleResize)

  timer = setInterval(() => {
    loadDashboard()
    loadHourlyStats()
  }, 30000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
  window.removeEventListener('resize', handleResize)
  gaugeChartInstance?.dispose()
  barChartInstance?.dispose()
})
</script>

<style scoped>
.dashboard {
  padding: 0;
}

.stat-card {
  text-align: center;
}

.chart-card {
  height: 100%;
}

.in-park-container {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px 0;
}

.gauge-chart {
  width: 220px;
  height: 220px;
}

.in-park-info {
  margin-left: 30px;
  text-align: center;
}

.in-park-count {
  font-size: 48px;
  font-weight: bold;
  color: #1890ff;
  line-height: 1;
}

.in-park-label {
  font-size: 16px;
  color: #666;
  margin-top: 8px;
}

.in-park-max {
  font-size: 13px;
  color: #999;
  margin-top: 12px;
}

.bar-chart {
  height: 300px;
  width: 100%;
}
</style>
