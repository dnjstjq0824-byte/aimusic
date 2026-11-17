# AVIBE SSR · SEO · 검색/AI 엔진 노출 설계 (2026-11-14)

| 항목 | 내용 |
|---|---|
| 발행 | CEO/PO |
| 목표 | 공개 페이지(트랙·플레이리스트·크리에이터·무드/용도 랜딩·가이드)를 **Google·Naver·Bing·Yahoo! JAPAN·AI 답변 엔진(ChatGPT 검색·Perplexity·Google AI 개요 등)·SNS 미리보기·임베드**에 최대한 많이, 정확하게 노출 |
| 현재 | Vite SPA(`web/`) → 크롤러가 받는 HTML은 빈 껍데기. `<html lang="ko">` 고정, 언어별 URL·sitemap·구조화 데이터·OG 이미지 없음 |
| 관련 | 화면 구조는 `AVIBE_UI구조_와이어프레임_20261114.md`, 작업 항목은 작업지시서 WI-S*·WI-P1 |

---

## 1. 채택 기술 (추가·교체)

| 영역 | 채택 | 이유 / 비고 |
|---|---|---|
| 웹 프레임워크 | **Nuxt 4.x** (Vue 3 + TypeScript) — Vite SPA 교체 | 기존 Vue SFC·Pinia·Tailwind를 그대로 살리면서 라우트별 SSR/SWR/CSR 혼합 렌더링. **Nuxt 3은 OSS 지원이 2026-07-31에 끝났으므로 사용 금지.** 정확한 버전은 WI-P1 착수 시 최신 안정판으로 고정(`^` 금지) |
| 서버 런타임 | **Nitro**, 배포 대상 Cloudflare Workers(프리셋 `cloudflare_module` — 착수 시 Nitro 문서로 철자 확인) | 전 세계 엣지에서 SSR → 일본·태국 사용자 응답 지연 최소 |
| 엣지 구성 | 기존 `worker.js`(API·HLS 프록시·재생 권한)는 **그대로 앞단 유지**, 페이지 요청만 Service Binding으로 Nuxt Worker에 전달 | 검증된 재생 권한 코드(`scripts/worker-grant.test.mjs`)를 건드리지 않음 |
| 다국어 라우팅 | **@nuxtjs/i18n** (vue-i18n 내장), 전략 `prefix_except_default`, 기본 ko | `/` = 한국어, `/en/…`, `/ja/…`, `/zh-cn/…`, `/th/…`. hreflang·canonical 자동 |
| SEO 모듈 | **Nuxt SEO 모듈군**: `@nuxtjs/sitemap`, `@nuxtjs/robots`, `nuxt-schema-org`, `nuxt-og-image`, `nuxt-site-config` | sitemap·robots·JSON-LD·동적 OG 이미지 표준화 |
| 이미지 | `@nuxt/image` + Cloudflare 이미지 변환(R2 원본) | 커버 160/320/640/1200, AVIF/WebP 자동 |
| 상태 | `@pinia/nuxt` | 기존 스토어 이전 |
| 측정 | Lighthouse CI(`@lhci/cli`) in GitHub Actions, `web-vitals` RUM → 자체 수집 엔드포인트, Cloudflare Web Analytics(쿠키 없음) | 성능 예산 회귀 차단 |
| 엔진(Go) | SEO 데이터 API, slug/handle, IndexNow 발송, oEmbed | 새 언어 도입 없음 |
| 언어 총괄 | 웹 TypeScript · 엔진 Go · 생성 Python · 모바일 Dart | **새 언어 추가 없음** — 프레임워크만 교체 |

### 1.1 비용·결정 (PO 확인 필요)
- Workers **무료 플랜은 요청당 CPU 10ms** 한도라 Nuxt SSR이 넘을 수 있다. **Workers Paid(월 $5~)** 가 권장안이다.
- 대안: VPS(Docker)에서 Nuxt를 Node 서버로 띄우고 Cloudflare 캐시를 앞에 둔다(추가 비용 0). 단, 현재 VPS는 API·워커·DB·FFmpeg를 함께 돌리므로 부하와 해외 지연이 불리하다.
- **CEO 권장: Workers Paid.** PO 승인 전까지 WI-P1은 두 대상 모두에 배포 가능한 구성(프리셋만 교체)으로 만든다.

---

## 2. 렌더링 전략 (Nitro `routeRules`)

