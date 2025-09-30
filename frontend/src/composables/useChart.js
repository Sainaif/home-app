import { onMounted, onUnmounted, ref } from 'vue'
import * as echarts from 'echarts'

/**
 * Composable for managing ECharts instances
 * Handles initialization, updates, and cleanup
 */
export function useChart(chartRef, getOptions) {
  const chartInstance = ref(null)

  onMounted(() => {
    console.log('[useChart] onMounted, chartRef exists:', !!chartRef.value)
    if (chartRef.value) {
      // Initialize chart with dark theme
      console.log('[useChart] Initializing chart on element:', chartRef.value)
      chartInstance.value = echarts.init(chartRef.value, null, {
        renderer: 'canvas',
      })
      console.log('[useChart] Chart instance created:', !!chartInstance.value)

      // Set initial options
      if (getOptions) {
        const options = getOptions()
        chartInstance.value.setOption(options)
      }

      // Handle window resize
      const resizeHandler = () => {
        chartInstance.value?.resize()
      }
      window.addEventListener('resize', resizeHandler)

      // Store cleanup function
      chartRef.value._resizeHandler = resizeHandler
    } else {
      console.warn('[useChart] chartRef.value is null on mount')
    }
  })

  onUnmounted(() => {
    // Cleanup
    if (chartRef.value?._resizeHandler) {
      window.removeEventListener('resize', chartRef.value._resizeHandler)
    }
    chartInstance.value?.dispose()
  })

  const updateChart = (options) => {
    console.log('[useChart] updateChart called, chartInstance exists:', !!chartInstance.value)
    if (chartInstance.value) {
      console.log('[useChart] Setting chart options')
      chartInstance.value.setOption(options, true)
    } else {
      console.warn('[useChart] Chart instance not initialized yet')
    }
  }

  const showLoading = () => {
    chartInstance.value?.showLoading('default', {
      text: 'Ładowanie...',
      color: '#9333ea',
      textColor: '#fff',
      maskColor: 'rgba(0, 0, 0, 0.8)',
    })
  }

  const hideLoading = () => {
    chartInstance.value?.hideLoading()
  }

  return {
    chartInstance,
    updateChart,
    showLoading,
    hideLoading,
  }
}

/**
 * Generate chart options for prediction forecast
 * @param {Object} data - Prediction data with historical and predicted data
 * @param {string} target - Target name (electricity, gas, etc.)
 */
export function getPredictionChartOptions(data, target) {
  const { historicalDates, historicalValues, predictedDates, predictedValues, lowerBound, upperBound } = data

  // Combine historical and predicted dates for x-axis
  const allDates = [...historicalDates, ...predictedDates]

  return {
    backgroundColor: 'transparent',
    title: {
      text: `Prognoza: ${target}`,
      left: 'center',
      textStyle: {
        color: '#fff',
        fontSize: 18,
        fontWeight: 'bold',
      },
    },
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(0, 0, 0, 0.9)',
      borderColor: '#9333ea',
      borderWidth: 1,
      textStyle: {
        color: '#fff',
      },
      axisPointer: {
        type: 'cross',
        label: {
          backgroundColor: '#9333ea',
        },
      },
      formatter: (params) => {
        let tooltip = `<div style="font-weight: bold; margin-bottom: 5px;">${params[0].axisValue}</div>`
        params.forEach((param) => {
          tooltip += `<div style="display: flex; align-items: center; margin: 3px 0;">
            <span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: ${param.color}; margin-right: 5px;"></span>
            <span>${param.seriesName}: ${param.value?.toFixed(2) || '-'}</span>
          </div>`
        })
        return tooltip
      },
    },
    legend: {
      data: ['Historia', 'Prognoza', 'Dolny przedział', 'Górny przedział'],
      top: 35,
      textStyle: {
        color: '#9ca3af',
      },
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: 80,
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: allDates,
      axisLine: {
        lineStyle: {
          color: '#4b5563',
        },
      },
      axisLabel: {
        color: '#9ca3af',
        formatter: (value) => {
          const date = new Date(value)
          return date.toLocaleDateString('pl-PL', { month: 'short', year: '2-digit' })
        },
      },
    },
    yAxis: {
      type: 'value',
      axisLine: {
        lineStyle: {
          color: '#4b5563',
        },
      },
      axisLabel: {
        color: '#9ca3af',
        formatter: (value) => value.toFixed(2),
      },
      splitLine: {
        lineStyle: {
          color: '#374151',
          type: 'dashed',
        },
      },
    },
    series: [
      // Historical data line
      {
        name: 'Historia',
        type: 'line',
        data: [...historicalValues, ...Array(predictedDates.length).fill(null)],
        lineStyle: {
          color: '#3b82f6',
          width: 2,
        },
        itemStyle: {
          color: '#3b82f6',
        },
        symbol: 'circle',
        symbolSize: 4,
      },
      // Confidence interval area (lower to upper) - only for predicted portion
      {
        name: 'Przedział ufności',
        type: 'line',
        data: [...Array(historicalDates.length).fill(null), ...lowerBound],
        lineStyle: {
          opacity: 0,
        },
        stack: 'confidence',
        symbol: 'none',
        areaStyle: {
          color: 'rgba(147, 51, 234, 0.2)',
        },
        showInLegend: false,
      },
      {
        name: 'Przedział ufności',
        type: 'line',
        data: [...Array(historicalDates.length).fill(null), ...upperBound.map((upper, i) => upper - lowerBound[i])],
        lineStyle: {
          opacity: 0,
        },
        stack: 'confidence',
        symbol: 'none',
        areaStyle: {
          color: 'rgba(147, 51, 234, 0.2)',
        },
        showInLegend: false,
      },
      // Lower bound line - only for predicted portion
      {
        name: 'Dolny przedział',
        type: 'line',
        data: [...Array(historicalDates.length).fill(null), ...lowerBound],
        lineStyle: {
          color: '#ec4899',
          width: 1,
          type: 'dashed',
        },
        symbol: 'none',
      },
      // Upper bound line - only for predicted portion
      {
        name: 'Górny przedział',
        type: 'line',
        data: [...Array(historicalDates.length).fill(null), ...upperBound],
        lineStyle: {
          color: '#ec4899',
          width: 1,
          type: 'dashed',
        },
        symbol: 'none',
      },
      // Prediction line (main) - only for predicted portion
      {
        name: 'Prognoza',
        type: 'line',
        data: [...Array(historicalDates.length).fill(null), ...predictedValues],
        lineStyle: {
          color: '#9333ea',
          width: 3,
        },
        itemStyle: {
          color: '#9333ea',
        },
        symbol: 'circle',
        symbolSize: 6,
        emphasis: {
          itemStyle: {
            color: '#a855f7',
            borderColor: '#fff',
            borderWidth: 2,
          },
        },
      },
    ],
  }
}