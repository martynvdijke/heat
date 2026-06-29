## [1.41.4](https://github.com/martynvdijke/heat/compare/v1.41.3...v1.41.4) (2026-06-29)

## [1.41.3](https://github.com/martynvdijke/heat/compare/v1.41.2...v1.41.3) (2026-06-26)


### Bug Fixes

* restore admin.js script tag and expose missing functions on window ([afeeea9](https://github.com/martynvdijke/heat/commit/afeeea9b90cce9119d368cf39e48e2f9afffa23e))

## [1.41.2](https://github.com/martynvdijke/heat/compare/v1.41.1...v1.41.2) (2026-06-26)


### Bug Fixes

* expose admin functions globally, add HTMX loading indicators ([cdfabd3](https://github.com/martynvdijke/heat/commit/cdfabd3d4de668c67b901c1bd9ece808c215a95e))

## [1.41.1](https://github.com/martynvdijke/heat/compare/v1.41.0...v1.41.1) (2026-06-26)


### Bug Fixes

* add IDs to rounds/seasons sub-tab buttons so loadRoundsList triggers on tab show ([5a64eeb](https://github.com/martynvdijke/heat/commit/5a64eebb2dd6372d25f4f7316f2b3f630e02ce2a))

# [1.41.0](https://github.com/martynvdijke/heat/compare/v1.40.0...v1.41.0) (2026-06-26)


### Features

* add editable round drafts with finalization and stats integration ([c399ce1](https://github.com/martynvdijke/heat/commit/c399ce1de6f5a7fd6db22174979761f2440e69db))

# [1.40.0](https://github.com/martynvdijke/heat/compare/v1.39.1...v1.40.0) (2026-06-25)


### Bug Fixes

* add regression tests for gzip ws exclusion, SQLite PRAGMAs, and admin page timing ([ab6daaa](https://github.com/martynvdijke/heat/commit/ab6daaa716a6368b275bde2795360f1e67387237))


### Features

* add team name to home page, fix E2E tests, optimize backend ([c4e207f](https://github.com/martynvdijke/heat/commit/c4e207fd977a4d1155f4cadfcd38854986746589))

## [1.39.1](https://github.com/martynvdijke/heat/compare/v1.39.0...v1.39.1) (2026-06-25)

# [1.39.0](https://github.com/martynvdijke/heat/compare/v1.38.3...v1.39.0) (2026-06-24)


### Bug Fixes

* set Content-Type header in admin handler for gzip compatibility ([85dbf54](https://github.com/martynvdijke/heat/commit/85dbf54e2720e683f0373245840fe0676df7e988)), closes [#adminCategories](https://github.com/martynvdijke/heat/issues/adminCategories)


### Features

* add e2e test coverage change proposal with specs and tasks ([e252443](https://github.com/martynvdijke/heat/commit/e252443cea247a309460297416f0ccf2b2ad70e7))


### Performance Improvements

* backend optimizations - multi-conn, indexes, N+1 fixes, caching, compression ([b83dceb](https://github.com/martynvdijke/heat/commit/b83dcebe21770f7fd3b682799daa3aaf0a082096))

## [1.38.3](https://github.com/martynvdijke/heat/compare/v1.38.2...v1.38.3) (2026-06-22)


### Bug Fixes

* **deps:** update all non-major dependencies ([#44](https://github.com/martynvdijke/heat/issues/44)) ([2c6f6b3](https://github.com/martynvdijke/heat/commit/2c6f6b3d59597c1e8afe7d191e7cbc010a1dc6c9))

## [1.38.2](https://github.com/martynvdijke/heat/compare/v1.38.1...v1.38.2) (2026-06-20)

## [1.38.1](https://github.com/martynvdijke/heat/compare/v1.38.0...v1.38.1) (2026-06-19)


### Bug Fixes

* **deps:** update all non-major dependencies ([#43](https://github.com/martynvdijke/heat/issues/43)) ([8e22e2f](https://github.com/martynvdijke/heat/commit/8e22e2f146480864a2175056d9100a2bfc70d211))

# [1.38.0](https://github.com/martynvdijke/heat/compare/v1.37.4...v1.38.0) (2026-06-19)


### Bug Fixes

* add theme import to stats, seasons, trophies, admin pages ([fed601c](https://github.com/martynvdijke/heat/commit/fed601ccc31ddb8c981cd5924c96888a235911c1))
* update eink e2e tests for data-theme attribute and new localStorage key ([e71027a](https://github.com/martynvdijke/heat/commit/e71027a888d465a614970212fe8830423344628e))


### Features

* make eink a proper third theme alongside light/dark, remove inline FOUC scripts ([cc66b43](https://github.com/martynvdijke/heat/commit/cc66b43c0cf74168ef3ad7d3e2024a09248959bc))

## [1.37.4](https://github.com/martynvdijke/heat/compare/v1.37.3...v1.37.4) (2026-06-18)


### Bug Fixes

* **deps:** update all non-major dependencies to v1.14.46 ([#41](https://github.com/martynvdijke/heat/issues/41)) ([e001ab1](https://github.com/martynvdijke/heat/commit/e001ab19d53f3c507e8188ff8d3cdd62f8df9c9c))

## [1.37.3](https://github.com/martynvdijke/heat/compare/v1.37.2...v1.37.3) (2026-06-17)


### Bug Fixes

* use esbuild in Docker build and fix app/app.go version bump ([e458310](https://github.com/martynvdijke/heat/commit/e458310e3a3e1ed79f111dc9d194756081f34575))

## [1.37.2](https://github.com/martynvdijke/heat/compare/v1.37.1...v1.37.2) (2026-06-17)


### Bug Fixes

* navbar contrast, e-ink visibility and mobile layout ([521f9dc](https://github.com/martynvdijke/heat/commit/521f9dc94bb31ec468aede12421bc03b5422bae0))

## [1.37.1](https://github.com/martynvdijke/heat/compare/v1.37.0...v1.37.1) (2026-06-17)


### Bug Fixes

* exclude controller page from body-layout test (requires auth, redirects to login with flex) ([9910eb3](https://github.com/martynvdijke/heat/commit/9910eb3fc484202111df0fd6efaa32f8cbfa98d9))
* scope page-specific body CSS to prevent cascade conflict ([b914ff5](https://github.com/martynvdijke/heat/commit/b914ff5faca2bb955b697330695126b41daa1e00))

# [1.37.0](https://github.com/martynvdijke/heat/compare/v1.36.1...v1.37.0) (2026-06-15)


### Bug Fixes

* clean up backup dir before prune test to prevent flaky failures ([ff7b9ad](https://github.com/martynvdijke/heat/commit/ff7b9ad6ddee4512c6201f08f7e15e4bac2376a9))


### Features

* add e-ink mode with mobile visibility fix ([2868105](https://github.com/martynvdijke/heat/commit/2868105454395e35b30abae428fdd2809c2e7e25))

## [1.36.1](https://github.com/martynvdijke/heat/compare/v1.36.0...v1.36.1) (2026-06-15)

# [1.36.0](https://github.com/martynvdijke/heat/compare/v1.35.3...v1.36.0) (2026-06-14)


### Bug Fixes

* align templates with JS expectations after frontend consolidation ([1be47f8](https://github.com/martynvdijke/heat/commit/1be47f8faf0c55b5425facbd11b413b5a18486ff))
* **E2E:** skip Playwright actionability check on mobile emulation ([a50b993](https://github.com/martynvdijke/heat/commit/a50b99366d16817ab39fcabdc84d574e39b40f3e))
* prevent concurrent WebSocket write panic in broadcast goroutines ([cdd7b24](https://github.com/martynvdijke/heat/commit/cdd7b24f488813695f1004df2a96b160328dce7e))
* remove redundant ci.yml (replaced by ci.yaml + release.yaml) to fix CI failure on main push ([8afa47e](https://github.com/martynvdijke/heat/commit/8afa47e9189839fa7602cb718439db45e21efcb0))
* resolve CI failures - malformed GeoJSON seed data and JS guard ([ff77ba8](https://github.com/martynvdijke/heat/commit/ff77ba8a9fcab6208cd1763a176b0c2a12beff55))
* resolve CI failures after frontend consolidation ([72279d6](https://github.com/martynvdijke/heat/commit/72279d669e5c7e1345970b0ea1c8b66ef99d734e))


### Features

* consolidate frontend technical debt ([78af2e1](https://github.com/martynvdijke/heat/commit/78af2e1db960df56cd764c3a7899a878632f1f5a))

## [1.35.3](https://github.com/martynvdijke/heat/compare/v1.35.2...v1.35.3) (2026-06-11)


### Bug Fixes

* prevent nil context panic in OTel shutdown and add container restart policy ([6049b6f](https://github.com/martynvdijke/heat/commit/6049b6ff66aec5e93a3e95c79602a86612ba1b07))

## [1.35.2](https://github.com/martynvdijke/heat/compare/v1.35.1...v1.35.2) (2026-06-11)


### Bug Fixes

* **deps:** update all non-major dependencies ([#39](https://github.com/martynvdijke/heat/issues/39)) ([00616a3](https://github.com/martynvdijke/heat/commit/00616a3d58223fa3dd053f6cf638fb1235410618))

## [1.35.1](https://github.com/martynvdijke/heat/compare/v1.35.0...v1.35.1) (2026-06-10)

# [1.35.0](https://github.com/martynvdijke/heat/compare/v1.34.0...v1.35.0) (2026-06-09)


### Bug Fixes

* **deps:** update all non-major dependencies ([1c91698](https://github.com/martynvdijke/heat/commit/1c916986d2723a5f4d31a952eb7fcbad26e2b9dc))
* guard Docker login/push on PRs to prevent failures on fork PRs ([c02fe20](https://github.com/martynvdijke/heat/commit/c02fe209b888b528b45342f66f28904a2266ba52))
* resolve release workflow startup_failure by adding missing actions:read and checks:read permissions to caller job ([0bb0599](https://github.com/martynvdijke/heat/commit/0bb05995b24027282a6c2af9f189a22c2146601f))
* update Playwright tests to match restructured admin UI tabs ([e01be27](https://github.com/martynvdijke/heat/commit/e01be27121252fc33edb7fafb8694df7c4540bbb)), closes [#adminTabs](https://github.com/martynvdijke/heat/issues/adminTabs) [#adminCategories](https://github.com/martynvdijke/heat/issues/adminCategories)
* use githubToken instead of otelToken for otel-cicd-action@v4 ([3034850](https://github.com/martynvdijke/heat/commit/3034850dd9e77d5d1820fe5b8014a5e3f73353c8))


### Features

* add otlpAuthorization input for Bearer auth ([16cb09b](https://github.com/martynvdijke/heat/commit/16cb09b24138e1c18f153af6d4fbd7a417a369ba))
* add umami analytics proposal (openspec) ([1ee59c5](https://github.com/martynvdijke/heat/commit/1ee59c5e55892ba27c8128eb9a515bbc4ff551c8))
* admin/controller UI cleanup - toast system, tab reorg, bug fixes ([8129851](https://github.com/martynvdijke/heat/commit/8129851ce301687989616714ac84ea4ca8837615))

# [1.34.0](https://github.com/martynvdijke/heat/compare/v1.33.0...v1.34.0) (2026-06-06)


### Features

* wire up OTel settings frontend in admin panel ([153e225](https://github.com/martynvdijke/heat/commit/153e225818caabfbe7611da965bf146821fa9ebf))

# [1.33.0](https://github.com/martynvdijke/heat/compare/v1.32.0...v1.33.0) (2026-06-06)


### Features

* add OTel endpoint admin configuration with DB-backed settings ([8e8db8f](https://github.com/martynvdijke/heat/commit/8e8db8f5d266c5c1a9d52887803a3b508d4f20ec))

# [1.32.0](https://github.com/martynvdijke/heat/compare/v1.31.0...v1.32.0) (2026-06-06)


### Features

* add admin Logs UI and onboard settings handlers to structured logging ([d2aa288](https://github.com/martynvdijke/heat/commit/d2aa288206c306fe3df8b4b92a5b276768ea3208))

# [1.31.0](https://github.com/martynvdijke/heat/compare/v1.30.7...v1.31.0) (2026-06-06)


### Features

* add central logging admin tab with OTel integration ([506e00f](https://github.com/martynvdijke/heat/commit/506e00fd3b52dee9f9fa9b8547133c75ec569883))

## [1.30.7](https://github.com/martynvdijke/heat/compare/v1.30.6...v1.30.7) (2026-06-05)


### Bug Fixes

* **deps:** update all non-major dependencies to v1.14.45 ([#37](https://github.com/martynvdijke/heat/issues/37)) ([8d99533](https://github.com/martynvdijke/heat/commit/8d995339a23e5b542ee03c6bdb77212e71a61fb5))

## [1.30.6](https://github.com/martynvdijke/heat/compare/v1.30.5...v1.30.6) (2026-06-03)

## [1.30.5](https://github.com/martynvdijke/heat/compare/v1.30.4...v1.30.5) (2026-05-28)


### Bug Fixes

* **deps:** update all non-major dependencies to v1.44.0 ([#34](https://github.com/martynvdijke/heat/issues/34)) ([96a45a2](https://github.com/martynvdijke/heat/commit/96a45a23d311179f1640dd45581250a2ede815b8))

## [1.30.4](https://github.com/martynvdijke/heat/compare/v1.30.3...v1.30.4) (2026-05-26)


### Bug Fixes

* playwright e2e test flakiness - add retries and timeouts for SQLite contention ([902a564](https://github.com/martynvdijke/heat/commit/902a564b56509026b1c860d27e20c9fc7f6ec4dd))

## [1.30.3](https://github.com/martynvdijke/heat/compare/v1.30.2...v1.30.3) (2026-05-26)


### Bug Fixes

* invalid timezone UTC+1, use Europe/Amsterdam instead ([0424230](https://github.com/martynvdijke/heat/commit/04242304c113071ff2a64c24defdcad779c67d0b))

## [1.30.3](https://github.com/martynvdijke/heat/compare/v1.30.2...v1.30.3) (2026-05-26)


### Bug Fixes

* invalid timezone UTC+1, use Europe/Amsterdam instead ([0424230](https://github.com/martynvdijke/heat/commit/04242304c113071ff2a64c24defdcad779c67d0b))

## [1.30.2](https://github.com/martynvdijke/heat/compare/v1.30.1...v1.30.2) (2026-05-25)


### Bug Fixes

* remove stalePr from renovate.json (no longer valid in Renovate v37) ([bfec7b2](https://github.com/martynvdijke/heat/commit/bfec7b21ed27c200f500bb82cf8474e0a960ee6d))

## [1.30.1](https://github.com/martynvdijke/heat/compare/v1.30.0...v1.30.1) (2026-05-25)


### Bug Fixes

* remove stalePrAge from renovate.json (removed in Renovate v37) ([79c3b35](https://github.com/martynvdijke/heat/commit/79c3b35e79b77a1ab8ac3eaaf4afbfb9b6c789b5))

# [1.30.0](https://github.com/martynvdijke/heat/compare/v1.29.6...v1.30.0) (2026-05-24)


### Features

* add F1-style race start light system + fix web design issues ([0ebdfc0](https://github.com/martynvdijke/heat/commit/0ebdfc088e17731a41da6b6af7d6213ea03c563e))

## [1.29.6](https://github.com/martynvdijke/heat/compare/v1.29.5...v1.29.6) (2026-05-24)


### Bug Fixes

* **racing:** initialize result slices as empty to avoid null JSON response ([a6006b6](https://github.com/martynvdijke/heat/commit/a6006b6b7a2a7a12044ccc96d0b5dfb5c04c2ede))

## [1.29.5](https://github.com/martynvdijke/heat/compare/v1.29.4...v1.29.5) (2026-05-23)

## [1.29.4](https://github.com/martynvdijke/heat/compare/v1.29.3...v1.29.4) (2026-05-23)

## [1.29.3](https://github.com/martynvdijke/heat/compare/v1.29.2...v1.29.3) (2026-05-23)


### Bug Fixes

* **ui:** add aria-hidden to icons, fix FA version, add autocomplete, replace alerts ([4625785](https://github.com/martynvdijke/heat/commit/462578589f4b088edce4d40a895a6aab8aa42162))

## [1.29.2](https://github.com/martynvdijke/heat/compare/v1.29.1...v1.29.2) (2026-05-23)

## [1.29.1](https://github.com/martynvdijke/heat/compare/v1.29.0...v1.29.1) (2026-05-22)


### Bug Fixes

* **deps:** update all non-major dependencies ([#29](https://github.com/martynvdijke/heat/issues/29)) ([f7e9140](https://github.com/martynvdijke/heat/commit/f7e9140fa004ea383a2edc87da19f6f3f8b0229c))

# [1.29.0](https://github.com/martynvdijke/heat/compare/v1.28.1...v1.29.0) (2026-05-21)


### Features

* add new entity schemas and Dutch locale support ([8cc62d6](https://github.com/martynvdijke/heat/commit/8cc62d64321306111dcb5052f694274ae1a9a2db))

## [1.28.1](https://github.com/martynvdijke/heat/compare/v1.28.0...v1.28.1) (2026-05-20)


### Bug Fixes

* ensure Gotify notification always fires on release workflow ([dd0f084](https://github.com/martynvdijke/heat/commit/dd0f0848933862f77ebf0339084794c2f4560ab0))

# [1.28.0](https://github.com/martynvdijke/heat/compare/v1.27.0...v1.28.0) (2026-05-19)


### Bug Fixes

* update admin e2e tests to use htmx button and car_color field ([5e8df5a](https://github.com/martynvdijke/heat/commit/5e8df5a25ba237ee848f29d802182935bc1c906e))


### Features

* integrate Ent ORM for schema management and core entities ([238a863](https://github.com/martynvdijke/heat/commit/238a8635877c38d5fd2a2d815ef806da13c57753))
* migrate CRUD admin panels to htmx ([6244799](https://github.com/martynvdijke/heat/commit/6244799429d4b9d3636977667fd2e4d881af48fd))

# [1.27.0](https://github.com/martynvdijke/heat/compare/v1.26.0...v1.27.0) (2026-05-18)


### Bug Fixes

* compile all Go files in e2e server command (go run . instead of main.go) ([1989739](https://github.com/martynvdijke/heat/commit/19897393b6f7fc71e911d889fcdef13a5bbc412c))


### Features

* add opentelemetry tracing and prometheus metrics support ([6ca1928](https://github.com/martynvdijke/heat/commit/6ca1928c66768c09c876021e6d05762c74952e46))

# [1.26.0](https://github.com/martynvdijke/heat/compare/v1.25.0...v1.26.0) (2026-05-16)


### Bug Fixes

* resolve tech debt - WebSocket mutex, duplicate migrations/functions, AI difficulty bug, dead code ([44e34c3](https://github.com/martynvdijke/heat/commit/44e34c370cbe244ea4a25fef7f4bb16fef0cf0b2))
* resolve test failures for admin login race and car_color selector ([bb9f0ce](https://github.com/martynvdijke/heat/commit/bb9f0ce6f48cde241f516495c57087920993a2a7))
* track static/js/i18n.js in git for CI builds ([5924597](https://github.com/martynvdijke/heat/commit/5924597cc9ece78377acc534a7e64f0d2128c4b3))


### Features

* add constructor/team championship with standings, CRUD, and racer assignment ([9866b8a](https://github.com/martynvdijke/heat/commit/9866b8a91a695c70b45bb4a2dc48c0aeb7284e22))
* add custom livery editor with color picker and live car preview ([7294f6b](https://github.com/martynvdijke/heat/commit/7294f6b01481b642014fa3e7af654ba1f424ed46))
* add i18n support with EN/DE translations, language detection, and API ([3bb3e0c](https://github.com/martynvdijke/heat/commit/3bb3e0c39c2b2784d2c3b227bde1739e9ed7f638))
* add printable race report page with print-to-PDF support ([79f97f9](https://github.com/martynvdijke/heat/commit/79f97f90e67de1a5dca63db87d424e79634de204))
* add PWA support with manifest, service worker, and offline caching ([58dd4be](https://github.com/martynvdijke/heat/commit/58dd4be312fa29ea5d27857c51f8f300cee8b7eb))
* add race radio log with WebSocket broadcast and filtering ([72c3276](https://github.com/martynvdijke/heat/commit/72c327648994c9f07bdff380af42fbac4aad3212))

# [1.25.0](https://github.com/martynvdijke/heat/compare/v1.24.1...v1.25.0) (2026-05-15)


### Features

* add sharable driver stats links with email-friendly driver page ([cbe96c6](https://github.com/martynvdijke/heat/commit/cbe96c6618475989f0137f50ecd8f40bac825064))

## [1.24.1](https://github.com/martynvdijke/heat/compare/v1.24.0...v1.24.1) (2026-05-15)

# [1.24.0](https://github.com/martynvdijke/heat/compare/v1.23.2...v1.24.0) (2026-05-15)


### Bug Fixes

* fall back to racer_stats table when season filter returns empty ([341b307](https://github.com/martynvdijke/heat/commit/341b3079728cac7970ba006f444a4c0425dd0c39))


### Features

* remove features plan, add tests for deck builder, shared race control, AI difficulty ([fb66222](https://github.com/martynvdijke/heat/commit/fb66222e6faf3b1a25c2c96108e4faaa7ce993d6))

## [1.23.2](https://github.com/martynvdijke/heat/compare/v1.23.1...v1.23.2) (2026-05-15)


### Bug Fixes

* recover missing upload files on duplicate hash detection ([c4124a9](https://github.com/martynvdijke/heat/commit/c4124a986df379d2ca6578b08b1173e9c9124827))

## [1.23.1](https://github.com/martynvdijke/heat/compare/v1.23.0...v1.23.1) (2026-05-14)


### Bug Fixes

* **deps:** update all non-major dependencies ([da75567](https://github.com/martynvdijke/heat/commit/da7556741f951d33424d4e5d11dd036fe84a653a))

# [1.23.0](https://github.com/martynvdijke/heat/compare/v1.22.2...v1.23.0) (2026-05-13)


### Bug Fixes

* adapt swaggo/files v2 import to use embed FS via webdav handler ([111f0ae](https://github.com/martynvdijke/heat/commit/111f0ae40d1c3934a800397b396a5ff2123c812f))
* **deps:** update module github.com/swaggo/files to v2 ([332de35](https://github.com/martynvdijke/heat/commit/332de35dd1e06f8684a1d3839b2f1ca7f72a53ce))
* inline swagger handler into main.go to support go run main.go ([2a76317](https://github.com/martynvdijke/heat/commit/2a763171b14369229f29029c0362ed5d385132f0))
* resolve e2e test failures across all browsers ([37be585](https://github.com/martynvdijke/heat/commit/37be58563922c89cef645bd286662217364e4b01))
* trigger CI ([0fe1038](https://github.com/martynvdijke/heat/commit/0fe103844132ef37ac2d9be5d8ac8ccea6649be6))


### Features

* add game mechanics simulation, multi-user support, and race enhancements ([83d025c](https://github.com/martynvdijke/heat/commit/83d025cb4b06ce32f0db82298fb7485a26d83500))
* add UI presentation, deeper stats, multiplayer enhancements, and sound FX ([cbde244](https://github.com/martynvdijke/heat/commit/cbde244916dac690c7645a65cf07d9dc487af3be))

## [1.22.2](https://github.com/martynvdijke/heat/compare/v1.22.1...v1.22.2) (2026-05-11)


### Bug Fixes

* **deps:** update all non-major dependencies ([b65c8b5](https://github.com/martynvdijke/heat/commit/b65c8b5c16c58c00c86035b712c82730ec52bdab))
* **deps:** update module github.com/swaggo/files to v2 ([5298a85](https://github.com/martynvdijke/heat/commit/5298a85e512442f6bb1c50aa85fecf5f714e6725))

## [1.22.1](https://github.com/martynvdijke/heat/compare/v1.22.0...v1.22.1) (2026-05-10)


### Bug Fixes

* populate win distribution and driver performance sections from driver stats ([b93c2d6](https://github.com/martynvdijke/heat/commit/b93c2d6d955f1f97c168da068bbf291b2b06f66d))

# [1.22.0](https://github.com/martynvdijke/heat/compare/v1.21.0...v1.22.0) (2026-05-09)


### Features

* add workflow to auto-delete stale branches older than 90 days ([35546ff](https://github.com/martynvdijke/heat/commit/35546ff96acccffb01cf8f635c4545eb47e39d7b))

# [1.21.0](https://github.com/martynvdijke/heat/compare/v1.20.0...v1.21.0) (2026-05-09)


### Features

* auto-prune backups keeping last 7 by default ([ad26ebd](https://github.com/martynvdijke/heat/commit/ad26ebd6078a8f338a52efdea5d5cc1be05e6a42))

# [1.20.0](https://github.com/martynvdijke/heat/compare/v1.19.1...v1.20.0) (2026-05-09)


### Bug Fixes

* disable fullyParallel to prevent cross-browser DB state conflicts ([ff3becf](https://github.com/martynvdijke/heat/commit/ff3becfa8f001779cb0f5d20153d4530f36489e3))
* explicitly list browsers for Playwright install in CI ([ec506d8](https://github.com/martynvdijke/heat/commit/ec506d8d5ddbf58c3d1acb139373d281e515f24e))
* remove WebKit/Safari projects, only Chromium + Firefox (WebKit not available in CI) ([0c478b0](https://github.com/martynvdijke/heat/commit/0c478b015e7babde090b410feacabc544c351620))


### Features

* add multi-device Playwright projects and install all browsers in CI ([a627646](https://github.com/martynvdijke/heat/commit/a62764638cf65737956798fd0ba7273d69a82707))

## [1.19.1](https://github.com/martynvdijke/heat/compare/v1.19.0...v1.19.1) (2026-05-09)


### Bug Fixes

* **deps:** update module golang.org/x/crypto to v0.51.0 ([289caf7](https://github.com/martynvdijke/heat/commit/289caf77f555ae29ce4efcf13e35cbcb2a277d48))
* remove non-Chromium Playwright projects (CI only installs Chromium) ([38adb38](https://github.com/martynvdijke/heat/commit/38adb38cdb54199c95969a5bd8de9220e008fe5d))
* run go mod tidy after updating golang.org/x/crypto to v0.51.0 ([55c252f](https://github.com/martynvdijke/heat/commit/55c252fcb9540041b91c62d36caa855e6f34c474))

# [1.19.0](https://github.com/martynvdijke/heat/compare/v1.18.0...v1.19.0) (2026-05-09)


### Bug Fixes

* remove non-Chromium Playwright projects (CI only installs Chromium) ([288a0a8](https://github.com/martynvdijke/heat/commit/288a0a8419630c8af4080462e592ecda218f1a8a))
* win distribution chart labels missing racer names ([3caeba9](https://github.com/martynvdijke/heat/commit/3caeba9ab907939f2102f62a167b210a0061aff8))


### Features

* add multi-device Playwright projects (desktop, tablet, mobile) ([0052074](https://github.com/martynvdijke/heat/commit/0052074284eeca62518e78a57c576a027497b685))
* add responsive typography, hamburger nav, and multi-device layout ([6818125](https://github.com/martynvdijke/heat/commit/68181255fca3300e7df291e0eeb08e65e81a7387))


### Reverts

* remove Go vulnerability scan from CI and pre-push ([bcb3a7a](https://github.com/martynvdijke/heat/commit/bcb3a7a39082d9ee6598e19a3863ba027016387b))

# [1.18.0](https://github.com/martynvdijke/heat/compare/v1.17.0...v1.18.0) (2026-05-07)


### Bug Fixes

* update stats cards test for seasons stat replacing races ([69644ae](https://github.com/martynvdijke/heat/commit/69644ae0d4b9ae333cf9943d3cacd4640a462d54))


### Features

* add seasons page, remove race history from index, per-season stats filtering ([414b05f](https://github.com/martynvdijke/heat/commit/414b05fdaff040cefe76c091c11b433c5519b32b))

# [1.17.0](https://github.com/martynvdijke/heat/compare/v1.16.1...v1.17.0) (2026-05-06)


### Features

* integrate swaggo/swag + gin-swagger for auto-generated OpenAPI docs ([1be2452](https://github.com/martynvdijke/heat/commit/1be2452a890dad50f5cf98ada186c8c2b0ecab89))

## [1.16.1](https://github.com/martynvdijke/heat/compare/v1.16.0...v1.16.1) (2026-05-06)


### Bug Fixes

* race condition in upload handler, add try/catch to frontend uploads ([e89a6ef](https://github.com/martynvdijke/heat/commit/e89a6ef18b15cf3ec9cd9b5be83c6f31159ee112))

# [1.16.0](https://github.com/martynvdijke/heat/compare/v1.15.4...v1.16.0) (2026-05-06)


### Bug Fixes

* db connection deadlock in TakeRoundSnapshot, add season/round snapshot tests ([27f0c23](https://github.com/martynvdijke/heat/commit/27f0c23b69d8de1706282ada932d60beeeae3d00))
* remove archive tab from admin, add seasons tab, fix 500 errors by rebuilding ([00fbf43](https://github.com/martynvdijke/heat/commit/00fbf432611d937ac8e126812ca660c886c22c5c))
* v17 migration creates seasons table properly, fix round snapshot deadlock, add comprehensive tests ([4b5234b](https://github.com/martynvdijke/heat/commit/4b5234ba5a309561014bceeaddc854a729457310))


### Features

* add flag system with safety car/red flag overlay and per-driver flag toasts via WebSocket ([0447c7b](https://github.com/martynvdijke/heat/commit/0447c7bbe02a8d93d7ab96560fe41132704dfe0f))
* Bootstrap modal-based fullscreen flags, stat form cleanup, persistent chequered flag toggle, thorough tests ([cbffc7d](https://github.com/martynvdijke/heat/commit/cbffc7dddfcc07f35dc9c66cc728bc10ce1c7072))
* round snapshot system for points progression, fix trophies case bug, add round table to HTML ([2a7871d](https://github.com/martynvdijke/heat/commit/2a7871d153e342788c11b0269ba34df2aa46bb48))
* seasons with round snapshots, admin seasons/rounds tabs, fix flag modal colors, remove commentary ([35ac957](https://github.com/martynvdijke/heat/commit/35ac957960f8f51243c47a15acdee716708aea77))

## [1.15.4](https://github.com/martynvdijke/heat/compare/v1.15.3...v1.15.4) (2026-05-06)


### Bug Fixes

* rewrite image upload handler with MIME verification, dimension limits, and GIF passthrough ([db4385d](https://github.com/martynvdijke/heat/commit/db4385d98c33191bd3dd50abded20931c3a3e17d))

## [1.15.3](https://github.com/martynvdijke/heat/compare/v1.15.2...v1.15.3) (2026-05-06)


### Bug Fixes

* move email/analytics/backup tab panes inside .tab-content to fix white screen ([2047d38](https://github.com/martynvdijke/heat/commit/2047d386330f241eafe0aa136d5c2a253c34a33d))

## [1.15.2](https://github.com/martynvdijke/heat/compare/v1.15.1...v1.15.2) (2026-05-05)


### Bug Fixes

* revert settings sidebar grouping and add profile picture upload tests ([56819ca](https://github.com/martynvdijke/heat/commit/56819ca9735bb11f5dbd0d93008c7dc909b3f327))

## [1.15.1](https://github.com/martynvdijke/heat/compare/v1.15.0...v1.15.1) (2026-05-05)


### Bug Fixes

* remove strict Origin/Host check from CSRF middleware to fix admin POST requests behind reverse proxy ([c015a01](https://github.com/martynvdijke/heat/commit/c015a01cacb6c97cb82b6dcc78b690a7e560a1e2))

# [1.15.0](https://github.com/martynvdijke/heat/compare/v1.14.0...v1.15.0) (2026-05-05)


### Features

* add DNS (did not start) tracking to racer stats and enhance race deletion ([f17bea2](https://github.com/martynvdijke/heat/commit/f17bea25ecb4c72a2b861a476b85cc9534f09729))

# [1.14.0](https://github.com/martynvdijke/heat/compare/v1.13.4...v1.14.0) (2026-05-05)


### Features

* add periodic database backup with admin panel management ([e07ea47](https://github.com/martynvdijke/heat/commit/e07ea4736f390f7ebb65fa2fb96634471144211c))
* replace prompt-based stats editing with proper modal in admin panel ([923e7d8](https://github.com/martynvdijke/heat/commit/923e7d89b8bd046f899417c684180bc0fc083e09))

## [1.13.4](https://github.com/martynvdijke/heat/compare/v1.13.3...v1.13.4) (2026-05-05)


### Bug Fixes

* remove MaxBytesReader breaking multipart uploads, fix null stats crash, fix CSP source maps ([1f65ca3](https://github.com/martynvdijke/heat/commit/1f65ca30938d5b51cf738ebb7c55526e3589a3b3))

## [1.13.3](https://github.com/martynvdijke/heat/compare/v1.13.2...v1.13.3) (2026-05-05)


### Bug Fixes

* add error handling to upload functions matching traces pattern ([4249b86](https://github.com/martynvdijke/heat/commit/4249b865692a708e2953f263508c66afbe7d3aec))
* trigger ci ([f9016da](https://github.com/martynvdijke/heat/commit/f9016dade3b725b7d52041c8f72012cded2bf3f6))

## [1.13.2](https://github.com/martynvdijke/heat/compare/v1.13.1...v1.13.2) (2026-05-05)


### Bug Fixes

* support legacy SHA-256 password login and remove unused chat page ([0adf194](https://github.com/martynvdijke/heat/commit/0adf194c3e938301978db1b8ee493392ebfaceea))

## [1.13.1](https://github.com/martynvdijke/heat/compare/v1.13.0...v1.13.1) (2026-05-05)


### Bug Fixes

* protect SessionStore from concurrent map access and fix logout cookie Secure flag ([f575e0c](https://github.com/martynvdijke/heat/commit/f575e0cb1f3fa69cc7dc8141c896494021a0f6ad))

# [1.13.0](https://github.com/martynvdijke/heat/compare/v1.12.4...v1.13.0) (2026-05-05)


### Bug Fixes

* Playwright test failures from CSP and stale DB ([b2cd28d](https://github.com/martynvdijke/heat/commit/b2cd28d4006da7c2a1bbedd3d34b1bdeb363994e))


### Features

* implement security hardening (Phase 1 + Phase 4) ([306156d](https://github.com/martynvdijke/heat/commit/306156d434de83ba36b0ed7b9f84375381943539))

## [1.12.4](https://github.com/martynvdijke/heat/compare/v1.12.3...v1.12.4) (2026-05-05)


### Bug Fixes

* bump hardcoded version from 1.11.1 to 1.12.2 to match releases ([6a23ef0](https://github.com/martynvdijke/heat/commit/6a23ef0ff1e707fc011723999428bfefc514e2b4))

## [1.12.3](https://github.com/martynvdijke/heat/compare/v1.12.2...v1.12.3) (2026-05-05)

## [1.12.2](https://github.com/martynvdijke/heat/compare/v1.12.1...v1.12.2) (2026-05-05)


### Bug Fixes

* **deps:** update module golang.org/x/crypto to v0.50.0 ([1a31540](https://github.com/martynvdijke/heat/commit/1a315400383dd26298589de038a9ad2e280e1301))

## [1.12.1](https://github.com/martynvdijke/heat/compare/v1.12.0...v1.12.1) (2026-05-04)

# [1.12.0](https://github.com/martynvdijke/heat/compare/v1.11.1...v1.12.0) (2026-05-04)


### Features

* add Umami analytics and fix security issues with Denver ([c28fa66](https://github.com/martynvdijke/heat/commit/c28fa66d755aa93cf5a2f42310b18c6910e2ce2a))

## [1.11.1](https://github.com/martynvdijke/heat/compare/v1.11.0...v1.11.1) (2026-05-04)


### Bug Fixes

* remove export hack, drop uploads tab, fix stats, add regression tests ([6836691](https://github.com/martynvdijke/heat/commit/68366913b335981ae44e4016526473ec79d16234))

# [1.11.0](https://github.com/martynvdijke/heat/compare/v1.10.0...v1.11.0) (2026-05-04)


### Bug Fixes

* resolve database test isolation and compile errors with Denver ([d4cf3d6](https://github.com/martynvdijke/heat/commit/d4cf3d62c2d31f51c1972eed61f4c30e3380519f))


### Features

* add email support for race result overview emails ([9069be4](https://github.com/martynvdijke/heat/commit/9069be4851a7f226eb44bee714155a9f13036cfd))

# [1.10.0](https://github.com/martynvdijke/heat/compare/v1.9.1...v1.10.0) (2026-05-04)


### Features

* add uploads tab with file picker and gallery to admin panel ([ab1ca5f](https://github.com/martynvdijke/heat/commit/ab1ca5f64d9a7f0d3db97d26a3024bf1f74d84a7))

## [1.9.1](https://github.com/martynvdijke/heat/compare/v1.9.0...v1.9.1) (2026-05-04)


### Bug Fixes

* separate AI tab, fix racer creation, fix default images and GeoJSON loading ([9e7f31d](https://github.com/martynvdijke/heat/commit/9e7f31d684d18c98156c893291249cd86f7e9cc0))

# [1.9.0](https://github.com/martynvdijke/heat/compare/v1.8.0...v1.9.0) (2026-05-03)


### Features

* add AI image-based track extraction with configurable endpoint ([1876054](https://github.com/martynvdijke/heat/commit/18760547428ee8490178b3e2fbc8955c9f806646))

# [1.8.0](https://github.com/martynvdijke/heat/compare/v1.7.1...v1.8.0) (2026-05-03)


### Features

* add file upload with hashing, resizing, and thumbnailing ([ca135fd](https://github.com/martynvdijke/heat/commit/ca135fdddd5522579b206926030fcc992916c664))

## [1.7.1](https://github.com/martynvdijke/heat/compare/v1.7.0...v1.7.1) (2026-05-03)


### Bug Fixes

* bootstrap load order, login redirect, and add admin CRUD tests ([c76d412](https://github.com/martynvdijke/heat/commit/c76d412d1d82adc35cf8e21e66ecdc5865817591))

# [1.7.0](https://github.com/martynvdijke/heat/compare/v1.6.1...v1.7.0) (2026-05-03)


### Bug Fixes

* build TypeScript before Playwright tests in CI with Denver ([cd7c338](https://github.com/martynvdijke/heat/commit/cd7c3387ad77678e0ee208f33e3496cf7461a9b1))
* preserve ts/ subdirectory in Docker build with Denver ([e47b4f3](https://github.com/martynvdijke/heat/commit/e47b4f3f24815a582269a29276610ae8ef0a9195))
* resolve Gin static route conflict with Denver ([f7adbc3](https://github.com/martynvdijke/heat/commit/f7adbc39c54e789d926e338b0383e6cadd9a6692))
* strip TypeScript export declarations from compiled JS with Denver ([9beb9e6](https://github.com/martynvdijke/heat/commit/9beb9e69df49fe502c49c42b960005c472104d8d))


### Features

* migrate to Gin framework and TypeScript with Denver ([8f5ae33](https://github.com/martynvdijke/heat/commit/8f5ae335931c3e117c6a59ac4dedd48a9a4323ba))

## [1.6.1](https://github.com/martynvdijke/heat/compare/v1.6.0...v1.6.1) (2026-05-03)


### Bug Fixes

* ui fixes and dev improvment ([9a3f2b2](https://github.com/martynvdijke/heat/commit/9a3f2b2123e5ba19378900ba61c7bc0a912ec8a1))

# [1.6.0](https://github.com/martynvdijke/heat/compare/v1.5.0...v1.6.0) (2026-05-02)


### Features

* add Gotify notification support ([6f0cf3c](https://github.com/martynvdijke/heat/commit/6f0cf3cbd5379e5b7f756fedfd5ea74cfda1262f))

# [1.5.0](https://github.com/martynvdijke/heat/compare/v1.4.0...v1.5.0) (2026-05-01)


### Features

* add admin stats, protect controller, remove chat, improve UI ([b8de591](https://github.com/martynvdijke/heat/commit/b8de591d51d8e93df8fd3fb9a4d0a2fd6fd0326b))

# [1.4.0](https://github.com/martynvdijke/heat/compare/v1.3.1...v1.4.0) (2026-05-01)


### Features

* **heat:** add season stats, trophy room, mobile controller, live chat, and one-off race tracking ([9802cba](https://github.com/martynvdijke/heat/commit/9802cbaff0f5b46f7a31d84be86d877689a8c8c0))

## [1.3.1](https://github.com/martynvdijke/heat/compare/v1.3.0...v1.3.1) (2026-05-01)


### Bug Fixes

* **deps:** update module github.com/mattn/go-sqlite3 to v1.14.44 ([#15](https://github.com/martynvdijke/heat/issues/15)) ([6166469](https://github.com/martynvdijke/heat/commit/61664693a2bb44cddfc6602bc68d834bf7c19825))

# [1.3.0](https://github.com/martynvdijke/heat/compare/v1.2.3...v1.3.0) (2026-04-30)


### Features

* add season stats, trophy room, mobile controller, live chat, and one-off race tracking ([5997ca8](https://github.com/martynvdijke/heat/commit/5997ca8a95fd78d64ef06004c48a3bfca03d8d55))

## [1.2.3](https://github.com/martynvdijke/heat/compare/v1.2.2...v1.2.3) (2026-04-29)

## [1.2.2](https://github.com/martynvdijke/heat/compare/v1.2.1...v1.2.2) (2026-04-22)


### Bug Fixes

* db persistense ([f2ee7c5](https://github.com/martynvdijke/heat/commit/f2ee7c55dcb18894c87567ff3fa8a55694c98907))

## [1.2.1](https://github.com/martynvdijke/heat/compare/v1.2.0...v1.2.1) (2026-04-21)


### Bug Fixes

* add all racers colors ([3d37d99](https://github.com/martynvdijke/heat/commit/3d37d99457fc6fb1d3e6507847454efc9369f883))

# [1.2.0](https://github.com/martynvdijke/heat/compare/v1.1.2...v1.2.0) (2026-04-20)


### Features

* add in experimental ai image background and opther ui fixes ([8eaf0d1](https://github.com/martynvdijke/heat/commit/8eaf0d15774cd81bf9a6d7539581d4eeba904709))

## [1.1.2](https://github.com/martynvdijke/heat/compare/v1.1.1...v1.1.2) (2026-04-20)

## [1.1.1](https://github.com/martynvdijke/heat/compare/v1.1.0...v1.1.1) (2026-04-19)

# [1.1.0](https://github.com/martynvdijke/heat/compare/v1.0.5...v1.1.0) (2026-04-18)


### Bug Fixes

* **ci:** premissions ([f5b8ba7](https://github.com/martynvdijke/heat/commit/f5b8ba7617ab845bd38abb761a4366ef70417dd6))
* **docs:** update docs ([43fb907](https://github.com/martynvdijke/heat/commit/43fb907b0a70e6a1940dbb7b2a3c527fcd3377a8))
* gitignore ([5ce3f50](https://github.com/martynvdijke/heat/commit/5ce3f50c5c5602384b243adfb2e60aec2d5a3f95))
* more fixes ([3f6ab72](https://github.com/martynvdijke/heat/commit/3f6ab7272dbe1dc0d681ffd4e262c9e11faee75d))
* tests ([b9cb3fd](https://github.com/martynvdijke/heat/commit/b9cb3fdeace6b7aa5d923935e4f4dc9eddd9a503))
* update ci workflow to also push ([421d1e8](https://github.com/martynvdijke/heat/commit/421d1e882b0f09ea47e7d9009fcbdb7676d75ba3))


### Features

* add go unit tests and integrate into CI workflow ([842c072](https://github.com/martynvdijke/heat/commit/842c07289509baf3a25da6509632619333ba1f6a))
* add real-time circuit map support using Monza GeoJSON ([6f56049](https://github.com/martynvdijke/heat/commit/6f560499da13a04bdab4e9bdbb92c84ef44eec1f))
* add sortable leaderboard, track selector, race history, driver stats, and commentary quotes ([8699c6b](https://github.com/martynvdijke/heat/commit/8699c6b3ce63af824eb65b48c42f04749ea0b33c))
* add Swagger/OpenAPI documentation for the API ([2af0618](https://github.com/martynvdijke/heat/commit/2af0618e77649615ff7a1712de6700e13a9bbb71))
* implement real-time racer positions via websockets with gap display ([43be897](https://github.com/martynvdijke/heat/commit/43be8976caa8cd603ac501278550c1d772a12e54))
* **test:** add in taskfile ([1772e55](https://github.com/martynvdijke/heat/commit/1772e551b3cf1f0aee70768766591a60a418e133))
* **test:** add playwitgh tests ([1142d0a](https://github.com/martynvdijke/heat/commit/1142d0aebc66bc330da9fd24453ddd966d21ace3))

## [1.0.5](https://github.com/martynvdijke/heat/compare/v1.0.4...v1.0.5) (2026-04-17)

## [1.0.4](https://github.com/martynvdijke/heat/compare/v1.0.3...v1.0.4) (2026-04-15)

## [1.0.3](https://github.com/martynvdijke/heat/compare/v1.0.2...v1.0.3) (2026-04-15)


### Bug Fixes

* frontend api premissions ([564372d](https://github.com/martynvdijke/heat/commit/564372df060d5acb3be077cdf49b14ad49bf54ad))

## [1.0.2](https://github.com/martynvdijke/heat/compare/v1.0.1...v1.0.2) (2026-04-15)


### Bug Fixes

* use /db ([cd6067a](https://github.com/martynvdijke/heat/commit/cd6067a760038d110accfe63cb4046dabd1b9044))

## [1.0.1](https://github.com/martynvdijke/heat/compare/v1.0.0...v1.0.1) (2026-04-15)

# 1.0.0 (2026-04-15)


### Bug Fixes

* docs ([10d2da2](https://github.com/martynvdijke/heat/commit/10d2da22ec1514e2d62ffdb07ba524c9bbef8c97))
* renovate ([0ac988d](https://github.com/martynvdijke/heat/commit/0ac988d1bf202a7e801c90f54fe62dbfff4f6344))
* severla fixes ([b207581](https://github.com/martynvdijke/heat/commit/b207581a0a66f8b7610cf8040719cfc6048a4205))


### Features

* add go backend ([1336a0e](https://github.com/martynvdijke/heat/commit/1336a0e223cd3d98866fa4ce4ac92eb4a7563819))
* inital setup ([7baaf02](https://github.com/martynvdijke/heat/commit/7baaf022e7888c05dd5b9ddb9407639d9effb2d1))
* initial commit ([7aa3086](https://github.com/martynvdijke/heat/commit/7aa3086928f1da9e28b4a9f1eda035ac8e1f6f47))
* more fixes working version ([831f34a](https://github.com/martynvdijke/heat/commit/831f34a9dae3e98b77ccdc6afbb1bdbe4b524155))
