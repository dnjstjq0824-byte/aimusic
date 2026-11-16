// R13 check: ko refund strings must equal the pre-R13 constants exactly.
const fs = require('node:fs');
const ko = JSON.parse(fs.readFileSync('web/i18n/locales/ko/checkout.json', 'utf8')).checkout;
const want = [
  '결제 후 7일 이내이고 이용 내역이 없으면 전액 환불됩니다(청약철회).',
  '그 외에는 즉시 해지 시 이용한 일수만큼 일할 계산해 뺀 금액을 환불합니다. 위약금은 없습니다(중도해지).',
  '일반해지를 선택하면 만료일까지 이용하고 종료되며, 환불은 없습니다.',
  '이용 내역은 30초 이상 재생 또는 다운로드 1회를 말합니다.',
  '이용권은 1개월 단기 패스로 자동 연장되지 않습니다. 해지·환불은 내 이용권에서 직접 신청할 수 있습니다.',
  '환불 금액은 결제 수단에 따라 3~5영업일 안에 반환됩니다.',
];
const wantSummary = '7일 이내 미이용 시 전액 환불, 그 외 즉시 해지 시 이용일수 일할 차감 후 환불(위약금 없음), 일반해지 시 만료일까지 이용.';
let fail = 0;
want.forEach((w, i) => {
  const k = `refundL${i + 1}`;
  if (ko[k] !== w) { console.log(`DIFF ${k}:\n  got:  ${ko[k]}\n  want: ${w}`); fail++; }
});
if (ko.refundSummary !== wantSummary) { console.log(`DIFF refundSummary:\n  got:  ${ko.refundSummary}\n  want: ${wantSummary}`); fail++; }
console.log(fail === 0 ? 'REFUND-KO-IDENTICAL (7/7)' : `REFUND-KO-DIFFERS (${fail})`);
process.exit(fail === 0 ? 0 : 1);
