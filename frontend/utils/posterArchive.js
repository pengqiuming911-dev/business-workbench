// 喜报下载后回传服务端归档(PNG + 字段 + content_hash),可复现。
export async function sha256Hex(b64) {
  const bin = atob(b64)
  const buf = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) buf[i] = bin.charCodeAt(i)
  const digest = await crypto.subtle.digest('SHA-256', buf)
  return Array.from(new Uint8Array(digest)).map(b => b.toString(16).padStart(2, '0')).join('')
}

export async function archivePoster(artifact, dataUrl) {
  if (!artifact || !dataUrl) return
  try {
    const b64 = dataUrl.split(',')[1]
    const hash = await sha256Hex(b64)
    const res = await fetch('/api/posters/artifact', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        product_id: artifact.product_id,
        observation_date: artifact.observation_date,
        fields: artifact,
        png_base64: b64,
        content_hash: hash,
      }),
    })
    if (!res.ok) console.error('喜报归档失败:', res.status, await res.text().catch(() => ''))
  } catch (e) {
    console.error('喜报归档失败:', e)
  }
}
