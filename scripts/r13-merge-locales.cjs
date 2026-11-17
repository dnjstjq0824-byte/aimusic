// R13 one-shot: merges new keyed strings into all 6 web locales.
// ko = source, en = translation, ja/zh-CN/th = machine-draft (Beta, G-L),
// en-XA = pseudo-localized from en (style: prior WIs).
// Usage: node scripts/r13-merge-locales.cjs [--check]
const fs = require('node:fs');
const path = require('node:path');

const LOCALES = ['ko', 'en', 'ja', 'zh-CN', 'th', 'en-XA'];
const DIR = path.join(__dirname, '..', 'web', 'i18n', 'locales');

// file -> dotted path -> { ko, en, ja, zh, th }
const KEYS = {
  common: {
    'common.actions.logout': { ko: '로그아웃', en: 'Log out', ja: 'ログアウト', zh: '退出登录', th: 'ออกจากระบบ' },
    'common.actions.reload': { ko: '새로고침', en: 'Reload', ja: '再読み込み', zh: '刷新', th: 'โหลดใหม่' },
    'common.paginator.prevShort': { ko: '이전', en: 'Back', ja: '戻る', zh: '上一页', th: 'ก่อนหน้า' },
    'common.paginator.nextShort': { ko: '다음', en: 'Next', ja: '次へ', zh: '下一页', th: 'ถัดไป' },
    'common.tabs.list': { ko: '탭', en: 'Tabs', ja: 'タブ', zh: '选项卡', th: 'แท็บ' },
    'common.httpError.home': { ko: '홈으로', en: 'Home', ja: 'ホームへ', zh: '回到首页', th: 'หน้าแรก' },
    'common.httpError.backHome': { ko: '홈으로 돌아가기', en: 'Back home', ja: 'ホームに戻る', zh: '返回首页', th: 'กลับหน้าแรก' },
    'common.httpError.exploreCta': { ko: '음원 탐색하기', en: 'Explore tracks', ja: '楽曲を探す', zh: '发现音乐', th: 'สำรวจเพลง' },
  },
  checkout: {
    'checkout.title': { ko: '이용권 결제', en: 'Pass checkout', ja: '利用券の購入', zh: '购买通行证', th: 'ชำระเงินพาส' },
    'checkout.desc': { ko: '선택한 요금제의 주문서를 확인하고 Toss Payments로 안전하게 결제하세요.', en: 'Review your plan order and pay securely with Toss Payments.', ja: '選択したプランの注文書を確認し、Toss Paymentsで安全にお支払いください。', zh: '请确认所选方案的订单，并通过 Toss Payments 安全支付。', th: 'ตรวจสอบคำสั่งซื้อของแผนที่เลือกและชำระเงินอย่างปลอดภัยด้วย Toss Payments' },
    'checkout.failTitle': { ko: '주문 처리 실패', en: 'Order failed', ja: '注文処理に失敗しました', zh: '订单处理失败', th: 'คำสั่งซื้อล้มเหลว' },
    'checkout.sheet': { ko: '주문서', en: 'Order', ja: '注文書', zh: '订单', th: 'ใบสั่งซื้อ' },
    'checkout.pending': { ko: '결제 대기', en: 'Awaiting payment', ja: '支払い待ち', zh: '等待支付', th: 'รอชำระเงิน' },
    'checkout.fItem': { ko: '상품', en: 'Item', ja: '商品', zh: '商品', th: 'สินค้า' },
    'checkout.fOrder': { ko: '주문번호', en: 'Order number', ja: '注文番号', zh: '订单号', th: 'หมายเลขคำสั่งซื้อ' },
    'checkout.fPeriod': { ko: '유효기간', en: 'Validity', ja: '有効期間', zh: '有效期', th: 'ระยะเวลาใช้งาน' },
    'checkout.periodValue': { ko: '결제일로부터 1개월 (자동연장 없음)', en: '1 month from payment (no auto-renewal)', ja: '支払日から1か月（自動更新なし）', zh: '自支付日起1个月（不自动续费）', th: '1 เดือนนับจากวันชำระเงิน (ไม่ต่ออายุอัตโนมัติ)' },
    'checkout.fAmount': { ko: '결제금액', en: 'Amount', ja: 'お支払い金額', zh: '支付金额', th: 'ยอดชำระ' },
    'checkout.consentTitle': { ko: '청약철회 및 환불 기준 동의 (필수)', en: 'Withdrawal and refund terms consent (required)', ja: 'クーリングオフ・返金基準への同意（必須）', zh: '同意撤回与退款标准（必填）', th: 'ยินยอมเงื่อนไขการยกเลิกและคืนเงิน (บังคับ)' },
    'checkout.consentAgree': { ko: '위 청약철회·환불 기준을 확인했으며 이에 동의합니다. (필수)', en: 'I have read and agree to the withdrawal/refund terms above. (Required)', ja: '上記のクーリングオフ・返金基準を確認し、同意します。（必須）', zh: '我已阅读并同意上述撤回/退款标准。（必填）', th: 'ฉันได้อ่านและยอมรับเงื่อนไขข้างต้นแล้ว (บังคับ)' },
    'checkout.preparing': { ko: '준비 중', en: 'Preparing', ja: '準備中', zh: '准备中', th: 'กำลังเตรียม' },
    'checkout.preparingDesc': { ko: '결제 준비 중입니다. 잠시 후 다시 시도해 주세요.', en: 'Payments are being prepared. Please try again shortly.', ja: '決済の準備中です。しばらくしてからお試しください。', zh: '支付准备中，请稍后重试。', th: 'กำลังเตรียมการชำระเงิน กรุณาลองอีกครั้งในภายหลัง' },
    'checkout.backToPricing': { ko: '요금제로 돌아가기', en: 'Back to pricing', ja: '料金プランに戻る', zh: '返回方案页', th: 'กลับไปหน้าค่าบริการ' },
    'checkout.payCta': { ko: 'Toss로 {amount} 결제하기', en: 'Pay {amount} with Toss', ja: 'Tossで{amount}を支払う', zh: '使用 Toss 支付 {amount}', th: 'ชำระ {amount} ด้วย Toss' },
    'checkout.errNoPlan': { ko: '요금제 코드가 없습니다. 요금제에서 다시 선택해 주세요.', en: 'No plan code. Please pick a plan again.', ja: 'プランコードがありません。料金プランから選び直してください。', zh: '缺少方案代码，请重新选择方案。', th: 'ไม่มีรหัสแผน กรุณาเลือกแผนอีกครั้ง' },
    'checkout.errPlanNotFound': { ko: '요금제를 찾을 수 없습니다. 요금제에서 다시 선택해 주세요.', en: 'Plan not found. Please pick a plan again.', ja: 'プランが見つかりません。料金プランから選び直してください。', zh: '未找到方案，请重新选择。', th: 'ไม่พบแผน กรุณาเลือกอีกครั้ง' },
    'checkout.payWindowErr': { ko: '결제창 오류', en: 'Payment window error', ja: '決済ウィンドウのエラー', zh: '支付窗口错误', th: 'ข้อผิดพลาดหน้าต่างชำระเงิน' },
    'checkout.payCancelled': { ko: '결제가 취소되었거나 실패했습니다. 다시 시도해 주세요.', en: 'Payment was cancelled or failed. Please try again.', ja: '決済がキャンセルされたか失敗しました。もう一度お試しください。', zh: '支付已取消或失败，请重试。', th: 'การชำระเงินถูกยกเลิกหรือไม่สำเร็จ กรุณาลองอีกครั้ง' },
    'checkout.cfApproving': { ko: '결제 승인 중…', en: 'Confirming payment…', ja: '決済承認中…', zh: '正在确认支付…', th: 'กำลังยืนยันการชำระเงิน…' },
    'checkout.cfDontClose': { ko: '창을 닫지 마세요.', en: "Don't close this window.", ja: 'ウィンドウを閉じないでください。', zh: '请勿关闭窗口。', th: 'อย่าปิดหน้าต่างนี้' },
    'checkout.cfDone': { ko: '결제가 완료되었습니다', en: 'Payment complete', ja: '決済が完了しました', zh: '支付完成', th: 'ชำระเงินเสร็จสมบูรณ์' },
    'checkout.cfPass': { ko: '이용권', en: 'Pass', ja: '利用券', zh: '通行证', th: 'พาส' },
    'checkout.cfRefundLabel': { ko: '환불 기준', en: 'Refund terms', ja: '返金基準', zh: '退款标准', th: 'เงื่อนไขคืนเงิน' },
    'checkout.cfRefundNote': { ko: '해지·환불은 내 라이브러리에서 신청할 수 있습니다.', en: 'Cancel or request a refund in your library.', ja: '解約・返金はマイライブラリから申請できます。', zh: '可在我的音乐库中取消或申请退款。', th: 'ยกเลิกหรือขอคืนเงินได้ในคลังของฉัน' },
    'checkout.cfGoLibrary': { ko: '내 라이브러리로 이동', en: 'Go to my library', ja: 'マイライブラリへ移動', zh: '前往我的音乐库', th: 'ไปที่คลังของฉัน' },
    'checkout.cfFailed': { ko: '결제 승인 실패', en: 'Payment confirmation failed', ja: '決済承認に失敗しました', zh: '支付确认失败', th: 'ยืนยันการชำระเงินล้มเหลว' },
    'checkout.cfFailBanner': { ko: '승인 실패', en: 'Confirmation failed', ja: '承認失敗', zh: '确认失败', th: 'ยืนยันล้มเหลว' },
    'checkout.cfBadResponse': { ko: 'Toss 결제 응답이 올바르지 않습니다. (paymentKey/orderId/amount 누락)', en: 'Invalid Toss payment response. (paymentKey/orderId/amount missing)', ja: 'Toss決済の応答が正しくありません。(paymentKey/orderId/amountの欠落)', zh: 'Toss 支付响应无效。（缺少 paymentKey/orderId/amount）', th: 'การตอบกลับชำระเงิน Toss ไม่ถูกต้อง (ขาด paymentKey/orderId/amount)' },
    'checkout.refundL1': { ko: '결제 후 7일 이내이고 이용 내역이 없으면 전액 환불됩니다(청약철회).', en: 'Full refund within 7 days of payment with no usage (withdrawal).', ja: '支払後7日以内で利用履歴がない場合は全額返金されます（クーリングオフ）。', zh: '支付后7日内且无使用记录可全额退款（撤回）。', th: 'คืนเงินเต็มจำนวนหากไม่มีการใช้งานภายใน 7 วันหลังชำระเงิน (เพิกถอน)' },
    'checkout.refundL2': { ko: '그 외에는 즉시 해지 시 이용한 일수만큼 일할 계산해 뺀 금액을 환불합니다. 위약금은 없습니다(중도해지).', en: 'Otherwise, immediate cancellation refunds the pro-rated remainder by days used. No penalty (mid-term).', ja: 'それ以外は即時解約時に利用日数分を日割りで差し引いた金額を返金します。違約金はありません（中途解約）。', zh: '其他情况立即取消时，按已使用天数比例扣除后退款，无违约金（中途取消）。', th: 'กรณีอื่น ยกเลิกทันทีจะคืนเงินส่วนที่เหลือตามสัดส่วนวันที่ใช้ ไม่มีค่าปรับ (ยกเลิกกลางคัน)' },
    'checkout.refundL3': { ko: '일반해지를 선택하면 만료일까지 이용하고 종료되며, 환불은 없습니다.', en: 'Standard cancellation ends service at expiry with no refund.', ja: '通常解約を選ぶと満了日まで利用して終了し、返金はありません。', zh: '选择普通取消可使用至到期后结束，不予退款。', th: 'ยกเลิกแบบทั่วไปจะใช้ได้จนหมดอายุโดยไม่มีการคืนเงิน' },
    'checkout.refundL4': { ko: '이용 내역은 30초 이상 재생 또는 다운로드 1회를 말합니다.', en: 'Usage means 30+ seconds of playback or 1 download.', ja: '利用履歴とは30秒以上の再生または1回のダウンロードを指します。', zh: '使用记录指播放30秒以上或下载1次。', th: 'การใช้งานหมายถึงการเล่น 30 วินาทีขึ้นไปหรือดาวน์โหลด 1 ครั้ง' },
    'checkout.refundL5': { ko: '이용권은 1개월 단기 패스로 자동 연장되지 않습니다. 해지·환불은 내 이용권에서 직접 신청할 수 있습니다.', en: 'Passes are 1-month short passes with no auto-renewal. Cancel or refund directly from My Pass.', ja: '利用券は1か月の短期パスで自動更新されません。解約・返金はマイ利用券から直接申請できます。', zh: '通行证为1个月短期有效，不自动续费。可在我的通行证中直接取消或退款。', th: 'พาสมีอายุ 1 เดือน ไม่ต่ออายุอัตโนมัติ ยกเลิกหรือขอคืนเงินได้โดยตรงจากพาสของฉัน' },
    'checkout.refundL6': { ko: '환불 금액은 결제 수단에 따라 3~5영업일 안에 반환됩니다.', en: 'Refunds return in 3-5 business days depending on method.', ja: '返金額は決済手段により3〜5営業日以内に返還されます。', zh: '退款金额将视支付方式在3至5个工作日内返还。', th: 'เงินคืนจะเข้าภายใน 3-5 วันทำการขึ้นอยู่กับช่องทางชำระเงิน' },
    'checkout.refundSummary': { ko: '7일 이내 미이용 시 전액 환불, 그 외 즉시 해지 시 이용일수 일할 차감 후 환불(위약금 없음), 일반해지 시 만료일까지 이용.', en: 'Full refund if unused within 7 days; otherwise pro-rated refund on immediate cancellation (no penalty); standard cancellation runs to expiry.', ja: '7日以内未利用なら全額返金、それ以外の即時解約は利用日数日割り差し引き後返金（違約金なし）、通常解約は満了日まで利用。', zh: '7日内未使用全额退款，其他情况立即取消按使用天数比例退款（无违约金），普通取消使用至到期。', th: 'คืนเต็มจำนวนหากไม่ใช้ภายใน 7 วัน กรณีอื่นยกเลิกทันทีคืนตามสัดส่วน (ไม่มีค่าปรับ) ยกเลิกทั่วไปใช้จนหมดอายุ' },
  },
  player: {
    'player.errPlayFailed': { ko: '음원을 재생할 수 없습니다. 네트워크 상태를 확인한 뒤 다시 시도해 주세요.', en: "Can't play this track. Check your connection and try again.", ja: '音源を再生できません。ネットワークを確認してもう一度お試しください。', zh: '无法播放该音乐，请检查网络后重试。', th: 'เล่นเพลงไม่ได้ ตรวจสอบเครือข่ายแล้วลองอีกครั้ง' },
    'player.errStreamLost': { ko: '스트리밍 연결이 끊겼습니다: {title}', en: 'Stream interrupted: {title}', ja: 'ストリーミングが切断されました：{title}', zh: '流媒体连接中断：{title}', th: 'สตรีมขาดการเชื่อมต่อ: {title}' },
    'player.errSkipLogin': { ko: '재생하려면 로그인이 필요합니다.', en: 'Log in to play.', ja: '再生にはログインが必要です。', zh: '播放需要登录。', th: 'เข้าสู่ระบบเพื่อเล่นเพลง' },
    'player.errSkipUnplayable': { ko: '재생할 수 없는 음원을 건너뜁니다: {title}', en: 'Skipping unplayable track: {title}', ja: '再生できない音源をスキップします：{title}', zh: '跳过无法播放的音乐：{title}', th: 'ข้ามเพลงที่เล่นไม่ได้: {title}' },
    'player.storeErrPlayFailed': { ko: '음원을 재생할 수 없습니다. 네트워크 상태를 확인해 주세요.', en: "Can't play this track. Check the network.", ja: '音源を再生できません。ネットワークを確認してください。', zh: '无法播放该音乐，请检查网络。', th: 'เล่นเพลงไม่ได้ กรุณาตรวจสอบเครือข่าย' },
  },
  studio: {
    'studio.upload.creditLyrics': { ko: '가사', en: 'Lyrics', ja: '歌詞', zh: '歌词', th: 'เนื้อเพลง' },
    'studio.upload.creditVocals': { ko: '보컬', en: 'Vocals', ja: 'ボーカル', zh: '人声', th: 'เสียงร้อง' },
    'studio.upload.creditInstrumental': { ko: '반주', en: 'Instrumental', ja: '伴奏', zh: '伴奏', th: 'ดนตรีประกอบ' },
    'studio.upload.contribLyrics': { ko: '작사', en: 'Lyrics writing', ja: '作詞', zh: '作词', th: 'แต่งเนื้อเพลง' },
    'studio.upload.contribMelody': { ko: '멜로디 입력', en: 'Melody input', ja: 'メロディー入力', zh: '旋律输入', th: 'ใส่ทำนอง' },
    'studio.upload.contribArrange': { ko: '편곡', en: 'Arrangement', ja: '編曲', zh: '编曲', th: 'เรียบเรียง' },
    'studio.upload.contribPerform': { ko: '실연', en: 'Performance', ja: '実演', zh: '表演', th: 'การแสดง' },
    'studio.upload.contribMix': { ko: '믹스·선별', en: 'Mix and curation', ja: 'ミックス・選別', zh: '混音与筛选', th: 'มิกซ์และคัดเลือก' },
    'studio.upload.blockBase': { ko: '선택하신 요금제 또는 생성 시점의 음원은 상업적 단독 음원 스트리밍 유통이 불가하여 업로드할 수 없습니다.', en: "The selected plan or generation-time track can't be distributed as commercial standalone streaming, so it can't be uploaded.", ja: '選択したプランまたは生成時点の音源は商用単独ストリーミング配信ができないため、アップロードできません。', zh: '所选方案或生成时的音乐无法作为商业独立流媒体发行，因此无法上传。', th: 'แผนที่เลือกหรือเพลง ณ เวลาสร้างไม่สามารถเผยแพร่แบบสตรีมมิงเดี่ยวเชิงพาณิชย์ได้ จึงอัปโหลดไม่ได้' },
    'studio.upload.blockBasePlan': { ko: '선택하신 요금제({plan}) 또는 생성 시점의 음원은 상업적 단독 음원 스트리밍 유통이 불가하여 업로드할 수 없습니다.', en: "The selected plan ({plan}) or generation-time track can't be distributed as commercial standalone streaming, so it can't be uploaded.", ja: '選択したプラン（{plan}）または生成時点の音源は商用単独ストリーミング配信ができないため、アップロードできません。', zh: '所选方案（{plan}）或生成时的音乐无法作为商业独立流媒体发行，因此无法上传。', th: 'แผนที่เลือก ({plan}) หรือเพลง ณ เวลาสร้างไม่สามารถเผยแพร่แบบสตรีมมิงเดี่ยวเชิงพาณิชย์ได้ จึงอัปโหลดไม่ได้' },
    'studio.upload.blockTier': { ko: '비상업/무료 요금제 차단: {code}', en: 'Non-commercial/free-tier block: {code}', ja: '非商用・無料プランのブロック：{code}', zh: '非商业/免费方案拦截：{code}', th: 'บล็อกแผนที่ไม่ใช่เชิงพาณิชย์/ฟรี: {code}' },
    'studio.upload.blockSyncOnly': { ko: '영상 배경음악 Sync 전용 라이선스 차단: {code}', en: 'Sync-only video-BGM license block: {code}', ja: '映像BGM Sync専用ライセンスのブロック：{code}', zh: '视频BGM Sync专用许可拦截：{code}', th: 'บล็อกสัญญาอนุญาต Sync-only สำหรับ BGM วิดีโอ: {code}' },
    'studio.upload.blockRetro': { ko: '유료 구독 시작일 이전에 생성된 음원 소급 적용 불가: {code}', en: 'Generation predates the paid subscription (no retroactive cover): {code}', ja: '有料購読開始日より前に生成された音源は遡及適用不可：{code}', zh: '生成早于付费订阅开始日，无法追溯适用：{code}', th: 'เพลงสร้างก่อนเริ่มสมัครแบบชำระเงิน (ไม่มีผลย้อนหลัง): {code}' },
    'studio.upload.blockExpired': { ko: '유료 구독 만료 이후 생성된 음원: {code}', en: 'Generated after the paid subscription expired: {code}', ja: '有料購読の満了後に生成された音源：{code}', zh: '生成于付费订阅到期之后：{code}', th: 'เพลงสร้างหลังหมดอายุสมัครแบบชำระเงิน: {code}' },
    'studio.upload.needFile': { ko: '음원 파일을 먼저 선택해 주세요.', en: 'Choose an audio file first.', ja: '先に音源ファイルを選択してください。', zh: '请先选择音频文件。', th: 'กรุณาเลือกไฟล์เพลงก่อน' },
    'studio.upload.stagedDone': { ko: '스테이징 업로드 완료. 파일 무결성/형식 검사가 finalize 단계에서 진행됩니다.', en: 'Staging upload done. Integrity/format checks run at finalize.', ja: 'ステージングアップロード完了。整合性・形式チェックはfinalize段階で行われます。', zh: '暂存上传完成，完整性/格式检查将在 finalize 阶段进行。', th: 'อัปโหลด staging เสร็จ การตรวจความสมบูรณ์/รูปแบบจะทำในขั้น finalize' },
    'studio.upload.discFull': { ko: 'FULL_AI: 본 음원은 AI가 전곡 생성한 음원입니다.', en: 'FULL_AI: this track was fully generated by AI.', ja: 'FULL_AI：本音源はAIが全曲生成した音源です。', zh: 'FULL_AI：本音乐由 AI 全曲生成。', th: 'FULL_AI: เพลงนี้ AI สร้างทั้งเพลง' },
    'studio.upload.discAssist': { ko: 'AI_ASSISTED_HUMAN: 본 음원은 인간 창작자가 AI 보조 도구를 사용해 제작한 음원입니다.', en: 'AI_ASSISTED_HUMAN: made by a human creator with AI assist tools.', ja: 'AI_ASSISTED_HUMAN：本音源は人間の制作者がAI補助ツールを使って制作した音源です。', zh: 'AI_ASSISTED_HUMAN：本音乐由人类创作者使用 AI 辅助工具制作。', th: 'AI_ASSISTED_HUMAN: เพลงนี้มนุษย์สร้างโดยใช้เครื่องมือช่วย AI' },
    'studio.status.REJECTED': { ko: '반려됨', en: 'Rejected', ja: '差し戻し', zh: '已驳回', th: 'ถูกปฏิเสธ' },
    'studio.status.PROCESSING': { ko: '처리 중', en: 'Processing', ja: '処理中', zh: '处理中', th: 'กำลังประมวลผล' },
    'studio.genType.FULL_AI': { ko: '순수 AI 생성', en: 'Pure AI', ja: 'ピュアAI生成', zh: '纯 AI 生成', th: 'AI ล้วน' },
    'studio.genType.AI_ASSISTED_HUMAN': { ko: 'AI 어시스트', en: 'AI-assisted', ja: 'AIアシスト', zh: 'AI 辅助', th: 'AI ช่วยสร้าง' },
  },
  auth: {
    'auth.ageGate.title': { ko: '만 14세 이상 확인', en: "Confirm you're 14+", ja: '14歳以上の確認', zh: '年满14岁确认', th: 'ยืนยันอายุ 14 ปีขึ้นไป' },
    'auth.ageGate.desc': { ko: '소셜 계정으로 처음 로그인했습니다. AVIBE는 만 14세 이상만 이용할 수 있습니다. 스트리밍·결제·업로드를 이용하려면 연령 확인이 필요합니다.', en: 'You signed in with a social account for the first time. AVIBE is 14+ only. Age confirmation is required for streaming, payments, and uploads.', ja: 'ソーシャルアカウントで初めてログインしました。AVIBEは14歳以上のみ利用できます。ストリーミング・決済・アップロードには年齢確認が必要です。', zh: '您首次使用社交账号登录。AVIBE 仅限14岁以上使用。使用流媒体、支付和上传需要年龄确认。', th: 'คุณเข้าสู่ระบบด้วยบัญชีโซเชียลครั้งแรก AVIBE ใช้ได้เฉพาะอายุ 14 ปีขึ้นไป ต้องยืนยันอายุก่อนใช้สตรีมมิง การชำระเงิน และการอัปโหลด' },
    'auth.ageGate.confirm': { ko: '만 14세 이상입니다. (필수)', en: 'I am 14 or older. (Required)', ja: '14歳以上です。（必須）', zh: '我已年满14岁。（必填）', th: 'ฉันอายุ 14 ปีขึ้นไป (บังคับ)' },
    'auth.ageGate.start': { ko: '확인하고 시작하기', en: 'Confirm and start', ja: '確認して開始', zh: '确认并开始', th: 'ยืนยันและเริ่มต้น' },
    'auth.ageGate.done': { ko: '연령 확인이 완료되었습니다.', en: 'Age confirmed.', ja: '年齢確認が完了しました。', zh: '年龄确认完成。', th: 'ยืนยันอายุแล้ว' },
    'auth.social.sessionErr': { ko: '소셜 로그인 세션을 확인할 수 없습니다.', en: "Couldn't verify the social login session.", ja: 'ソーシャルログインのセッションを確認できません。', zh: '无法验证社交登录会话。', th: 'ยืนยันเซสชันโซเชียลไม่ได้' },
  },
  legal: {
    'legal.consentGate.title': { ko: '필수 약관 동의', en: 'Required terms consent', ja: '必須規約への同意', zh: '必填条款同意', th: 'ยินยอมข้อกำหนดบังคับ' },
    'legal.consentGate.desc': { ko: '개정된 필수 약관이 있습니다. 스트리밍·결제·업로드를 계속 이용하려면 아래 약관에 동의해 주세요.', en: 'Required terms were updated. Agree below to keep using streaming, payments, and uploads.', ja: '必須規約が改定されました。ストリーミング・決済・アップロードを続けるには以下に同意してください。', zh: '必填条款已更新，请同意以下条款以继续使用流媒体、支付和上传。', th: 'ข้อกำหนดบังคับมีการปรับปรุง กรุณายินยอมด้านล่างเพื่อใช้สตรีมมิง การชำระเงิน และการอัปโหลดต่อ' },
    'legal.consentGate.viewDoc': { ko: '전문 보기', en: 'Read full text', ja: '全文を見る', zh: '查看全文', th: 'อ่านฉบับเต็ม' },
    'legal.consentGate.consent': { ko: '동의하기', en: 'Agree', ja: '同意する', zh: '同意', th: 'ยินยอม' },
    'legal.consentGate.done': { ko: '약관 동의가 완료되었습니다.', en: 'Terms consent recorded.', ja: '規約への同意が完了しました。', zh: '条款同意完成。', th: 'บันทึกความยินยอมแล้ว' },
    'legal.aiBadge.aiTitle': { ko: 'AI 생성 음원', en: 'AI-generated track', ja: 'AI生成音源', zh: 'AI 生成音乐', th: 'เพลงที่ AI สร้าง' },
    'legal.aiBadge.certTitle': { ko: '상업 라이선스 검증 완료 (Gate1-3 통과 + 심사 승인)', en: 'Commercial license verified (Gates 1-3 + review passed)', ja: '商用ライセンス検証完了（Gate1-3通過＋審査承認）', zh: '商业许可验证完成（Gate1-3 通过＋审核批准）', th: 'ตรวจสอบสัญญาอนุญาตเชิงพาณิชย์แล้ว (ผ่าน Gate1-3 + อนุมัติ)' },
    'legal.aiBadge.certLabel': { ko: '상업적 이용권 보유 인증', en: 'Certified commercial-use license', ja: '商用利用権保有認証', zh: '商业使用权认证', th: 'รับรองสิทธิ์ใช้เชิงพาณิชย์' },
    'legal.precheck.checking': { ko: '라이선스 사전검사 진행 중', en: 'License pre-check running', ja: 'ライセンス事前検査進行中', zh: '许可预检进行中', th: 'กำลังตรวจสัญญาอนุญาตเบื้องต้น' },
    'legal.precheck.pass': { ko: 'Gate1-2 통과', en: 'Gates 1-2 passed', ja: 'Gate1-2通過', zh: 'Gate1-2 通过', th: 'ผ่าน Gate1-2' },
    'legal.precheck.fail': { ko: '업로드 차단', en: 'Upload blocked', ja: 'アップロードブロック', zh: '上传被拦截', th: 'อัปโหลดถูกบล็อก' },
    'legal.precheck.checkingDesc': { ko: '상업 라이선스 적격성을 확인하고 있습니다.', en: 'Checking commercial license eligibility.', ja: '商用ライセンスの適格性を確認しています。', zh: '正在检查商业许可资格。', th: 'กำลังตรวจคุณสมบัติสัญญาอนุญาตเชิงพาณิชย์' },
    'legal.precheck.passDesc': { ko: '상업용 구독·생성일·제공자 정책 검사를 통과했습니다. 다음 단계로 진행하세요.', en: 'Paid subscription, generation date, and provider policy checks passed. Continue.', ja: '有料購読・生成日・プロバイダー方針の検査に合格しました。次に進んでください。', zh: '付费订阅、生成日期和提供方政策检查已通过，请继续。', th: 'ผ่านการตรวจสมัครแบบชำระเงิน วันที่สร้าง และนโยบายผู้ให้บริการแล้ว ดำเนินการต่อ' },
    'legal.precheck.failDesc': { ko: '검사에 실패했습니다.', en: 'The check failed.', ja: '検査に失敗しました。', zh: '检查失败。', th: 'การตรวจล้มเหลว' },
    'legal.cert.desc': { ko: '인증서 번호로 진위를 확인합니다. 인쇄 버튼으로 PDF 보관이 가능합니다.', en: 'Verify authenticity by certificate number. Use print to save a PDF.', ja: '証書番号で真偽を確認します。印刷ボタンでPDF保存が可能です。', zh: '可凭证书编号验证真伪，可使用打印按钮保存 PDF。', th: 'ตรวจสอบความถูกต้องด้วยหมายเลขใบรับรอง ใช้ปุ่มพิมพ์เพื่อบันทึก PDF' },
    'legal.cert.print': { ko: '인쇄 / PDF 저장', en: 'Print / Save PDF', ja: '印刷 / PDF保存', zh: '打印 / 保存 PDF', th: 'พิมพ์ / บันทึก PDF' },
    'legal.cert.loadFail': { ko: '인증서를 확인하지 못했습니다', en: "Couldn't verify the certificate", ja: '証書を確認できませんでした', zh: '无法验证证书', th: 'ตรวจสอบใบรับรองไม่ได้' },
    'legal.cert.verified': { ko: '진위 확인됨', en: 'Verified', ja: '真偽確認済み', zh: '已验证真伪', th: 'ตรวจสอบแล้ว' },
    'legal.cert.fTitle': { ko: '곡명', en: 'Track', ja: '曲名', zh: '歌曲', th: 'เพลง' },
    'legal.cert.fProvider': { ko: 'AI 플랫폼', en: 'AI platform', ja: 'AIプラットフォーム', zh: 'AI 平台', th: 'แพลตฟอร์ม AI' },
    'legal.cert.fTier': { ko: '사용 요금제', en: 'Plan used', ja: '使用プラン', zh: '使用的方案', th: 'แผนที่ใช้' },
    'legal.cert.fGenerated': { ko: '생성일', en: 'Generated', ja: '生成日', zh: '生成日期', th: 'วันที่สร้าง' },
    'legal.cert.fGates': { ko: '게이트 통과', en: 'Gates passed', ja: 'ゲート通過', zh: '已通过网关', th: 'ผ่านเกณฑ์' },
    'legal.cert.gatesPass': { ko: 'Gate 1-3 통과', en: 'Gates 1-3 passed', ja: 'Gate 1-3通過', zh: 'Gate 1-3 通过', th: 'ผ่าน Gate 1-3' },
    'legal.cert.gatesPartial': { ko: '일부 미통과', en: 'Partially passed', ja: '一部未通過', zh: '部分未通过', th: 'ผ่านบางส่วน' },
    'legal.cert.fIssued': { ko: '발급일', en: 'Issued', ja: '発行日', zh: '签发日期', th: 'วันที่ออก' },
    'legal.cert.na': { ko: '미기재', en: 'Not stated', ja: '未記載', zh: '未填写', th: 'ไม่ระบุ' },
    'legal.cert.notice': { ko: '본 인증은 생성 시점의 유료 상업 플랜 보유, 당시 정책상 상업 이용 허용, 중복·재등록 아님을 파트너 신고와 자동 검사에 근거해 확인한 절차이며, 권리 관계를 보증하지 않습니다.', en: 'This certification confirms paid commercial-plan holding at generation time, policy-allowed commercial use, and non-duplication based on partner filings and automated checks; it does not warrant rights ownership.', ja: '本認証は生成時点の有料商用プラン保有、当時の方針上の商用利用許可、重複・再登録でないことをパートナー申告と自動検査に基づき確認した手続きであり、権利関係を保証しません。', zh: '本认证基于合作方申报与自动检查，确认生成时持有付费商业方案、当时政策允许商业使用且非重复/重新注册，不保证权利关系。', th: 'ใบรับรองนี้ยืนยันการถือแผนเชิงพาณิชย์แบบชำระเงิน ณ เวลาสร้าง การอนุญาตใช้เชิงพาณิชย์ตามนโยบายขณะนั้น และไม่ซ้ำ/ลงทะเบียนซ้ำ จากคำชี้แจงพาร์ตเนอร์และการตรวจอัตโนมัติ ไม่รับประกันความสัมพันธ์ด้านสิทธิ์' },
    'legal.cert.errNoNumber': { ko: '인증서 번호가 없습니다.', en: 'No certificate number.', ja: '証書番号がありません。', zh: '缺少证书编号。', th: 'ไม่มีหมายเลขใบรับรอง' },
    'legal.cert.errNotFound': { ko: '인증서를 찾을 수 없습니다. 번호를 확인해 주세요.', en: 'Certificate not found. Check the number.', ja: '証書が見つかりません。番号を確認してください。', zh: '未找到证书，请确认编号。', th: 'ไม่พบใบรับรอง กรุณาตรวจสอบหมายเลข' },
  },
  playlist: {
    'playlist.picker.title': { ko: '플레이리스트에 담기', en: 'Add to playlist', ja: 'プレイリストに入れる', zh: '添加到播放列表', th: 'เพิ่มลงเพลย์ลิสต์' },
    'playlist.picker.descWithTrack': { ko: '「{title}」을(를) 보관할 플레이리스트를 선택하세요.', en: 'Choose a playlist to save "{title}" to.', ja: '「{title}」を保存するプレイリストを選んでください。', zh: '请选择要保存「{title}」的播放列表。', th: 'เลือกเพลย์ลิสต์ที่จะบันทึก "{title}"' },
    'playlist.picker.desc': { ko: '보관할 플레이리스트를 선택하세요.', en: 'Choose a playlist to save to.', ja: '保存するプレイリストを選んでください。', zh: '请选择要保存到的播放列表。', th: 'เลือกเพลย์ลิสต์ที่จะบันทึก' },
    'playlist.picker.empty': { ko: '생성된 플레이리스트가 없습니다.', en: 'No playlists yet.', ja: 'プレイリストがありません。', zh: '还没有播放列表。', th: 'ยังไม่มีเพลย์ลิสต์' },
    'playlist.picker.createNew': { ko: '새 플레이리스트 만들기', en: 'New playlist', ja: '新しいプレイリスト', zh: '新建播放列表', th: 'เพลย์ลิสต์ใหม่' },
    'playlist.picker.public': { ko: '공개', en: 'Public', ja: '公開', zh: '公开', th: 'สาธารณะ' },
    'playlist.picker.private': { ko: '비공개', en: 'Private', ja: '非公開', zh: '私密', th: 'ส่วนตัว' },
    'playlist.picker.add': { ko: '담기 →', en: 'Add →', ja: '入れる →', zh: '添加 →', th: 'เพิ่ม →' },
    'playlist.picker.createMore': { ko: '+ 새로 만들기', en: '+ New', ja: '+ 新規作成', zh: '+ 新建', th: '+ สร้างใหม่' },
    'playlist.picker.namePh': { ko: '새 플레이리스트 제목', en: 'New playlist title', ja: '新しいプレイリスト名', zh: '新播放列表标题', th: 'ชื่อเพลย์ลิสต์ใหม่' },
    'playlist.picker.create': { ko: '생성', en: 'Create', ja: '作成', zh: '创建', th: 'สร้าง' },
  },
};

