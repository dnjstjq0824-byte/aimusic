# Cloudflare R2 & CDN 프로덕션 전환 및 보안 가이드 (Phase 6)

본 문서는 로컬 개발 환경(MinIO)에서 프로덕션 Cloudflare 인프라(Cloudflare R2 + CDN + WAF + Signed URL)로 무중단 전환하기 위한 가이드입니다.

---

## 1. 아키텍처 개요

```mermaid
flowchart LR
  subgraph Clients["Clients"]
    WEB["Web Client (Vue 3)"]
    MOB["Mobile Client (Flutter)"]
  end

  subgraph Cloudflare["Cloudflare Edge"]
    WAF["Cloudflare WAF / DDoS Guard"]
    CDN["Cloudflare CDN (HLS Caching)"]
    R2_PUB[("Cloudflare R2<br/>public-hls")]
    R2_STG[("Cloudflare R2<br/>staging-private")]
  end

  subgraph Backend["Kubernetes / Cloud VM"]
    API["Go API Server (:8080)"]
    WRK["Go Media Worker (FFprobe/FFmpeg)"]
    PG[("PostgreSQL 16")]
    RD[("Redis 7")]
  end

  WEB -->|HTTPS API Requests| WAF --> API
  MOB -->|HTTPS API Requests| WAF --> API
  API -->|SigV4 PresignPut| R2_STG
  API -->|SigV4 PresignGet / Signed HMAC URL| R2_PUB
  WEB -->|HLS Stream Requests (.m3u8/.ts)| CDN --> R2_PUB
  MOB -->|HLS Stream Requests (.m3u8/.ts)| CDN --> R2_PUB
  WRK -->|Download Staging / Upload Transcoded| R2_STG & R2_PUB
```

---

## 2. Cloudflare R2 버킷 구성

1. **`staging-private` (비공개 스테이징 버킷)**
   - 용도: 파트너가 업로드한 원본 오디오(WAV/FLAC/MP3) 및 커버 이미지 임시 보관
   - 접근 제어: **공개 접근 차단 (Public Access Disabled)**
   - API 서버가 발급한 SigV4 Presigned PUT/GET URL을 통해서만 접근
   - 수명 주기(Lifecycle): 변환 완료 후 또는 반려 시 즉시 삭제 (보존 기간 7일 만료 규칙 권장)

2. **`public-hls` (스트리밍 배포 버킷)**
   - 용도: HLS 마스터 플레이리스트(`master.m3u8`), 서브 스트림(`stream_0.m3u8`, `stream_1.m3u8`), 오디오 세그먼트(`.ts`), 파형 데이터(`peaks.json`)
   - 도메인 연동: Cloudflare Custom Domain (예: `https://cdn.aimusic.example.com`)
   - 캐시 헤더:
     - `.m3u8`: `Cache-Control: public, max-age=60, s-maxage=60`
     - `.ts`: `Cache-Control: public, max-age=86400, s-maxage=604800, immutable`

---

## 3. 환경 변수 전환 매핑 (`.env`)

로컬 MinIO 환경에서 Cloudflare R2로 전환할 때는 코드 변경 없이 `.env` 파일의 스토리지 드라이버 변수만 교체합니다.

```ini
# ==============================================================================
# Storage Driver Selection (local vs r2)
# ==============================================================================
STORAGE_DRIVER=r2

# ==============================================================================
# Cloudflare R2 S3-Compatible API Credentials
# Cloudflare 대시보드 -> R2 -> Manage R2 API Tokens 에서 발급
# ==============================================================================
R2_ACCOUNT_ID=your_cloudflare_account_id_hex
R2_ACCESS_KEY_ID=your_r2_access_key_id
R2_SECRET_ACCESS_KEY=your_r2_secret_access_key

# R2 버킷 명칭
R2_STAGING_BUCKET=staging-private
R2_PUBLIC_HLS_BUCKET=public-hls

# R2 S3 호환 엔드포인트 URL
# https://<R2_ACCOUNT_ID>.r2.cloudflarestorage.com
S3_ENDPOINT=https://your_cloudflare_account_id_hex.r2.cloudflarestorage.com
S3_REGION=auto
S3_USE_SSL=true

# ==============================================================================
# Cloudflare CDN 스트리밍 도메인
# ==============================================================================
CDN_BASE_URL=https://cdn.aimusic.example.com

# ==============================================================================
# HLS 스트리밍 Signed URL 토큰 설정
# ==============================================================================
HLS_SIGNING_KEY=your_production_hmac_sha256_secret_key
HLS_URL_TTL_SECONDS=300
```

---

## 4. Cloudflare WAF 및 보안 설정 가이드

1. **Signed URL 검증 / Token Authentication**
   - API 서버가 클라이언트에게 발급하는 HLS URL은 `exp`(만료 시각), `jti`(일회성 토큰 ID), `sig`(HMAC-SHA256 서명) 쿼리 파라미터를 포함합니다.
   - Cloudflare CDN 엣지에서 WAF Custom Rule 또는 Cloudflare Workers를 통해 `sig` 유효성을 검증하고, 만료된 토큰의 `.m3u8` 요청을 엣지에서 즉시 차단(403)하도록 구성할 수 있습니다.

2. **DDoS 방어 및 Rate Limiting**
   - 인증 엔드포인트(`/api/v1/auth/login`, `/api/v1/auth/register`): IP당 분당 20회 초과 시 429 차단
   - 스트리밍 발급 엔드포인트(`/api/v1/tracks/:id/stream`): 사용자당 분당 60회 초과 시 레이트 리밋 적용

3. **CORS (Cross-Origin Resource Sharing)**
   - R2 버킷 CORS 정책:
     ```json
     [
       {
         "AllowedOrigins": [
           "https://aimusic.example.com",
           "https://partner.aimusic.example.com",
           "https://admin.aimusic.example.com"
         ],
         "AllowedMethods": ["GET", "PUT", "HEAD"],
         "AllowedHeaders": ["*"],
         "ExposeHeaders": ["ETag", "Content-Length"],
         "MaxAgeSeconds": 3600
       }
     ]
     ```

---

## 5. 배포 체크리스트

- [ ] `.env` 파일이 Git에 절대 커밋되지 않았는지 확인 (`.gitignore` 적용 완료)
- [ ] R2 API Token에 `staging-private` 및 `public-hls` 버킷의 읽기/쓰기 권한이 부여되었는지 확인
- [ ] PostgreSQL 접속 비밀번호 및 `AES256_KEY_BASE64` 마스터 암호화 키가 최소 32바이트 이상 엔트로피로 주입되었는지 확인
- [ ] Cloudflare CDN 캐시 퍼지 API 연동 준비 (저작권 분쟁으로 인한 테이크다운 발생 시 `.m3u8` 즉시 캐시 무효화)
