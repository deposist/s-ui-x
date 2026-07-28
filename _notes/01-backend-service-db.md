# 01 — Backend: service/ + database/

## database/ (GORM + SQLite, WAL)

### Файлы
| Файл | Назначение |
|---|---|
| `db.go` | Инициализация БД (`OpenDB`, `InitDB`), `GetDB()`, пул соединений, guard'ы индексов/FK, drop deprecated tables, singleton `*gorm.DB` под `dbMu` RWMutex |
| `model/model.go` | Модели: `Setting`, `Tls`, `User`, `Client`, `ClientIP`, `Stats`, `Changes`, `AuditEvent`, `Tokens` |
| `model/endpoints.go` | `Endpoint` — динамические поля в `Options json.RawMessage`, merge на Marshal |
| `model/inbounds.go` | `Inbound` — FK на `tls`; `Addrs`, `OutJson` вынесены; остальное в `Options`; `MarshalFull()` для admin |
| `model/outbounds.go` | `Outbound` — type+tag+`Options`; `warp` нормализуется в `wireguard` |
| `model/services.go` | `Service` — sing-box rule/services; FK на `tls`, options JSON |
| `backup.go` | Экспорт БД (`GetDb`/`PrepareDbBackup`/`ImportDB`) с пагинацией; детект x-ui (`isXUIDatabase`); rollback при фейле; SIGHUP-рестарт |
| `bulk.go` | `SafeSQLiteBatchSize`, `CreateInBatchesSafe`, `SaveInBatchesSafe` (лимит SQLite `SQLITE_MAX_VARIABLE_NUMBER=999`) |
| `adapt.go` | `AdaptToCurrentVersion` — идемпотентная нормализация: rehash plaintext→bcrypt, восстановление индексов, bump `settings.version` |
| `initial_admin.go` | Пишет `initial-admin.txt` (chmod 600) со случайным 24-символьным паролем на первом запуске |
| `reset_hooks.go` | Реестр хуков (`RegisterResetHook`/`ResetCaches`) — чистит in-memory кэши после импорта |
| `importxui/` (23 файла) | Одноразовый импортёр из 3x-ui SQLite (read-only): inbounds, wg endpoints, Reality TLS, clients в одной транзакции; dry-run; детект диалектов (`dialect_3xui_mhsanaei.go`); `Report` |

### Схема (таблицы, ключевые поля)
- **settings** — `(id, key UNIQUE, value)` — key/value конфиг
- **tls** — `(id, name, server JSON, client JSON)`; `id=0` — sentinel `__none__` (`ensureNoTLSRow()`)
- **inbounds** — `(id, type, tag UNIQUE, tls_id FK, addrs JSON, out_json JSON, options JSON)`
- **outbounds** — `(id, type, tag UNIQUE, options JSON)`; default `direct` сидится
- **services** — `(id, type, tag UNIQUE, tls_id FK, options JSON)`
- **endpoints** — `(id, type, tag UNIQUE, options JSON, ext JSON)` — WG/warp
- **users** — `(id, username, password[bcrypt], lastLogins, forcePasswordReset)`
- **tokens** — `(id, desc, tokenHash INDEX, tokenPrefix, scope, enabled, expiry, timestamps, lastUsedAt, lastUsedIp, userId FK)`
- **clients** — `(id, enable, name, subSecret INDEX, config JSON, inbounds JSON, links JSON, volume, expiry, up/down, group, limitIp, ipLimitMode, lastOnline, lastIpCount, delayStart, autoReset, resetDays, nextReset, totalUp/Down)`
- **client_ips** — `(id, clientName, ip, ipHash UNIQUE-with-clientName, ipDisplay, firstSeen, lastSeen INDEX)` — хэш-ключ, PII-light
- **stats** — `(id, dateTime, resource, tag, direction bool, traffic int64)` — тайм-серия трафика
- **changes** — `(id, dateTime, actor, key, action, obj JSON)` — аудит изменений admin
- **audit_events** — `(id, dateTime, actor, event, resource, severity, ip, userAgent, details JSON)` + индексы

### Миграции — два слоя
1. **`cmd/migration/main.go`** — последовательные `to1_1 … to1_7`, привязка по semver к `settings.version`, каждая в транзакции.
2. **`database.InitDB()`** — `AutoMigrate` всех моделей + ручной SQL для уникальных индексов, drop deprecated, sentinel TLS.
- **`AdaptToCurrentVersion()`** на каждом старте — rehash паролей, bump версии.
- **Импорт .db**: `ImportDB()` → staging → integrity check → **отклоняет 3x-ui БД** → rename живой в `.backup` → swap → `MigrateDb()`+`InitDB()`+`ResetCaches()` → SIGHUP.

### Инициализация
`InitDB(dbPath)` → `OpenDB` (WAL, `_busy_timeout=10000`, `_foreign_keys=on`, малый пул) → seed default `direct` → `AutoMigrate` → drop deprecated → sentinel `tls(id=0)` → custom индексы → `initUser` (admin + random 24-char в `initial-admin.txt`) → `AdaptToCurrentVersion()`.

---

## service/ (бизнес-логика, 122 файла)

