# 소셜 로그인 설정 (WI-A1)

엔진은 Google·Kakao·Apple·Naver·LINE 5개 공급자를 지원한다. 키(환경변수)만
넣으면 바로 작동하고, 키가 없는 공급자는 버튼이 안 보일 뿐이다.
WeChat은 어댑터 인터페이스만 있고 구현은 보류(중국 법인 필요)다.

## 0. 공통

- 콜백 URI 규칙: `{OAUTH_REDIRECT_BASE_URL}/{google|kakao|apple|naver|line}/callback`
- 운영값: `https://avibemusic.com/api/v1/auth/oauth/{google|kakao|apple|naver|line}/callback`
- 각 개발자 콘솔에 위 값을 **정확히 그대로** 등록한다(뒤 슬래시 없음).
- 키 반영 = `.env`에 추가 + API 컨테이너 재시작. 코드·DB 변경 없음.
- API 시작 로그에 비활성 공급자와 빠진 변수 이름만 찍힌다(값 노출 없음).
- 환경변수 표:

| 공급자 | 필수 | 선택 |
|---|---|---|
| Google | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | |
| Kakao | `KAKAO_CLIENT_ID` (REST API 키) | `KAKAO_CLIENT_SECRET` (콘솔에서 켰을 때만) |
| Apple | `APPLE_SERVICES_ID`, `APPLE_TEAM_ID`, `APPLE_KEY_ID`, `APPLE_PRIVATE_KEY` (p8 PEM 전문) | |
| Naver | `NAVER_CLIENT_ID`, `NAVER_CLIENT_SECRET` | |
| LINE | `LINE_CHANNEL_ID`, `LINE_CHANNEL_SECRET` | |
| 공통 | `OAUTH_REDIRECT_BASE_URL`, `OAUTH_STATE_SECRET` | |

`OAUTH_STATE_SECRET`이 비면 `JWT_ACCESS_SECRET`으로 대체하고 경고를 남긴다.
`APPLE_PRIVATE_KEY`는 p8 파일 내용 전체(개행 포함)를 넣는다.

## 1. Google

1. Google Cloud Console → APIs & Services → OAuth consent screen: 앱 등록.
2. Credentials → Create Credentials → OAuth client ID → Web application.
3. Authorized redirect URIs에 `{base}/google/callback` 추가.
4. 발급된 Client ID/Secret을 `.env`에 입력.
5. Scopes: `openid email profile` (코드에 고정). 외부 공개 전환 시 Google
   OAuth 검증(민감 범위 심사)이 필요할 수 있다.

## 2. Kakao

1. Kakao Developers → 내 애플리케이션 → 제품 설정 → 카카오 로그인 → 활성화.
2. Redirect URI에 `{base}/kakao/callback` 등록 후 저장.
3. 동의항목: 닉네임(필수) + 이메일. **이메일 동의항목은 비즈 앱 전환
   (사업자번호 등록)이 필요**하고, 검수를 통과해야 실사용자에게 보인다.
4. REST API 키를 `KAKAO_CLIENT_ID`에 입력. 보안 → Client Secret 코드를
   사용하면 그 값을 `KAKAO_CLIENT_SECRET`에 입력(미사용이면 빈칸).
5. 사용자가 이메일 제공을 거부하면 엔진이 이메일 입력 화면
   (`/signup/complete-email`)으로 보낸다.

## 3. Apple

1. developer.apple.com → Certificates, Identifiers & Profiles → Identifiers →
   Services IDs 신규 등록. 이 식별자가 `APPLE_SERVICES_ID`
   (예: `com.avibe.signin`)다.
2. Services ID에서 Sign In with Apple 체크 → Configure:
   - Primary App ID 선택.
   - Website URLs → Domains에 `avibemusic.com` 등록. **도메인 확인**(Apple이
     제시하는 확인 절차)을 먼저 마쳐야 한다.
   - Return URLs에 `{base}/apple/callback` 등록.
3. Keys → Sign in with Apple 키 생성 → Key ID가 `APPLE_KEY_ID`.
   이때 1회만 내려받을 수 있는 p8 파일을 `APPLE_PRIVATE_KEY`에 통째로 입력.
