const dbModule = require('../db')
const { fetchLatestPrice } = require('./priceService')

const TOOL_DEFINITIONS = [
  {
    type: 'function',
    function: {
      name: 'search_products',
      description: '根据关键词搜索产品名称。返回匹配的产品列表（id、名称、存续状态）。用于用户模糊查找某个产品时使用。',
      parameters: {
        type: 'object',
        properties: {
          keyword: { type: 'string', description: '搜索关键词，如产品名称的一部分' },
        },
        required: ['keyword'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'get_product_detail',
      description: '获取指定产品的详细信息，包括产品结构、标的、入场价、敲出线、派息线等全部字段。当用户询问某个具体产品的详情时使用。',
      parameters: {
        type: 'object',
        properties: {
          product_id: { type: 'string', description: '产品 ID（航班编号）' },
        },
        required: ['product_id'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'get_observations',
      description: '获取指定产品的观察日记录，包括每个月的观察日期、敲出价、派息线、标的价格、是否敲出/派息等。当用户询问产品的观察结果或历史表现时使用。',
      parameters: {
        type: 'object',
        properties: {
          product_id: { type: 'string', description: '产品 ID（航班编号）' },
        },
        required: ['product_id'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'get_price',
      description: '获取标的证券的最新实时价格（从东方财富 API 获取）。当用户询问某个标的的当前价格时使用。',
      parameters: {
        type: 'object',
        properties: {
          code: { type: 'string', description: '标的代码，如 sh000300、sz399006' },
        },
        required: ['code'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'get_dashboard_stats',
      description: '获取业务总览统计数据，包括产品总数、存续产品数、客户总数、渠道总数等。当用户询问整体业务情况或统计数据时使用。',
      parameters: {
        type: 'object',
        properties: {},
      },
    },
  },
]

async function executeTool(name, args) {
  try {
    switch (name) {
      case 'search_products':
        return executeSearchProducts(args)
      case 'get_product_detail':
        return executeGetProductDetail(args)
      case 'get_observations':
        return executeGetObservations(args)
      case 'get_price':
        return executeGetPrice(args)
      case 'get_dashboard_stats':
        return executeGetDashboardStats(args)
      default:
        return { error: `Unknown tool: ${name}` }
    }
  } catch (err) {
    return { error: err.message }
  }
}

function executeSearchProducts({ keyword }) {
  const db = dbModule.db
  const like = `%${keyword}%`
  const rows = db.exec(
    'SELECT id, name, holding_status, code FROM products WHERE name LIKE ? LIMIT 10',
    [like]
  )[0]?.values || []
  return {
    count: rows.length,
    products: rows.map(r => ({
      id: r[0], name: r[1], holding_status: r[2], code: r[3],
    })),
  }
}

function executeGetProductDetail({ product_id }) {
  const db = dbModule.db
  const rows = db.exec('SELECT * FROM products WHERE id = ?', [product_id])
  if (!rows[0] || rows[0].values.length === 0) {
    return { error: `Product ${product_id} not found` }
  }
  const columns = rows[0].columns
  const values = rows[0].values[0]
  const product = {}
  columns.forEach((col, i) => { product[col] = values[i] })
  return { product }
}

function executeGetObservations({ product_id }) {
  const observations = dbModule.queryObservationsByProduct(product_id)
  return {
    product_id,
    count: observations.length,
    observations: observations.map(o => ({
      observation_date: o.observation_date,
      knockout_price: o.knockout_price,
      dividend_line: o.dividend_line,
      underlying_price: o.underlying_price,
      is_knocked_out: o.is_knocked_out,
      is_dividend: o.is_dividend,
    })),
  }
}

async function executeGetPrice({ code }) {
  const price = await fetchLatestPrice(code)
  return { code, price, fetched_at: new Date().toISOString() }
}

function executeGetDashboardStats() {
  const db = dbModule.db
  const totalProducts = db.exec('SELECT COUNT(*) FROM products')[0]?.values[0][0] || 0
  const ongoingProducts = db.exec("SELECT COUNT(*) FROM products WHERE holding_status = '存续'")[0]?.values[0][0] || 0
  const completedProducts = db.exec("SELECT COUNT(*) FROM products WHERE holding_status = '已结束'")[0]?.values[0][0] || 0
  const totalCustomers = db.exec('SELECT COUNT(*) FROM customers')[0]?.values[0][0] || 0
  const totalChannels = db.exec('SELECT COUNT(*) FROM channels')[0]?.values[0][0] || 0
  return { totalProducts, ongoingProducts, completedProducts, totalCustomers, totalChannels }
}

module.exports = { TOOL_DEFINITIONS, executeTool }
