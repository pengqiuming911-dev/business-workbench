const dbModule = require('../db')

function searchDocs(query, limit = 5) {
  if (!query || !query.trim()) return []
  const keywords = query.trim().split(/\s+/).filter(k => k.length > 0)
  const docs = dbModule.getAllProductDocs()
  const scored = []

  for (const doc of docs) {
    const text = `${doc.doc_name} ${doc.parent_path} ${doc.raw_content || ''}`
    const lower = text.toLowerCase()
    let score = 0
    for (const kw of keywords) {
      const lowerKw = kw.toLowerCase()
      let idx = lower.indexOf(lowerKw)
      while (idx !== -1) {
        score += 1
        idx = lower.indexOf(lowerKw, idx + 1)
      }
    }
    if (score > 0) {
      let structure = null
      if (doc.structure_json) {
        try { structure = JSON.parse(doc.structure_json) } catch {}
      }
      scored.push({
        doc_name: doc.doc_name,
        parent_path: doc.parent_path,
        raw_content: doc.raw_content || '',
        structure,
        score,
      })
    }
  }

  scored.sort((a, b) => b.score - a.score)
  return scored.slice(0, limit)
}

function buildDocContext(docs) {
  if (!docs || docs.length === 0) return ''
  const parts = docs.map((d, i) => {
    const structStr = d.structure ? `\n结构信息：${JSON.stringify(d.structure)}` : ''
    return `[文档${i + 1}] ${d.doc_name} (${d.parent_path})\n${d.raw_content}${structStr}`
  })
  return `\n\n以下是与用户问题相关的文档资料，请参考这些文档回答问题：\n\n${parts.join('\n\n---\n\n')}`
}

module.exports = { searchDocs, buildDocContext }