### Ключевые сервисы
| Файл | Ответственность |
|---|---|
| `runtime.go` | `Runtime` — thread-safe holder `core.Core`, restartManager, auditWriter, telegramNotifier. Мост к живому ядру без import cycles |
| `server.go` | `ServerService` — статус системы (cpu/mem/disk/io/net/swap/sing-box/db) через gopsutil |
| `services.go` | `ServicesService` — CRUD `model.Service`; эмитит `entityCoreChange` |
| `inbounds.go` | `InboundService` (embeds `ClientService`) — get/save, генерация per-inbound links, регенерация подписок |
| `outbounds.go` | `OutboundService` — CRUD; сборка `failover` через `failover_assembly.go` |
| `endpoints.go` | `EndpointService` (embeds `WarpService`) — CRUD (wg/warp) |
| `client.go` | `ClientService` — CRUD, traffic/expiry (`clientTrafficOverLimitCondition`), регенерация ссылок, auto-reset |
| `client_secret.go` | Ротация client `subSecret` (debounced, аудит) |
| `user.go` | `UserService` — admin/users CRUD, rehash, last-login |
| `setting.go` | `SettingService` — get/set settings, seed-мапа дефолтов, reset, update-channel |
| `tls.go` | `TlsService` — CRUD TLS |
| `stats.go` | `StatsService` — сохранение трафика, агрегация, downsample, prune, online counts |
| `config.go` | `ConfigService` — сборка финального sing-box JSON из БД; инвалидация кэша подпи��ок |
| `panel.go` | `PanelService` — `RestartScheduler`, debounced SIGHUP-рестарт панели |
| `panel_update.go` / `panel_update_apply.go` | Self-update панели |
| `restart_manager.go` | `RestartScheduler` — debounce/cooldown, схлопывание частых правок в один рестарт |
| `coresync.go` | `postCommitCorePlan`/`entityCoreChange` — «reload by id» / «remove by tag» / «full restart» после commit |
| `tagrefs*.go` | `TagReference` + обход конфига/строк для каскадных reload |
| `warp.go` | `WarpService` — генерация Warp/WG кредов |
| `failover_*.go` | assembly / groups / select / validation / state / live — failover outbound |
| `ip_certificate_*.go` | acme / apply / renewal / service / store — жизненный цикл сертификатов |
| `audit.go` / `audit_listen.go` / `audit_writer.go` | Query аудита / listen-fallback события / буферизованный async-writer |
| `observability.go` | `ObservabilityService` — метрики buckets/cores |
| `doctor.go` | `DoctorService` — самодиагностика (config/DB/certs) с severity |
| `update.go` / `update_github.go` | Self-update (GitHub releases) |
| `telegram*.go` | `TelegramService`, encrypted backups (envelope), transport, scheduled reports |
| `session_rotation.go` | Хук инвалидации WS-токенов |
| `token_use_debouncer.go` | Async-дебаунсер телеметрии токенов |
| `paidsub_settings.go` | Тумблеры платных подписок |
| `secret_settings.go` | Управление секретами (web secret, install salt) |

### Точки входа бизнес-логики
- `ConfigService` — собирает live sing-box config из БД (на каждом рестарте/инвалидации кэша).
- `*Service.Save` (Inbound/Outbound/Endpoint/Services/Client) → возвращают `*entityCoreChange` → `coresync` схлопывает в `postCommitCorePlan`.
- `RestartScheduler.Schedule` — коллапс частых правок в один SIGHUP-reload.
- `StatsService.SaveStats` — периодическая запись дельт трафика.
- `ResetCaches` — сброс кэшей после импорта/рестарта.

---

## Data flow (ключевые связи)

1. **HTTP/panel** → методы **service** → живая БД через `database.GetDB()`.
2. Каждый entity-service: транзакция (`tx.Save`/`tx.Delete`) → возвращает `*entityCoreChange`.
3. Commit → merge всех changes в `postCommitCorePlan` (`coresync.go`).
4. Нужен рестарт → `RestartScheduler` debounce → SIGHUP. Иначе — точечные remove-by-tag + reload-by-id.
5. **На рестарте** → `ConfigService` читает Inbound/Outbound/Endpoint/Service (`Preload("Tls")`) → маршалит в sing-box JSON (per-model `MarshalJSON` мержит `Options`) → отдаёт `core.Core`.
6. **Stats**: sing-box API → `SaveStats` → таблица `stats` → query/aggregate/downsample/prune/online.
7. **Подписки**: сохранение inbound → `ClientService` регенерит per-client `links` (`util.LinkGenerator`) → sub URL. `client_secret.go` ротация `subSecret`.
8. **Failover**: `failover_groups` (members) → `failover_live` (статус) → `failover_select` (active) → `failover_assembly` (JSON для ядра).
9. **TLS**: `ip_certificate_service` → acme issue → apply → renewal cron; хранение `ip_certificate_store`; экспонируются как `tls` rows.
10. **Аудит**: действия → `audit_writer` (async) → `audit_events`; `audit.go` query.
11. **Миграции/импорт**: на старте `cmd/migration` гонит `to1_N` если версия старее; `AdaptToCurrentVersion` rehash. Импорт .db → `ImportDB` → SIGHUP → re-init.
12. **3x-ui import**: `database/importxui` read-only, dialect-детект, маппинг в одной транзакции, `Report`. Взаимоисключим с .db restore (тот отклоняет 3x-ui, детектя `client_traffics`/`inbound_client_ips`).
13. **Telegram**: `telegram_report_cron` шлёт отчёты/бэкапы через `telegram_transport`; encrypted через envelope.
14. **Self-update**: `update.go`+`update_github.go`, gate по каналу из `SettingService`.
15. **Диагностика**: `DoctorService` + `ObservabilityService`.
