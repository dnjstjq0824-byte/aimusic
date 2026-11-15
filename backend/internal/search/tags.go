// Package search holds the canonical tag registry (R5): English keys with
// display names in 5 locales. The engine stores and responds keys; the UI
// translates via web/i18n tags.json (same key set — keep in sync).
package search

import "strings"

// Facets.
const (
	FacetMood    = "mood"
	FacetGenre   = "genre"
	FacetUseCase = "usecase"
)

// Tag is one canonical key with per-locale display names (ko, en, ja, zh-cn, th).
type Tag struct {
	Key    string
	Facets []string
	Names  map[string]string
}

// Registry is the full canonical set (19 keys).
var Registry = []Tag{
	{Key: "focus", Facets: []string{FacetMood}, Names: map[string]string{"ko": "집중", "en": "Focus", "ja": "集中", "zh-cn": "专注", "th": "โฟกัส"}},
	{Key: "chill", Facets: []string{FacetMood}, Names: map[string]string{"ko": "휴식", "en": "Chill", "ja": "チル", "zh-cn": "放松", "th": "ชิลล์"}},
	{Key: "cafe", Facets: []string{FacetMood, FacetUseCase}, Names: map[string]string{"ko": "카페", "en": "Cafe", "ja": "カフェ", "zh-cn": "咖啡馆", "th": "คาเฟ่"}},
	{Key: "workout", Facets: []string{FacetMood}, Names: map[string]string{"ko": "운동", "en": "Workout", "ja": "ワークアウト", "zh-cn": "运动", "th": "ออกกำลังกาย"}},
	{Key: "drive", Facets: []string{FacetMood}, Names: map[string]string{"ko": "드라이브", "en": "Drive", "ja": "ドライブ", "zh-cn": "驾驶", "th": "ขับรถ"}},
	{Key: "sleep", Facets: []string{FacetMood}, Names: map[string]string{"ko": "수면", "en": "Sleep", "ja": "スリープ", "zh-cn": "睡眠", "th": "นอน"}},
	{Key: "cinematic", Facets: []string{FacetMood}, Names: map[string]string{"ko": "시네마틱", "en": "Cinematic", "ja": "シネマティック", "zh-cn": "电影感", "th": "ซีเนมาติก"}},
	{Key: "cyberpunk", Facets: []string{FacetMood}, Names: map[string]string{"ko": "사이버펑크", "en": "Cyberpunk", "ja": "サイバーパンク", "zh-cn": "赛博朋克", "th": "ไซเบอร์พังก์"}},
	{Key: "electronic", Facets: []string{FacetGenre}, Names: map[string]string{"ko": "일렉트로닉", "en": "Electronic", "ja": "エレクトロニック", "zh-cn": "电子", "th": "อิเล็กทรอนิกส์"}},
	{Key: "pop", Facets: []string{FacetGenre}, Names: map[string]string{"ko": "팝", "en": "Pop", "ja": "ポップ", "zh-cn": "流行", "th": "ป๊อป"}},
	{Key: "rock", Facets: []string{FacetGenre}, Names: map[string]string{"ko": "록", "en": "Rock", "ja": "ロック", "zh-cn": "摇滚", "th": "ร็อก"}},
	{Key: "hiphop", Facets: []string{FacetGenre}, Names: map[string]string{"ko": "힙합", "en": "Hip-hop", "ja": "ヒップホップ", "zh-cn": "嘻哈", "th": "ฮิปฮอป"}},
	{Key: "jazz", Facets: []string{FacetGenre}, Names: map[string]string{"ko": "재즈", "en": "Jazz", "ja": "ジャズ", "zh-cn": "爵士", "th": "แจ๊ส"}},
	{Key: "classical", Facets: []string{FacetGenre}, Names: map[string]string{"ko": "클래식", "en": "Classical", "ja": "クラシック", "zh-cn": "古典", "th": "คลาสสิก"}},
	{Key: "ambient", Facets: []string{FacetGenre}, Names: map[string]string{"ko": "앰비언트", "en": "Ambient", "ja": "アンビエント", "zh-cn": "氛围", "th": "แอมเบียนต์"}},
	{Key: "gym", Facets: []string{FacetUseCase}, Names: map[string]string{"ko": "헬스장", "en": "Gym", "ja": "ジム", "zh-cn": "健身房", "th": "ยิม"}},
	{Key: "work", Facets: []string{FacetUseCase}, Names: map[string]string{"ko": "작업", "en": "Work", "ja": "作業", "zh-cn": "工作", "th": "ทำงาน"}},
	{Key: "youtube", Facets: []string{FacetUseCase}, Names: map[string]string{"ko": "유튜브", "en": "YouTube", "ja": "ユーチューブ", "zh-cn": "YouTube", "th": "ยูทูบ"}},
	{Key: "party", Facets: []string{FacetUseCase}, Names: map[string]string{"ko": "파티", "en": "Party", "ja": "パーティー", "zh-cn": "派对", "th": "ปาร์ตี้"}},
}

var byKey map[string]Tag
var byName map[string][]string // normalized display name (or key) -> keys

func init() {
	byKey = make(map[string]Tag, len(Registry))
	byName = map[string][]string{}
	for _, t := range Registry {
		byKey[t.Key] = t
		seen := map[string]bool{}
		add := func(s string) {
			n := strings.ToLower(strings.TrimSpace(s))
			if n == "" || seen[n] {
				return
			}
			seen[n] = true
			byName[n] = append(byName[n], t.Key)
		}
		add(t.Key)
		for _, name := range t.Names {
			add(name)
		}
	}
}

// IsKey reports whether key is a canonical tag key (case-insensitive).
func IsKey(key string) bool {
	_, ok := byKey[strings.ToLower(strings.TrimSpace(key))]
	return ok
}

// KeysForQuery expands a raw search string to canonical keys when it exactly
// matches a key or any locale display name (case-insensitive). Empty when the
// query is not tag-shaped; title/creator matching handles the rest.
func KeysForQuery(q string) []string {
	return byName[strings.ToLower(strings.TrimSpace(q))]
}

// FacetsFor returns the facets of a canonical key (nil when unknown).
func FacetsFor(key string) []string {
	if t, ok := byKey[strings.ToLower(strings.TrimSpace(key))]; ok {
		return t.Facets
	}
	return nil
}
