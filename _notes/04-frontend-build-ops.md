# 04 — Frontend / Build / Ops / Docs

## Версия
- `config/version` = **1.5.10-beta7** (нота `docs/releases/v1.5.10-beta7.md`, 2026-06-25)
- frontend `package.json` тоже 1.5.10-beta7
- `[Unreleased]` пуст во всех CHANGELOG-{EN,RU,ZH}.md

## frontend/ — Vue 3 SPA

### Стек
- **Vue 3.5**, **Vuetify 4**, **Vite 8** (rolldown-vite), `vite-plugin-vuetify 2.1`
- **TypeScript 6** + `vue-tsc` (`vue-tsc --noEmit && vite build`)
- **Vue Router 5**, **Pinia 3**, **Vue I18n 11** (`legacy:false`)
- Charts: chart.js 4 + vue-chartjs 5
- Иконки: `@mdi/js` (tree-shake через `scripts/gen-mdi-icons.cjs`) + `lucide-vue-next`
- axios, notivue (тосты), moment, qrcode.vue, clipboard, vue3-persian-datetime-picker, yaml
- Тесты/линт: Vitest 4, Playwright 1.60 (+axe-core a11y), ESLint 10 flat

### Build (`vite.config.mts`)
- Manual chunks: `vendor-vue` / `vendor-vuetify` / `vendor-http`
- Ассеты с префиксами `app-`/`chunk-`/`style-` — **никогда с `_`** (Go embed дропает `_`/`.` → «пустая панель»)
- `optimizeDeps.exclude`: vuetify(+components/directives), vue-i18n (баг pre-bundle Rolldown), обход через `__VUE_I18N_FULL_INSTALL__:true`
- Dev: `port 3000`, proxy `/app/api → :2095`; alias `@`→`./src`
- HMR off при `SUI_E2E=1`; `{{ .BASE_URL }}` инжектится Go на embed

### Структура `frontend/src/`
```
components/
  nexus/        # НОВЫЙ дефолтный UI: data/ drawers/ overview/ primitives/
  presets/ protocols/ services/ settings/ tls/ transports/ tiles/  # classic
layouts/
  AuthenticatedHost.vue        # auth-shell для protected
  default/                     # AppBar/Default/Drawer/View (classic)
  nexus/                       # NexusShell/Sidebar/Topbar/ServerStatus/nexusMenu
  modals/
locales/        # en fa vi zhcn zhtw ru + index.ts (lazy) + tests
plugins/        # axios/httputil, websocket, base-url, vuetify
router/index.ts
store/
  modules/data.ts       # Pinia: config/inbounds/clients/... ; поллит /api/load каждые 10s
  modules/realtime.ts   # WS-сэмплы
  ws.ts
styles/settings.scss + nexus/_tokens.scss   # Emerald+Dracula, dark/light
types/  uiMode/  views/
views/    # 17 страниц + per-entity *NexusList.vue + nexus/Overview.vue + paidsub/
```

### Router
- `createWebHistory(getBaseUrl())`
- protected — дети `AuthenticatedHost.vue` с `meta.requiresAuth`
- 17 страниц: home, inbounds, clients, outbounds, services, endpoints, rules, tls, dns, admins, telegram, audit, migrate-xui, paid-subscriptions, settings, donations; `/basics`→`/settings?tab=basics`
- Session cookie **HttpOnly** (`api/session.go`)
- **Preload-error self-heal**: `vite:preloadError`/`router.onError` → ChunkLoadError после апгрейда → **1 reload/сессия** (`sessionStorage sui:preload-error-reload`)
- Поллинг `loadData()` каждые 10s, дельта через `?lu=<lastLoad>`

### Store (Pinia `Data`)
State: lastLoad, reloadItems, sub URIs, enableTraffic, onlines{inbound,outbound,user,failover}, config, inbounds, outbounds, services, endpoints, clients, tlsConfigs.
Actions: `loadData()` (авто-тосты из lastLog), `save(object, action, data)` — **единый `api/save`** для всего CRUD (server-side switch по object+action), `checkClientName`/`checkTag` (guard дублей).

### UI mode
- `UI_MODES=['classic','nexus']`, **`DEFAULT_UI_MODE='nexus'`**
- localStorage `sui:ui:mode`; `isNexusEnabled()` ← `VITE_ENABLE_NEXUS` (default true)
- палитры Emerald/Dracula в `styles/nexus/_tokens.scss`

