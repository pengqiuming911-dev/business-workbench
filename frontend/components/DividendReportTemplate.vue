<template>
  <div class="poster-wrapper">
    <div ref="stageRef" class="stage" aria-label="分红观察喜报">
      <div class="logo">衍选</div>
      <div class="bg-paper one"></div>
      <div class="bg-paper two"></div>

      <section class="paper">
        <div class="top-line"></div>
        <div class="title">{{ fields.title }}</div>
        <div class="subtitle">{{ fields.subtitle }}</div>
        <div class="outer-border"></div>
        <div class="congrats">{{ fields.congrats }}</div>
        <div class="message-box">
          <div class="congrat-text">{{ fields.congrat_text_prefix }}&nbsp; {{ fields.product_name }}</div>
          <div class="money">💴</div>
          <div class="date-row">派息观察日:{{ fields.observation_date_display }}</div>
        </div>
        <div class="thick-line"></div>
        <div class="label-yield">{{ fields.label_yield }}</div>
        <div class="yield">{{ fields.annualized_return }}<span class="pct">%</span></div>
        <div class="stat-label left">{{ fields.label_cumulative }} {{ fields.dividend_count }}次:</div>
        <div class="stat-label right">{{ fields.label_monthly }}</div>
        <div class="stat-num left">{{ fields.cumulative_dividend_rate }}<span class="pct">%</span></div>
        <div class="stat-num right">{{ fields.monthly_coupon }}<span class="pct">%</span></div>
        <div class="dash"></div>
        <div class="info">
          挂钩标的: {{ fields.underlying_name }}<br>
          ✅️入场价:【{{ fields.entry_price }}】<br>
          派息界限: {{ fields.dividend_barrier_value }}<br>
          止盈界限: {{ fields.knockout_value }}<br>
          末月降至: {{ fields.parachute_value }}<br>
          入场时间: {{ fields.entry_date_display }}<br>
          <div class="note">{{ fields.disclaimer }}</div>
        </div>
        <div class="qr" aria-label="二维码装饰">
          <b class="finder f1"></b><b class="finder f2"></b><b class="finder f3"></b>
          <i style="grid-column:6;grid-row:1"></i><i style="grid-column:8;grid-row:1"></i><i style="grid-column:6;grid-row:2"></i><i style="grid-column:8;grid-row:3"></i><i style="grid-column:6;grid-row:5"></i><i style="grid-column:7;grid-row:5"></i><i style="grid-column:9;grid-row:5"></i><i style="grid-column:11;grid-row:5"></i><i style="grid-column:13;grid-row:5"></i>
          <i style="grid-column:1;grid-row:6"></i><i style="grid-column:3;grid-row:6"></i><i style="grid-column:5;grid-row:6"></i><i style="grid-column:7;grid-row:6"></i><i style="grid-column:10;grid-row:6"></i><i style="grid-column:12;grid-row:6"></i>
          <i style="grid-column:2;grid-row:7"></i><i style="grid-column:4;grid-row:7"></i><i style="grid-column:6;grid-row:7"></i><i style="grid-column:8;grid-row:7"></i><i style="grid-column:9;grid-row:7"></i><i style="grid-column:13;grid-row:7"></i>
          <i style="grid-column:1;grid-row:8"></i><i style="grid-column:5;grid-row:8"></i><i style="grid-column:7;grid-row:8"></i><i style="grid-column:11;grid-row:8"></i><i style="grid-column:12;grid-row:8"></i>
          <i style="grid-column:3;grid-row:9"></i><i style="grid-column:6;grid-row:9"></i><i style="grid-column:8;grid-row:9"></i><i style="grid-column:10;grid-row:9"></i><i style="grid-column:13;grid-row:9"></i>
          <i style="grid-column:6;grid-row:10"></i><i style="grid-column:8;grid-row:10"></i><i style="grid-column:10;grid-row:10"></i><i style="grid-column:12;grid-row:10"></i>
          <i style="grid-column:5;grid-row:11"></i><i style="grid-column:7;grid-row:11"></i><i style="grid-column:9;grid-row:11"></i><i style="grid-column:11;grid-row:11"></i><i style="grid-column:13;grid-row:11"></i>
          <i style="grid-column:6;grid-row:12"></i><i style="grid-column:8;grid-row:12"></i><i style="grid-column:10;grid-row:12"></i><i style="grid-column:12;grid-row:12"></i>
          <i style="grid-column:5;grid-row:13"></i><i style="grid-column:7;grid-row:13"></i><i style="grid-column:9;grid-row:13"></i><i style="grid-column:11;grid-row:13"></i>
        </div>
        <div class="qr-caption">{{ fields.qr_caption }}</div>
      </section>
    </div>

    <div class="poster-actions">
      <button class="btn-download" :disabled="isGenerating" @click="downloadPng">
        {{ isGenerating ? '生成中...' : '下载图片' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick } from 'vue'
import html2canvas from 'html2canvas-pro'

const props = defineProps({
  fields: { type: Object, required: true },
})

const emit = defineEmits(['downloaded'])

const stageRef = ref(null)
const isGenerating = ref(false)

async function getPngDataUrl() {
  if (!stageRef.value) return null
  await nextTick()
  await document.fonts.ready
  const canvas = await html2canvas(stageRef.value, {
    scale: 2,
    backgroundColor: null,
    useCORS: true,
    logging: false,
  })
  return canvas.toDataURL('image/png')
}

async function downloadPng() {
  if (!stageRef.value || isGenerating.value) return
  isGenerating.value = true
  try {
    const dataUrl = await getPngDataUrl()
    if (!dataUrl) return
    const name = (props.fields.product_name || '产品').replace(/[\\/:*?"<>|]/g, '_')
    const link = document.createElement('a')
    link.download = `分红观察喜报_${name}_${props.fields.observation_date_display || props.fields.observation_date || ''}.png`
    link.href = dataUrl
    link.click()
    emit('downloaded', dataUrl)
  } catch (e) {
    console.error('生成喜报图片失败:', e)
  } finally {
    isGenerating.value = false
  }
}

defineExpose({ downloadPng, getPngDataUrl })
</script>

<style scoped>
.stage{
  --red:#cf141d;
  --deep-red:#c71018;
  --paper:#f5f0e7;
  --ink:#111;
}
*{box-sizing:border-box}
.stage{
  position:relative;
  width:819px;
  height:1456px;
  overflow:hidden;
  background:var(--red);
}
.logo{
  position:absolute;
  z-index:10;
  left:7px;
  top:8px;
  width:100px;
  height:99px;
  border-radius:17px;
  background:linear-gradient(160deg,#43a2ff 0%,#1979ee 62%,#1167dd 100%);
  box-shadow:0 3px 7px rgba(0,0,0,.12);
  color:#fff;
  display:flex;
  align-items:center;
  justify-content:center;
  font-weight:900;
  font-size:33px;
  letter-spacing:-4px;
  text-shadow:0 1px 2px rgba(0,0,0,.2);
}
.bg-paper{
  position:absolute;
  background:#f6f0e6;
  border:2px solid rgba(199,16,24,.38);
  box-shadow:0 7px 18px rgba(0,0,0,.22);
}
.bg-paper.one{
  width:750px;height:1268px;left:-48px;top:70px;transform:rotate(-8deg);
}
.bg-paper.two{
  width:747px;height:1246px;right:-118px;top:208px;transform:rotate(8deg);
}
.bg-paper::before{
  content:"";position:absolute;left:22px;top:45px;right:22px;height:7px;background:var(--red);border-radius:4px;
}
.bg-paper::after{
  content:"";position:absolute;left:24px;top:92px;width:160px;height:75px;
  border-top:5px solid rgba(199,16,24,.35);
  border-bottom:5px solid rgba(199,16,24,.35);
  opacity:.75;
}
.paper{
  position:absolute;
  left:38px;
  top:112px;
  width:746px;
  height:1327px;
  background:
    radial-gradient(circle at 20% 10%,rgba(255,255,255,.45),transparent 32%),
    radial-gradient(circle at 74% 55%,rgba(255,255,255,.26),transparent 25%),
    repeating-linear-gradient(0deg,rgba(70,52,30,.023) 0 2px,transparent 2px 5px),
    linear-gradient(100deg,#f7f2e9,#f1eadf 58%,#faf6ee);
  box-shadow:0 11px 19px rgba(0,0,0,.28);
  transform:rotate(-.2deg);
  color:var(--ink);
}
.outer-border{
  position:absolute;left:25px;right:25px;top:225px;bottom:33px;
  border:2px solid var(--red);
  box-shadow:inset 0 0 0 3px rgba(207,20,29,.16);
}
.top-line{
  position:absolute;left:20px;right:22px;top:18px;height:7px;background:var(--red);border-radius:8px;
}
.title{
  position:absolute;left:23px;right:16px;top:48px;
  color:var(--red);font-size:75px;line-height:1;font-weight:900;letter-spacing:1px;
  white-space:nowrap;
}
.subtitle{
  position:absolute;left:52px;right:52px;top:168px;height:22px;
  border-top:2px solid rgba(207,20,29,.55);border-bottom:2px solid rgba(207,20,29,.35);
  color:var(--red);font-family:Georgia,serif;font-size:16px;font-weight:700;letter-spacing:17px;text-align:center;line-height:19px;
}
.congrats{
  position:absolute;left:300px;top:217px;height:24px;min-width:136px;background:var(--red);color:#fff;
  font-family:Georgia,serif;font-size:13px;font-weight:700;letter-spacing:3px;text-align:center;line-height:24px;border-radius:2px;padding:0 10px;
}
.congrats:before,.congrats:after{content:"";position:absolute;top:0;border-top:12px solid transparent;border-bottom:12px solid transparent}
.congrats:before{left:-14px;border-right:14px solid var(--red)}
.congrats:after{right:-14px;border-left:14px solid var(--red)}
.message-box{
  position:absolute;left:58px;top:245px;width:638px;height:190px;border:3px solid #333;background:rgba(255,255,255,.18);
  box-shadow:inset 0 0 0 1px rgba(0,0,0,.16);
}
.congrat-text{
  position:absolute;left:47px;right:28px;top:26px;font-size:43px;line-height:1.18;font-weight:500;white-space:nowrap;
}
.date-row{position:absolute;left:125px;top:118px;font-size:34px;line-height:1.1;white-space:nowrap;}
.money{position:absolute;left:26px;top:100px;font-size:48px;filter:saturate(1.05)}
.thick-line{position:absolute;left:55px;right:43px;top:454px;border-top:5px solid #222;opacity:.75;box-shadow:0 10px 0 rgba(0,0,0,.45)}
.label-yield{position:absolute;left:51px;top:497px;font-size:43px;font-weight:500;}
.yield{position:absolute;left:213px;top:635px;color:var(--red);font-size:120px;font-weight:300;letter-spacing:12px;line-height:1;white-space:nowrap;}
.yield .pct{font-size:52px;letter-spacing:0;margin-left:-7px;font-weight:400;}
.stat-label{position:absolute;top:872px;font-size:29px;font-weight:500;}
.stat-label.left{left:52px}.stat-label.right{left:420px}
.stat-num{position:absolute;top:935px;color:var(--red);font-size:84px;font-weight:300;line-height:1;white-space:nowrap;}
.stat-num.left{left:50px}.stat-num.right{left:438px}.stat-num .pct{font-size:35px;margin-left:-9px;font-weight:400;}
.dash{position:absolute;left:52px;right:48px;top:1058px;border-top:2px dashed #777;opacity:.8;}
.info{position:absolute;left:61px;top:1077px;font-size:24px;line-height:1.46;color:#222;}
.note{font-size:16px;line-height:1.25;margin-top:4px;white-space:nowrap;}
.qr{
  position:absolute;left:548px;top:1072px;width:174px;height:174px;background:#fff;border:7px solid #fff;box-shadow:0 0 0 1px #ddd;
  display:grid;grid-template-columns:repeat(13,1fr);grid-template-rows:repeat(13,1fr);gap:2px;padding:6px;
}
.qr i{background:#1d1d1d;display:block;}
.qr i.empty{background:transparent}.qr .finder{grid-column:span 4;grid-row:span 4;background:#fff;border:7px solid #111;position:relative}.qr .finder:after{content:"";position:absolute;inset:10px;background:#111}.qr .f1{grid-column:1/5;grid-row:1/5}.qr .f2{grid-column:10/14;grid-row:1/5}.qr .f3{grid-column:1/5;grid-row:10/14}
.qr-caption{position:absolute;left:552px;top:1260px;font-size:25px;white-space:nowrap;}
[contenteditable="true"]{outline:none}
[contenteditable="true"]:focus{box-shadow:0 0 0 2px rgba(207,20,29,.35);background:rgba(255,255,255,.35)}

.poster-wrapper{ display:flex; flex-direction:column; align-items:center; }
.poster-actions{ margin-top:12px; }
.btn-download{ padding:8px 16px; background:#cf141d; color:#fff; border:none; border-radius:6px; cursor:pointer; }
.btn-download:disabled{ opacity:.6; cursor:default; }
</style>

