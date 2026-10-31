<!--
  赞助作者 · 通用悬浮组件（Vue3 + Element Plus）
  ── 每个项目只需改这 5 处（见 props 默认值与 ADAPT 注释）──
  1) qrReward / qrWechat  二维码图片路径
  2) storageKey           首次自动弹出的标记名
  3) hiddenRoutes         哪些路由不显示（谱书/打印/预览）
  4) 文案                 赞助说明
  5) 位置/尺寸             默认右侧垂直居中
-->
<template>
  <div class="sponsor-wrap">
    <button class="sponsor-fab" type="button" @click="open">
      <span v-for="(c, i) in labelChars" :key="i">{{ c }}</span>
    </button>

    <el-dialog v-model="visible" title="赞助作者" width="460px" align-center append-to-body
               class="sponsor-dialog" :close-on-click-modal="true">
      <el-tabs v-model="tab">
        <el-tab-pane label="赞助支持" name="reward">
          <div class="qr-box">
            <img class="qr" :src="qrReward" alt="赞助收款码" />
            <p class="tip">{{ rewardTip }}</p>
          </div>
        </el-tab-pane>
        <el-tab-pane label="加我微信" name="wechat">
          <div class="qr-box">
            <img class="qr" :src="qrWechat" alt="作者微信二维码" />
            <p class="tip">{{ wechatTip }}</p>
          </div>
        </el-tab-pane>
      </el-tabs>
      <p class="foot">赞助完全自愿，不影响功能使用</p>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { useRoute } from 'vue-router'
import qrReward from '@/assets/reward-qr.png'   // ADAPT ①：换成项目内路径
import qrWechat from '@/assets/wechat-qr.png'   // ADAPT ①

// ADAPT ②：每个项目唯一，避免和别的项目共用标记
const props = defineProps({
  storageKey: { type: String, default: 'sponsor_shown_v1' },
  // ADAPT ③：命中这些路由前缀时不显示
  hiddenRoutes: { type: Array, default: () => [] },
  label: { type: String, default: '赞助作者' },
  // ADAPT ④
  rewardTip: { type: String, default: '微信扫码赞助，金额随意，完全自愿' },
  wechatTip: { type: String, default: '扫码添加作者微信，交流反馈' },
  autoOpenDelay: { type: Number, default: 800 }
})

const visible = ref(false)
const tab = ref('reward')
const route = useRoute()
const labelChars = props.label.split('')
let timer = null

const hidden = () => props.hiddenRoutes.some(p => route.path.startsWith(p))

function open () { visible.value = true }
function close () { visible.value = false }

// 首次进入自动弹一次（localStorage 不可用则不弹、不报错）
onMounted(() => {
  if (hidden()) return
  try {
    if (localStorage.getItem(props.storageKey) === '1') return
  } catch (e) { return }
  timer = setTimeout(() => {
    try { localStorage.setItem(props.storageKey, '1') } catch (e) {}
    if (!hidden()) visible.value = true
  }, props.autoOpenDelay)
})
onBeforeUnmount(() => timer && clearTimeout(timer))
</script>

<style scoped>
/* ADAPT ⑤：位置与尺寸（默认右侧垂直居中、竖排四字） */
.sponsor-wrap { position: fixed; inset: 0 auto 0 auto; right: 0; top: 0; pointer-events: none; z-index: 1000; }
.sponsor-fab {
  position: fixed; right: 10px; top: 50%; transform: translateY(-50%);
  min-width: 52px; padding: 18px 14px; display: flex; flex-direction: column;
  align-items: center; gap: 2px; border: 1px solid rgba(180,150,90,.45);
  background: linear-gradient(180deg,#fdf6e3,#f3e3bd); color: #7a5c1f;
  border-radius: 12px; cursor: pointer; pointer-events: auto;
  font-size: 14px; line-height: 1.6; letter-spacing: 4px;
  box-shadow: 0 4px 14px rgba(0,0,0,.12); transition: .18s;
}
.sponsor-fab:hover { transform: translateY(-50%) translateX(-2px); background: linear-gradient(180deg,#fdf0d0,#ecd39f); }
.qr-box { text-align: center; min-height: 372px; display: flex; flex-direction: column; justify-content: center; gap: 12px; }
.qr { width: 240px; max-width: 100%; margin: 0 auto; display: block; }
.tip { color: #606266; font-size: 14px; margin: 0; }
.foot { text-align: center; color: #909399; font-size: 12px; margin: 6px 0 0; }

/* 打印/导出 PDF 绝不出现 */
@media print {
  .sponsor-wrap, .sponsor-fab, .sponsor-dialog,
  .el-overlay, .el-overlay-dialog { display: none !important; }
}
</style>