| 경로 | 렌더 | 캐시 | 색인 |
|---|---|---|---|
| `/`, `/explore/**` 랜딩, `/charts`, `/playlists`, `/business`, `/for-creators`, `/trust`, `/pricing`, `/guides/**`, `/create` | SSR | SWR 10분 | O |
| `/tracks/**`, `/playlists/:slug`(공개), `/creators/**` | SSR | SWR 1시간 + 변경 시 엔진이 퍼지 | O |
| `/certificate/**` | SSR | SWR 1시간 | O (인증서는 검증 링크 신뢰 신호) |
| `/legal/**` | SSR | 24시간 | O |
| `/explore?q=…`(검색 결과) | SSR | 캐시 없음 | **noindex, follow** |
| `/login`, `/signup/**`, 비밀번호·인증 | CSR | – | noindex |
| `/library`, `/me/**`, `/account/**`, `/rights-center`, `/checkout/**`, `/studio/**`, `/business/console/**`, `/admin/**`, `/store/player` | CSR(`ssr:false`) | – | noindex + robots Disallow |
| `/embed/**` | SSR 경량 | SWR 1시간 | noindex (크롤은 허용 — oEmbed 대상) |

- SSR 페이지는 **로그인 여부와 무관하게 같은 HTML**(개인화는 하이드레이션 후 클라이언트에서). 캐시 오염 방지.
- `localStorage`·`window`·`hls.js` 접근은 `onMounted`/`.client` 컴포넌트로 격리. hls.js는 첫 재생 시 지연 로드.

---

## 3. URL·다국어 규칙

| 항목 | 규칙 |
|---|---|
| 트랙 | `/tracks/{slug}-{shortId}` — slug는 원제목 소문자·공백→`-`(유니코드 허용), shortId는 base62 8자. 구 `/tracks/{uuid}`·slug 변경 시 **301** |
| 플레이리스트 | `/playlists/{slug}-{shortId}` (공개만) |
| 크리에이터 | `/creators/{handle}` — 엔진에 `handle` 신설(소문자 영숫자·`_`, 3~30자, 예약어 차단) |
| 랜딩 | `/explore/mood/{mood}`, `/explore/genre/{genre}`, `/explore/use/{useCase}` — 키는 영문 고정, 화면 문구만 번역 |
| 언어 | ko 접두 없음, 나머지 `/en`·`/ja`·`/zh-cn`·`/th`. 모든 공개 페이지에 5개 hreflang + `x-default`(= `/en/…`) |
| 접두 없는 화면 | 인증·개인·결제·스튜디오·콘솔·관리자·매장 플레이어는 **언어 접두를 만들지 않는다**(`@nuxtjs/i18n` 페이지 단위 `i18n: false` 또는 `customRoutes`). 엔진·메일·Toss가 만드는 주소(`/oauth/callback`, `/reset-password`, `/verify-email`, `/checkout/confirm`)가 언어와 무관하게 동작해야 하기 때문 |
| 임베드 | `/embed/track/{shortId}`, `/embed/playlist/{shortId}` — 언어는 `?lang=` 선택값 |
| 자동 이동 | **Accept-Language로 강제 리다이렉트하지 않는다**(크롤러가 한 언어만 보게 됨). 대신 상단 배너 "이 페이지를 English로 보기" 제안 1회 |
| canonical | 언어별 자기 자신. 필터·정렬 쿼리는 canonical에서 제거 |
| 후행 슬래시 | 없음으로 통일, 있으면 301 |

### 3.1 언어별 중복 콘텐츠 방지
트랙 제목은 번역하지 않으므로 5개 언어 페이지가 "껍데기만 다른 같은 페이지"가 되기 쉽다. 그래서 공개 페이지마다 **언어별 고유 텍스트**를 서버에서 만든다.
- 트랙 설명 자동 문장: 무드·장르·BPM·키·길이·용도를 언어별 템플릿으로 조합(예: EN "A 120 BPM chill electronic track in A minor, suited for cafés and study sessions.").
- AI 생성 표시·인증 상태 설명, 비슷한 곡·플레이리스트 링크 블록(내부 링크망).
- 크리에이터 소개는 원문 + "자동 번역" 표기 선택 제공(G-L 전에는 원문만).

---

## 4. 구조화 데이터 (JSON-LD, `nuxt-schema-org`)

| 페이지 | 타입 | 필수 속성 |
|---|---|---|
| 전역 | `Organization`, `WebSite` | name, url, logo, sameAs(공식 SNS — PO 제공) |
| 트랙 | `MusicRecording` | name, url, image, duration(ISO 8601), byArtist(`Person` 또는 `MusicGroup` → 크리에이터 URL), genre, datePublished, inLanguage(가사 언어 있을 때), `isrcCode`(ISRC 있을 때 — 엔진 `domain/isrc.go`), `license`(인증서 URL) |
| 플레이리스트 | `MusicPlaylist` | name, numTracks, track(`MusicRecording` 목록 상위 50) |
| 크리에이터 | `Person`/`MusicGroup` + `ProfilePage` | name, url, image, 대표곡 |
| 모든 하위 페이지 | `BreadcrumbList` | 홈 › 탐색 › 무드 › … |
| 가이드·FAQ | `Article`, `FAQPage`(화면에 보이는 Q&A만) | headline, dateModified, author(AVIBE) |