// Pseudo-localize en (prior-WI style): accented vowels, {placeholders} intact,
// wrapped in ⟦⟧ with ~40% ～ padding.
const ACCENT = { a: 'à', e: 'è', i: 'ì', o: 'ò', u: 'ù', A: 'À', E: 'È', I: 'Ì', O: 'Ò', U: 'Ù' };
function pseudo(s) {
  const kept = [];
  const masked = s.replace(/\{[^}]*\}/g, (m) => ` ${kept.push(m) - 1}`);
  const acc = masked.replace(/[aeiouAEIOU]/g, (c) => ACCENT[c]);
  const pad = '～'.repeat(Math.max(1, Math.round(s.length * 0.4)));
  const out = `⟦${acc}${pad}⟧`;
  return out.replace(/ (\d+)/g, (_, i) => kept[Number(i)]);
}

function deepSet(obj, dotted, value) {
  const parts = dotted.split('.');
  let cur = obj;
  for (let i = 0; i < parts.length - 1; i++) {
    cur[parts[i]] = cur[parts[i]] ?? {};
    cur = cur[parts[i]];
  }
  cur[parts[parts.length - 1]] = value;
}

function deepGet(obj, dotted) {
  let cur = obj;
  for (const p of dotted.split('.')) {
    if (cur == null || typeof cur !== 'object' || !(p in cur)) return undefined;
    cur = cur[p];
  }
  return cur;
}

const CHECK = process.argv.includes('--check');
let added = 0;
for (const [file, entries] of Object.entries(KEYS)) {
  for (const loc of LOCALES) {
    const p = path.join(DIR, loc, `${file}.json`);
    const json = fs.existsSync(p) ? JSON.parse(fs.readFileSync(p, 'utf8')) : {};
    for (const [dotted, vals] of Object.entries(entries)) {
      const existing = deepGet(json, dotted);
      const want = loc === 'en-XA' ? pseudo(vals.en) : vals[loc === 'zh-CN' ? 'zh' : loc];
      if (existing !== undefined) {
        if (existing !== want) {
          console.error(`CONFLICT ${loc}/${file}.json ${dotted}: existing differs`);
          process.exitCode = 1;
        }
        continue;
      }
      if (!CHECK) deepSet(json, dotted, want);
      added++;
    }
    if (!CHECK) fs.writeFileSync(p, JSON.stringify(json, null, 2) + '\n');
  }
}
console.log(CHECK ? `check: ${added} keys missing` : `merged: ${added} locale entries added`);
if (CHECK && added > 0) process.exitCode = 1;
