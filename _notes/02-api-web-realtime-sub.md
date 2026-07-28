# 02 — API / Web / Realtime / Sub

## api/ (17 не-тестовых файлов; всего 58 с тестами)

Две параллельные ветки хендлеров поверх единого `ApiService`.

### ApiService (`apiService.go`)
Композит ~20 сервисов (User/Setting/Config/Client/Tls/Inbound/Outbound/Endpoint/Services/Panel/Stats/Server/Audit/Observability/Telegram/Version/PanelUpdate/Doctor). `WithRuntime`/`NewApiService` протягивают общий `*service.Runtime`. Конверт ответа `Msg{Success, Msg, Obj}`, HTTP 200, текст ошибки скраблится через `redact.String` (сырое — только в лог).

### APIHandler — session-based (браузер) `apiHandler.go`
Монтируется на `{base}/api`. Middleware группы:
1. `checkLogin` (кроме `.../api/login` и `.../api/logout` — exact-match)
2. `csrfMiddleware`

POST (auth+CSRF): `login`, `changePass`, `addAdmin`, `deleteAdmin`, `save`, `restartApp`, `restartSb`, `linkConvert`, `subConvert`, `importdb`, `addToken`/`deleteToken`/`setTokenEnabled` (перезагружают токены), `logoutAllAdmins`, + import-xui.
GET: `/csrf`, `/load`, `/inbounds|outbounds|endpoints|services|tls|clients|config`, `/users`, `/settings`, `/stats`, `/stats/traffic`, `/status`, `/onlines`, `/logs`, `/changes`, `/keypairs`, `/getdb`, `/tokens`, `/singbox-config`, `/checkOutbound`, `/failover-status`, `/version`.
Подгруппы под тем же gate:
- `/doctor/{run,client}` — scope `admin|read|write|observability`; SSRF-валидация таргета
- `/security/audit` — только admin scope + rate-limit
- `/update/{status,check,apply}` — step-up re-auth для `apply`
- `/telegram/{test,backup,backup/run}` — admin scope; ручной backup rate-limited
- `/realtime/{ws-token,ws}` — см. realtime
- `/ip-monitor/:client` (GET) / `/:client/clear` (POST)
- `/observability/{history,core-history}` — observability или admin
- `paidsub.RegisterRoutes(g, …)` — модуль платных подписок

`save_dedup.go` — server-side защита от дублей-создания (mutex): для `new`/`addbulk` на clients/inbounds/outbounds/endpoints/services/tls. Claim блокирует одинаковый ключ на время выполнения + 3 c после; провал → `release()`.

### APIv2Handler — token-based `apiV2Handler.go`
Монтируется на `{base}/apiv2`. Middleware: `checkToken`. Аутентификация:
- `Authorization: Bearer <token>` (предпочтительно), ИЛИ
- legacy `Token: <token>` — **deprecated, sunset `Sat, 15 Aug 2026 GMT`**. После — 401 + Deprecation + Sunset. Пока живо — каждый вызов пишет `legacy_token_header_used` warn-аудит.

Токены в памяти (`tokensMu.RWMutex`, ключ = SHA-256 токена → `TokenInMemory{ID, TokenHash, Prefix, Scope, Enabled, Expiry, Username}`). `ReloadTokens()` на старте и после каждого create/delete/enable в v1.

**Scope enforcement** — `apiV2ActionScopes` гейтит каждый `:getAction`/`:postAction`. Браузерные сессии (без scope токена) пропускаются. Не-браузерные вызовы должны матчить один из allowed scopes; `admin` всегда добавлен.

### Прочие файлы api/
- `apiFoundation.go` / `apiFailover.go` / `apiUpdate.go` — вспомогательные ветки
- `audit.go`, `csrf.go`, `rateLimit.go`, `realtime.go`, `session.go` — соответствующие подсистемы
- `import_xui*.go` — импорт из x-ui (routes, rate-limit, rollback, plan-stream)
- `doctor.go`, `ip_monitor.go`, `save_dedup.go`, `utils.go` (`checkLogin`)

### Импорт x-ui (`api/import_xui*.go`)
Планирование стримится (plan-stream), rate-limit, deadline, rollback + realtime-события отката. Cap 64 MiB, SQLite magic-check, staging.

---

## middleware/ (4 файла)
Gin-middleware: логирование запросов, domain/redirect, доменная валидация, безопасность (заголовки). (Основная auth/CSRF-логика — в `api/`, не тут.)

---

## web/ (6 файлов, `web.go`)
- HTTP-сервер панели (Gin). Встроенный фронтенд через `//go:embed web/html`.
- **Таймауты**: read/write/header/idle выставлены.
- **TLS MinVersion 1.2**.
- Security-заголовки включены.
- Защита embed: файлы с `_`/`.` в начале Go-embed молча дропает → фронтенд собирается с префиксами `app-`/`chunk-`/`style-` (защита от «пустой панели»).
- `{{ .BASE_URL }}` инжектится в `index.html` на этапе embed.
- Fallback listen: если сохранённый listen IP исчез с хоста — fallback остаётся ограниченным (не расширяет экспозицию молча), пишет audit-fallback.

---

## realtime/ (5 файлов)
- WebSocket через `coder/websocket`.
- **Hardened**: single-use токены (`/realtime/ws-token` выдаёт, `/realtime/ws` принимает), Origin-проверки.
- Broadcast: маршалит событие один раз на всех подписчиков (перф).
- Топики: `TopicCoreState` и др. (состояние ядра, стата, failover, онлайны).
- Хук инвалидации токенов (`service/session_rotation.go`).

---

## sub/ (18 файлов) — отдача подписок
- Форматы: **link**, **json**, **clash** (+ info-заголовки трафика/срока).
- **Per-client sub-secrets**: URL с секретом на клиента. Legacy name-based URL работают пока `subSecretRequired=false`.
- Rate-limit по IP (`subRateLimitPerIP`, применяется в течение ~1 мин после сохранения).
- gzip, короткий output-кэш успешных ответов, no-store заголовки, санитизация заголовков.
- Отдельный HTTP-сервер (порт 2096, путь `/sub/`) со своими таймаутами.
- Опционально монтируется **лэндинг «Личный кабинет»** на `/cabinet/{subid}` через пакет `subpage/` — см. [`05-subpage-landing.md`](05-subpage-landing.md). По умолчанию выключен флагом `subPageEnabled`.