- 화면에 없는 정보를 JSON-LD에 넣지 않는다(구조화 데이터 스팸 정책).
- CI에서 대표 페이지 JSON-LD를 schema.org 검증기 로직으로 파싱 검사.

---

## 5. 메타·공유 미리보기

- 언어별 `<title>` 템플릿: 트랙 `{제목} – {크리에이터} | AVIBE`, 랜딩 `{무드} AI 음악 – 상업적 이용권 확인 | AVIBE`(번역).
- description 150자 내외, 언어별.
- Open Graph: `og:type=music.song`(트랙)·`music.playlist`·`profile`, `music:duration`, `music:musician`, `og:locale` + `og:locale:alternate` 4개.
- **동적 OG 이미지**(`nuxt-og-image`): 커버 + 제목 + 크리에이터 + ◈ Seal(인증서 있을 때만). 1200×630. CJK·태국어 글꼴 내장 — **태국어 제목 렌더링을 수용 검사에 포함**, 깨지면 텍스트 없는 커버 합성으로 폴백.
- `<meta name="robots" content="max-image-preview:large">` — Google Discover 노출 조건.
- X(트위터) `twitter:card=player` + `/embed/track/:id` (승인 필요 — 승인 전 `summary_large_image`).
- 카카오톡·LINE·Facebook·Threads는 OG를 그대로 쓴다. 카카오 공유 디버거·LINE 캐시 갱신 절차를 운영 문서에 기록.

---

## 6. 크롤링·색인 파이프라인

### 6.1 sitemap
- `/sitemap_index.xml` → 언어 × 유형별 하위 sitemap(tracks·playlists·creators·landings·guides·static), 파일당 5만 URL 이하, **실제 수정 시각 lastmod**, 각 URL에 hreflang alternates.
- 데이터는 엔진 신규 `GET /api/v1/seo/sitemap/{type}?cursor=`(READY·공개만, 1시간 캐시)에서 받는다.

### 6.2 robots.txt
```
User-agent: *
Allow: /
Disallow: /api/
Disallow: /account
Disallow: /me/
Disallow: /library
Disallow: /studio
Disallow: /business/console
Disallow: /admin
Disallow: /store/player
Disallow: /checkout
Disallow: /login
Disallow: /signup
Sitemap: https://avibemusic.com/sitemap_index.xml
```
- AI 검색·답변 봇(OAI-SearchBot, ChatGPT-User, PerplexityBot, Claude-SearchBot 등)과 학습 봇 모두 **공개 텍스트 페이지는 허용**한다(음원 파일은 서명 URL이라 원래 수집 불가). 과도 수집 봇은 Cloudflare 레이트 리밋으로 대응.
- 언어 접두 경로에도 같은 Disallow가 적용되도록 `@nuxtjs/robots`로 생성.

### 6.3 즉시 색인 — IndexNow
- 참여 엔진: **Bing · Naver · Yandex · Seznam · Yep**(한 곳에 보내면 공유됨). Google은 미참여.
- 엔진(Go)이 트랙 승인(READY)·수정·내림, 플레이리스트 공개 전환, 크리에이터 생성 시 해당 URL(5개 언어 전부)을 IndexNow로 발송. 키 파일 `/{key}.txt`를 Nuxt가 서빙. 배치·재시도·일일 상한 로그.
- Bing 색인은 ChatGPT 검색·Copilot·DuckDuckGo 노출의 기반이므로 우선순위가 높다.
- Google: sitemap 제출 + 내부 링크로만(구 sitemap ping은 폐지, Indexing API는 음악 페이지 대상 아님).

### 6.4 등록할 검색 콘솔 (PO 몫 — 소유 확인 메타/DNS는 Muse가 준비)
| 시장 | 콘솔 |
|---|---|
| 전 세계·일본·태국 | Google Search Console (Yahoo! JAPAN 검색은 Google 색인 기반) |
| 전 세계 | Bing Webmaster Tools (+ IndexNow) |
| 한국 | 네이버 서치어드바이저, Daum 웹마스터도구 |
| 중국 | Baidu 站长平台 — ICP 없이는 노출 제한. **보류**(사업 판단) |

---

## 7. "메타 엔진"형 노출 확대 — 검색 결과 밖에서도 보이게

