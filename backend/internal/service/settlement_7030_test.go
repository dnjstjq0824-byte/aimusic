package service

import (
	"testing"

	"avibe-platform/backend/internal/domain"
)

// Canon: 방향결정서 §5-1 — 유통 순매출 70:30 (파트너 70). 직접 판매·
// 라이선스는 해당 음원(파트너)에 직접 귀속, 매장 구독료 풀은 30초 이상
// 매장 재생 시간 비례 배분. Integer math floors at each step.

func TestSplitSettlement7030(t *testing.T) {
	cases := []struct {
		name                                    string
		pool, direct, psecs, tsecs, withholdNum int64
		wantShare, wantAttributed, wantGross    int64
		wantWithholding, wantNet                int64
	}{
		{
			name: "pool share plus direct, BUSINESS_33",
			pool: 100000, direct: 10000, psecs: 30, tsecs: 100, withholdNum: 33,
			wantShare: 30000, wantAttributed: 40000, wantGross: 28000,
			wantWithholding: 924, wantNet: 27076,
		},
		{
			name: "floor at each step",
			pool: 100, direct: 0, psecs: 1, tsecs: 3, withholdNum: 33,
			wantShare: 33, wantAttributed: 33, wantGross: 23,
			wantWithholding: 0, wantNet: 23,
		},
		{
			name: "zero denominator pays direct only",
			pool: 100000, direct: 50000, psecs: 0, tsecs: 0, withholdNum: 33,
			wantShare: 0, wantAttributed: 50000, wantGross: 35000,
			wantWithholding: 1155, wantNet: 33845,
		},
		{
			name: "zero inputs pay zero",
			pool: 0, direct: 0, psecs: 0, tsecs: 0, withholdNum: 33,
			wantShare: 0, wantAttributed: 0, wantGross: 0,
			wantWithholding: 0, wantNet: 0,
		},
		{
			name: "OTHER_88 withholding",
			pool: 0, direct: 100000, psecs: 0, tsecs: 0, withholdNum: 88,
			wantShare: 0, wantAttributed: 100000, wantGross: 70000,
			wantWithholding: 6160, wantNet: 63840,
		},
		{
			name: "share never exceeds the pool",
			pool: 1000, direct: 0, psecs: 120, tsecs: 100, withholdNum: 33,
			wantShare: 1000, wantAttributed: 1000, wantGross: 700,
			wantWithholding: 23, wantNet: 677,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			share, attributed, gross, withholding, net := splitSettlement7030(
				tc.pool, tc.direct, tc.psecs, tc.tsecs, tc.withholdNum)
			if share != tc.wantShare || attributed != tc.wantAttributed ||
				gross != tc.wantGross || withholding != tc.wantWithholding || net != tc.wantNet {
				t.Fatalf("got share=%d attributed=%d gross=%d withholding=%d net=%d, want %d/%d/%d/%d/%d",
					share, attributed, gross, withholding, net,
					tc.wantShare, tc.wantAttributed, tc.wantGross, tc.wantWithholding, tc.wantNet)
			}
		})
	}
}

func TestSettlementInputValidation(t *testing.T) {
	if _, err := (SettlementInput{StorePoolKRW: -1}).withholdNum(); err == nil {
		t.Fatalf("negative pool must be rejected")
	}
	if _, err := (SettlementInput{DirectKRW: -1}).withholdNum(); err == nil {
		t.Fatalf("negative direct must be rejected")
	}
	if _, err := (SettlementInput{IncomeType: "BOGUS"}).withholdNum(); err == nil {
		t.Fatalf("unknown income type must be rejected")
	}
	num, err := (SettlementInput{}).withholdNum()
	if err != nil || num != 33 {
		t.Fatalf("empty income type must default to BUSINESS_33, got num=%d err=%v", num, err)
	}
	num, err = (SettlementInput{IncomeType: domain.IncomeOther88}).withholdNum()
	if err != nil || num != 88 {
		t.Fatalf("OTHER_88 must map to 88, got num=%d err=%v", num, err)
	}
}
