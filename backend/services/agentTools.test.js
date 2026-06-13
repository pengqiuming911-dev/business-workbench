const assert = require('node:assert/strict')
const { TOOL_DEFINITIONS, executeTool } = require('./agentTools')

assert(Array.isArray(TOOL_DEFINITIONS))
assert(TOOL_DEFINITIONS.length >= 5, `Expected >=5 tools, got ${TOOL_DEFINITIONS.length}`)

const names = TOOL_DEFINITIONS.map(t => t.function.name)
assert(names.includes('search_products'), 'Missing search_products tool')
assert(names.includes('get_product_detail'), 'Missing get_product_detail tool')
assert(names.includes('get_observations'), 'Missing get_observations tool')
assert(names.includes('get_price'), 'Missing get_price tool')
assert(names.includes('get_dashboard_stats'), 'Missing get_dashboard_stats tool')

for (const tool of TOOL_DEFINITIONS) {
  assert(tool.type === 'function', `Tool ${tool.function.name} must have type "function"`)
  assert(tool.function.name, 'Tool must have a name')
  assert(tool.function.description, `Tool ${tool.function.name} missing description`)
  assert(tool.function.parameters, `Tool ${tool.function.name} missing parameters`)
}

console.log('agentTools tests passed')