1. **AI 답변 엔진 최적화(GEO)**
   - `/llms.txt`: AVIBE가 무엇인지, 인증 방식, 주요 랜딩·가이드 링크를 사실 위주로 정리(5개 언어 링크).
   - `/trust`, `/business`, `/guides/**`는 질문형 소제목 + 짧은 정답 문단 + 출처 링크 구조(답변 엔진이 인용하기 쉬운 형태).
   - 엔티티 일관성: Organization 이름·로고·sameAs를 모든 페이지 동일하게. 이후 Wikidata 항목 등록(PO).
2. **가이드 허브 `/guides/**`** — 검색 의도가 가장 강한 질문에 답하는 페이지:
   "AI로 만든 음악을 매장에서 틀어도 되나요?", "Suno 유료 플랜 곡을 상업적으로 쓸 수 있나요?", "유튜브 영상에 AI 음악을 쓸 때 확인할 것" 등. 근거는 `docs/handoff/20261111/음악생성AI_상업이용_플랫폼조사_재설계_20261111.md`.
   - 다른 서비스 이름은 **사실 설명으로만** 쓴다(우월성 비교·"대체" 표현 금지). 약관 인용에는 확인 날짜를 붙이고 분기마다 재확인.
3. **프로그래매틱 랜딩**: 무드 × 용도 조합(예: "카페용 재즈", "운동용 일렉트로닉")을 **곡이 최소 12개 이상인 조합만** 생성. 미달 조합은 404(얇은 페이지 양산 금지).
4. **임베드 + oEmbed**: `/embed/track/:id`, `/embed/playlist/:id` + `GET /oembed?url=` + `<link rel="alternate" type="application/json+oembed">`. 블로그·노션·디스코드·워드프레스에 붙으면 백링크와 유입이 동시에 생긴다. 임베드에 "AVIBE에서 듣기" 링크 필수, 재생은 기존 재생 권한 규칙 적용.
5. **공유 루프**: 트랙·플레이리스트 공유 버튼(링크 복사·카카오·LINE·X·Facebook), 공유 링크에 `utm_source`.
6. **앱 연결**: `/.well-known/assetlinks.json`(Android App Links), `/.well-known/apple-app-site-association`(iOS Universal Links) — 검색 결과에서 앱이 열리게.
7. **DSP 유통 연계**: 관리자 유통 기능(`views/admin/Distribution.vue`)으로 외부 플랫폼에 나간 곡은 트랙 페이지에 "다른 곳에서 듣기" 링크(사실 표기) — 외부 플랫폼 → AVIBE 역링크 확보.

---

## 8. 성능 예산 (검색 순위·전환 공통)
| 지표 | 기준(p75, 모바일) |
|---|---|
| LCP | ≤ 2.5s |
| INP | ≤ 200ms |
| CLS | ≤ 0.1 |
| TTFB(엣지 캐시 적중) | ≤ 200ms |
| 공개 페이지 초기 JS | ≤ 170KB gzip (hls.js·플레이어 전체는 지연 로드) |
| 글꼴 | 언어별 서브셋, 첫 화면 글꼴 1개만 preload |

Lighthouse CI: `/`, `/en/tracks/…`, `/th/playlists`, `/ja/business`를 PR마다 측정, SEO 점수 100·성능 90 미만이면 실패.

---

## 8.1 사이트 내 검색 (다국어)
- 현재 탐색 검색은 `title ILIKE` 한 가지(`backend/internal/repository/catalog.go:238`)라 크리에이터·태그·무드로 찾을 수 없고, 데이터가 늘면 느려진다.
- 이번 패키지: Postgres `pg_trgm` GIN 인덱스 + 제목·크리에이터 활동명·태그 표시명(5개 언어) 대상 검색, 유사도 순 정렬. 태국어·일본어·중국어처럼 띄어쓰기가 없거나 다른 언어도 부분 일치로 찾히는지 테스트에 포함.
- 곡이 수만 개를 넘으면 전용 검색 엔진(CJK·태국어 형태소 지원 여부 기준) 도입을 별도 검토.

## 9. 엔진(Go) 추가 사항 요약
- `tracks.short_id`, `tracks.slug`, `playlists.short_id/slug`, `users.handle`(크리에이터) + 마이그레이션·백필·유일성.
- `GET /api/v1/seo/sitemap/{type}`, `GET /api/v1/seo/resolve?path=`(구 URL → 정본 301 대상), `GET /oembed`.
- IndexNow 발송 큐(Redis) + 재시도 + 감사 로그.
- 공개 메타 응답에 SEO용 필드(설명 생성용 무드·장르·BPM·키·용도·ISRC·인증서 번호·수정 시각).
- 캐시 퍼지: 트랙·플레이리스트 변경 시 Cloudflare 캐시 퍼지 API 호출(토큰은 PO 발급).
