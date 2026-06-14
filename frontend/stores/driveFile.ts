import { reactive } from 'vue'

const SHEET_TOKEN = 'HdxnsNXSQhKoSItKiLwcnEmjn8b'
const SHEET_NAME = '航班服务交易总表'

const SHEET_TABS: Record<string, string> = {
  '产品表': '3JiyjX',
}

export const driveFile = reactive<{
  token: string
  name: string
  loaded: boolean
  sheets: Record<string, any[][]>
  loading: boolean
  error: string
}>({
  token: SHEET_TOKEN,
  name: SHEET_NAME,
  loaded: false,
  sheets: {},
  loading: false,
  error: '',
})

export async function loadWorkbook() {
  if (driveFile.loading) return
  driveFile.loading = true
  driveFile.loaded = false
  driveFile.error = ''
  driveFile.sheets = {}

  try {
    for (const [sheetName, sheetId] of Object.entries(SHEET_TABS)) {
      const url = `/api/drive/sheet-data?sheet_token=${driveFile.token}&sheet_id=${encodeURIComponent(sheetId)}`
      const res = await fetch(url)
      const data = await res.json()
      if (!res.ok) {
        throw new Error(data.error || `读取「${sheetName}」失败（${res.status}）`)
      }
      driveFile.sheets[sheetName] = data.rows || []
      console.log(`已加载「${sheetName}」：${driveFile.sheets[sheetName].length} 行`)
    }
    driveFile.loaded = true
    console.log('数据源加载完成：', SHEET_NAME)
  } catch (e) {
    driveFile.error = (e as Error).message
    console.error('加载数据源失败：', (e as Error).message)
  } finally {
    driveFile.loading = false
  }
}
