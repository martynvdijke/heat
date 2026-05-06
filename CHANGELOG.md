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