### i18n
- 6 локалей: en/fa/vi/zhcn/zhtw/ru, lazy-load, default `en`, localStorage `locale`
- Тесты parity (`localeParity.test.ts`)
- ⚠️ Установщик (bash) — только EN/RU/ZH; FA/VI лишь в панели

---

## Build & Ops

### install.sh (~26 KB bash)
- Мультиязычный EN/RU/ZH (`$SUI_LANG` → `/etc/s-ui/lang` → prompt → default EN при `curl|bash`)
- Опц. позиционный арг = версия-тег
- Deps per distro (apt/dnf/yum/pacman): wget curl tar tzdata
- **Два one-time секрета** на первом install (показ один раз):
  - `SUI_SECRETBOX_KEY` → `/etc/s-ui/secretbox.env` (0600) + systemd drop-in `10-secretbox-env.conf`
  - `SUI_COOKIE_KEY` → HMAC session cookie
- Upgrade: стоп sing-box, сохранение settings
- Случайные admin-креды на fresh install; checksum проверка загрузки
- Порты 2095/2096, пути `/app/`, `/sub/`
- ⚠️ TZ **не** в install.sh — `Europe/Moscow` в `service/setting.go`

### s-ui.sh (63 KB) — управляющий скрипт
Меню EN/RU/ZH: Install/Update/Custom version/Uninstall / Reset-Set-Show admin / Reset-Set-Show panel / Clear domain / Start-Stop-Restart-Status-Log / autostart / **BBR** (bbr↔cubic с проверкой) / SSL (acme.sh) / Cloudflare SSL / Language / Generate `SUI_COOKIE_KEY`.

### Makefile (PowerShell-driven audit gate)
`make audit`: build, vet, test-go, test-go-race, cover, gosec, vuln(govulncheck), lint-go(staticcheck+golangci-lint), fe-lint, fe-build, test-fe(vitest). `fe-typecheck` и `e2e` — **skipped**.

### Dockerfile (multi-stage, digest-pinned, multi-arch)
- `front-builder`: node:alpine (pinned) → `npm ci && npm run build`
- `backend-builder`: golang:1.26.4-alpine, `CGO_ENABLED=1`, gcc/musl
- **Pinned cronet-go .so** (dlopen root sing-box): `CRONET_GO_VERSION`, asset `v148.0.7778.96-1`, per-arch sha256 + `sha256sum -c`
- Финал build tags: `with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale` → бинарь `sui`
- Final: alpine (pinned), `ENV TZ=Europe/Moscow`, bash tzdata ca-certificates nftables, `ENTRYPOINT ["./entrypoint.sh"]`

### build.sh (локальная сборка)
```sh
cd frontend && npm i && npm run build && cd ..
mkdir -p web/html && rm -fr web/html/* && cp -R frontend/dist/* web/html/
BUILD_TAGS="with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_musl,badlinkname,tfogo_checklinkname0,with_tailscale"
# ARTIFACT_PLATFORM инжектится в config.ArtifactPlatform (для self-update)
go build -ldflags "$LDFLAGS" -tags "$BUILD_TAGS" -o sui main.go
```
⚠️ Набор тегов **отличается** от Dockerfile (тут `with_musl,badlinkname,tfogo_checklinkname0` вместо `with_purego`).

---

## docs/
- `docs/releases/` — поштучные ноты релизов (`vX.Y.Z.md`, `vX.Y.Z-betaN.md`)
- `.github/RELEASE_NOTES_*.md` — дублирующие ноты по версиям
- CHANGELOG-{EN,RU,ZH}.md — полная история (RU самый большой, ~196 KB)
- `docs/sing-box-tracker-revalidation.md` — про пин версии sing-box в трекерах
- Скриншоты/лого в `docs/`

## Замечания на будущую работу
- **Единая точка CRUD** — `api/save` (server switch по object+action) и Pinia `save()`. Менять модель данных → трогать обе стороны.
- **Nexus по умолчанию** — новые UI-фичи ждут в `components/nexus/` + `*NexusList.vue`; classic сохраняем для совместимости.
- **Go embed префиксы** — любые новые ассеты не должны начинаться с `_`/`.`.
- **Версия sing-box запинена** (`v1.13.13`) в `core/tracker_policy.go` — при апгрейде sing-box перепроверять инварианты трекеров.
- **Секреты через secretbox** — новые чувствительные settings шифровать (`service/secret_settings.go`, `ResealSecretSettings`).
- **Build tags различаются** Docker vs build.sh — держать в уме при воспроизведении билда.
