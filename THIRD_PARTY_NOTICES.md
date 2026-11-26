# Third-Party Notices (AVIBE)

Distributing AVIBE redistributes the open-source components below. Direct
dependencies are enumerated with their licenses (verified 2026-11-26 against
installed copies); transitive dependencies are summarized by license family
with lockfile pointers to the full closure. No GPL/AGPL/LGPL-only component
was found in any stack.

Refresh procedure: re-run the census per stack and update the counts.

- Go: `cd backend && go list -m all` (53 modules), licenses in module zips.
- npm: `web/package-lock.json` (1017 packages) + installed
  `node_modules/*/package.json` license fields.
- Flutter: `mobile/pubspec.lock` (111 packages) + pub-cache LICENSE files.
- Python: `generator/requirements.txt` (+ `-test.txt` for CI only).

## Direct dependencies

### Backend (Go)

| Module | Version | License |
|---|---|---|
| github.com/gin-gonic/gin | v1.10.0 | MIT |
| github.com/golang-jwt/jwt/v5 | v5.2.1 | MIT |
| github.com/google/uuid | v1.6.0 | BSD-3-Clause |
| github.com/jackc/pgx/v5 | v5.7.1 | MIT |
| github.com/redis/go-redis/v9 | v9.7.0 | BSD-2-Clause |
| golang.org/x/crypto | v0.31.0 | BSD-3-Clause |

### Generator (Python)

| Package | License |
|---|---|
| fastapi | MIT |
| pydantic | MIT |
| uvicorn | BSD-3-Clause |

(Test-only: pytest, httpx, ruff — never shipped.)

### Web (npm, runtime + build)

All MIT except where noted. Versions are the installed ones.

| Package | Version | License |
|---|---|---|
| @nuxtjs/i18n | 10.6.0 | MIT |
| @nuxtjs/robots | 7.0.2 | MIT |
| @nuxtjs/sitemap | 9.1.0 | MIT |
| @pinia/nuxt | 1.0.3 | MIT |
| @resvg/resvg-js | 2.6.2 | MPL-2.0 (see note) |
| @resvg/resvg-wasm | 2.6.2 | MPL-2.0 (see note) |
| @tosspayments/tosspayments-sdk | 2.10.0 | MIT |
| @unhead/vue | 3.4.3 | MIT |
| hls.js | 1.7.4 | Apache-2.0 |
| nuxt | 4.6.4 | MIT |
| nuxt-og-image | 7.0.7 | MIT |
| nuxt-schema-org | 7.0.1 | MIT |
| nuxt-site-config | 5.0.3 | MIT |
| pinia | 4.1.0 | MIT |
| satori | 0.63.2 | MPL-2.0 (see note) |
| unhead | 3.4.3 | MIT |
| vue | 3.5.45 | MIT |
| vue-router | 4.6.4 | MIT |
| web-vitals | 6.2.3 | Apache-2.0 |
| @nuxt/eslint | 1.17.0 | MIT |
| @nuxt/kit | 4.6.4 | MIT |
| @playwright/test | 1.57.0 | Apache-2.0 |
| @types/node | 26.6.6 | MIT |
| @vitejs/plugin-vue | 5.2.4 | MIT |
| @vitest/coverage-v8 | 2.1.9 | MIT |
| autoprefixer | 10.6.2 | MIT |
| eslint | 10.15.0 | MIT |
| happy-dom | 21.0.4 | MIT |
| postcss | 8.5.32 | MIT |
| tailwindcss | 3.4.19 | MIT |
| typescript | 5.9.3 | Apache-2.0 |
| vite | 5.4.21 | MIT |
| vitest | 2.1.9 | MIT |
| vue-tsc | 2.2.12 | MIT |

### Mobile (Flutter)

| Package | Version | License |
|---|---|---|
| flutter_riverpod | 2.6.1 | MIT |
| dio | 5.11.1 | MIT |
| go_router | 14.8.1 | BSD-3-Clause |
| just_audio | 0.9.46 | MIT |
| audio_service | 0.18.19 | MIT |
| audio_session | 0.1.25 | MIT |
| flutter_secure_storage | 9.2.4 | BSD-3-Clause |
| flutter_svg | 2.3.2 | MIT |
| json_annotation | 4.12.0 | BSD-3-Clause |
| intl | 0.20.3 | BSD-3-Clause |
| url_launcher | 6.3.3 | BSD-3-Clause |
| app_links | 7.2.2 | Apache-2.0 |
| flutter_secure_storage_platform_interface | 1.1.2 | BSD-3-Clause (tests) |
| flutter_lints | 4.0.0 | BSD-3-Clause (lint only) |
| Flutter SDK | 3.47.8 | BSD-3-Clause |

### Fonts (self-hosted, SIL OFL 1.1)

Full text: `web/public/fonts/licenses/OFL-1.1.txt`. Attribution: Pretendard
(c) Kil Hyung-jin; Noto (c) Google/Adobe.

| Family | Source | Version |
|---|---|---|
| Pretendard Variable | npm pretendard | 1.3.9 |
| Pretendard JP Variable | npm pretendard-jp | 1.3.9 |
| Noto Sans SC | npm @fontsource/noto-sans-sc | 5.3.1 |
| Noto Sans Thai | npm @fontsource/noto-sans-thai | 5.3.0 |

## Transitive closure (summary, 2026-11-26 census)

- Go (53 modules): MIT 27, Apache-2.0 6, BSD-3-Clause 17, ISC 1
  (davecgh/go-spew), BSD-2-Clause 1 (gopkg.in/check.v1), MIT/Apache-2.0
  dual 1 (yaml.v3). Copyleft: none.
- npm (1017 packages in package-lock.json): MIT ~898 (incl. installed
  690 + uninstalled optional-platform siblings 208), ISC 37, Apache-2.0
  29, BlueOak-1.0.0 15, BSD-3-Clause 8, BSD-2-Clause 5, MPL-2.0 27
  (satori/lightningcss/resvg + platform siblings), 0BSD 1, CC0/MIT-dual
  3. GPL/AGPL/LGPL-only: none.
- Flutter (111 packages in pubspec.lock): permissive only (MIT,
  BSD-2/3-Clause, Apache-2.0). Copyleft: none.
- Python: fastapi/uvicorn/pydantic (+ pinned CI-only pytest/httpx/ruff).

## Special notices

- MPL-2.0 components (satori, lightningcss, @resvg/resvg-js,
  resvg-wasm) are used unmodified: satori/resvg render OG images
  server-side, lightningcss runs at build time. No copyleft trigger;
  corresponding source is available from the upstream npm registry
  entries. Modifying or embedding these files changes the analysis.
- caniuse-lite (autoprefixer/browserslist build-time data) is
  CC-BY-4.0 (c) Ben Briggs: https://creativecommons.org/licenses/by/4.0/
- node-forge offers BSD-3-Clause OR GPL-2.0; this project uses it under
  BSD-3-Clause only (via listhen).
- argparse (Python-2.0) and tosource (zlib-style as-is) appear only in
  the build toolchain, not in shipped artifacts.
- ACE-Step generator weights: the ACE-Step CODE is MIT (see
  generator/server.py), but the WEIGHT license terms were not verified
  from the network. Confirm the weight terms before distributing any
  model artifact.

## AVIBE's own license

See LICENSE at the repository root.