4. 우측 상단 Membership의 Team ID가 `APPLE_TEAM_ID`.
5. Scopes: `name email` (코드에 고정). 이름은 최초 인증 때만 오므로 엔진이
   그때 저장하고, 비공개 릴레이 이메일도 그대로 허용한다.
6. Apple 콜백은 `form_post`(교차 사이트 POST)라 state 쿠키가
   `SameSite=None; Secure`로 나간다. **Apple 로그인은 HTTPS 환경에서만
   동작**한다(로컬 http 불가).

## 4. Naver

1. Naver Developers → 애플리케이션 등록 → 사용 API에서 네이버 로그인 선택.
2. 서비스 URL과 Callback URL(`{base}/naver/callback`) 등록.
3. 제공 정보에서 회원이름·이메일을 선택. **이메일 등 제공 항목은 검수
   (애플리케이션 검수 요청 → 승인)를 통과해야 실사용자에게 보인다.**
4. Client ID/Secret을 `.env`에 입력.
5. 이메일 제공을 거부한 사용자는 이메일 입력 화면으로 간다.

## 5. LINE

1. LINE Developers Console → Provider → LINE Login 채널 생성.
2. Channel ID/Secret을 `.env`에 입력.
3. LINE Login 설정 → Callback URL에 `{base}/line/callback` 등록.
4. Scopes: `openid profile email` (코드에 고정). **이메일 주소 권한은 별도
   신청·심사**가 필요하다(개인정보처리방침 URL 등록 후 Email address
   permission 신청). 권한이 없으면 이메일 입력 화면으로 간다.

## 6. 동작 규칙 (보안)

- (C-11) 같은 이메일의 기존 계정 자동 연결은 공급자가 이메일 인증을
  보증할 때만: Google·Apple `email_verified`, Kakao
  `is_email_verified` + `is_email_valid`. 그 외에는 "기존 계정으로 로그인
  후 연결" 안내(`ERR_OAUTH_LINK_LOGIN_REQUIRED`) → `/account/connections`.
- (C-12) state는 서명 + 10분 만료 + 1회용 + HttpOnly 쿠키 nonce 바인딩.
  Google·Kakao·Apple·LINE은 PKCE(S256), Apple·LINE은 OIDC nonce 검증.
  Naver는 공급자가 PKCE를 지원하지 않아 plain code flow다.
- (C-16) 콜백은 토큰 대신 60초·1회용 교환 코드를 넘기고, 웹이
  `POST /auth/oauth/exchange`로 토큰을 받는다. URL에 토큰이 남지 않는다.
- 연결/해제: 로그인 상태에서 `/account/connections`. 마지막 로그인 수단은
  해제 불가(`ERR_OAUTH_LAST_METHOD`). 엔진은 사용 가능한 비밀번호와
  소셜 가입 시 만든 무작위 비밀번호를 구분할 수 없어 보수적으로 막는다.

## 7. `.env` 예시 (값은 빈칸)

```env
OAUTH_REDIRECT_BASE_URL=https://avibemusic.com/api/v1/auth/oauth
OAUTH_STATE_SECRET=
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
KAKAO_CLIENT_ID=
KAKAO_CLIENT_SECRET=
APPLE_SERVICES_ID=
APPLE_TEAM_ID=
APPLE_KEY_ID=
APPLE_PRIVATE_KEY=
NAVER_CLIENT_ID=
NAVER_CLIENT_SECRET=
LINE_CHANNEL_ID=
LINE_CHANNEL_SECRET=
```

## 8. 키 없이 검증하기

실키 없이도 스텁 서버 단위 테스트로 전 공급자 흐름을 검증한다:

```sh
cd backend && go test ./internal/service/ -run TestOAuth -count=1 -v
```

CI E2E는 키 없이도 도는 가드(공급자 목록·start 거부·교환/연결 가드)를
포함한다. 실키 발급 후 첫 연동 실측은 PO 수동 확인 항목이다.
