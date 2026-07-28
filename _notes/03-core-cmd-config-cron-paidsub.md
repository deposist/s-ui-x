# 03 — core / cmd / config / cronjob / paidsub / прочее

## Startup: main.go + app/app.go

**main.go (24 строки):** нет аргументов → `runApp()`; есть → `cmd.ParseCmd()`. `runApp()` строит `app.NewApp()` → `Init()` → `Start()` → трапит `SIGHUP` → `RestartApp()` (Stop+Start без Init) и `SIGTERM` → `Stop()`.

**app/app.go:**
- **`Init()`:** log init → self-update rollback check (SR-012: детект фейл-бута, `os.Exit(1)` для отката через systemd) → **`migration.MigrateDb()`** → `database.InitDB` → `SettingService` → **`ResealSecretSettings()`** (пере-шифровка секретов под `SUI_SECRETBOX_KEY`, идемпотентно) → `ipmonitor.WarmUp()` → `core.NewCore()` + `service.NewRuntime(core)` → `ipmonitor.SecurityEventAuditHook` → `cronjob.NewCronJob()` → `web.NewServer` → `sub.NewServer()` → `ConfigService` → best-effort `paidsub.EnsureSchema()` + `service.EnsureFailoverSchema()` (не фатально).
- **`Start()`:** TZ + traffic-age из settings → `cronJob.Start` → web → sub → `paidsub.StartBot()` (gate `paidSubEnabled`) → `configService.StartCore()` (**не фатально** — панель живёт даже при битом конфиге) → `ClearPendingUpdate`.
- **`Stop()`:** упорядоченный teardown, каждый шаг с 5s таймаутом: RestartManager → cron → sub → web → StopCore → TokenUseDebouncer → TelegramNotifier → paidsub → AuditWriter.

## cmd/ — CLI

`cmd/cmd.go` — switch по `os.Args[1]`. Команды: `admin`, `uri`, `migrate`, `import-xui`, `ip-cert`, `decrypt-backup`, `setting`, `-v`.
- **admin.go** — `-reset` генерит **случайный 16-char пароль** (не admin/admin), печатает раз; `-show`; `-username/-password`.
- **import_xui.go** — `importxui.Import`. Флаги: `--src`, `--dry-run`, `--strategy {merge,replace,skip}`, `--report`, `--yes`, `--include-history`, `--include-routing`, `--host`. Pre-import backup.
- **decrypt_backup.go** — расшифровка Telegram-envelope. `--in/--out/--passphrase-stdin|--passphrase-env`. Wipe буферов, atomic write, generic `decryption_failed` (без утечки).
- **setting.go** — `-show/-reset/-clearDomain/-port/-path/-subPort/-subPath`; `getPublicIP()` (5 сервисов параллельно, 3s), `getPanelURI()`.
- **ip_certificate.go** — встроенный lego/ACME (без `os/exec`). `issue -ip -email [-port 80] [-no-renew]`, `renew`, `status`, `disable`.

### cmd/migration/ — драйвер миграций
`main.go` открывает БД напрямую (не через InitDB). Pre-flight: `ensureNoTLSForeignKeyParent` (sentinel `tls.id=0`), `verifyForeignKeysBeforeMigration` (`PRAGMA foreign_key_check`). Одна транзакция: читает `version`, no-op если равно, skip если БД новее (semver). Цепочка `to1_1…to1_7`. Финал: UPSERT `settings.version`, `PRAGMA wal_checkpoint(FULL)`.

**По версиям:**
- **1_1** — re-marshal clients JSON, `deleteOldWebSecret`, `changesObj`
- **1_2** — `moveJsonToDb` (legacy `bin/config.json` → БД; route `block`→`action:reject`, dns→`hijack-dns`, sniff), `migrateTls`, `dropInboundData`, `migrateClients` (tag→ID), drop `changes.index`
- **1_3** — `anytls_user_config` (клон trojan→anytls), `migrate_dns` (типизация DNS-серверов), `remove_outbound_strategy`
- **1_4** — токен-колонки (`token_hash`, `scope DEFAULT admin`, …) + таблица `audit_events` + backfill
- **1_5** — `clients.limit_ip/ip_limit_mode/last_online/last_ip_count/sub_secret`; таблица `client_ips`; **`installSalt`** (32 random) в settings; backfill `ip_hash=sha256(salt||ip)`, UUID `sub_secret`
- **1_6** — `audit_events` индексы `(event,date_time DESC)`, `(severity,date_time DESC)`
- **1_7** — `xui_sync_profiles` + `xui_known_hosts` (3x-ui sync feature)

## core/ — встроенный sing-box (26 файлов)

**core/main.go (`Core`):** `sync.RWMutex`-защищённый `*Box` + кэш менеджеров (Inbound/Outbound/Service/Endpoint Manager, Router, log.Factory). `NewCore()` сеет контекст через `sb.Context(ctx, InboundRegistry(), OutboundRegistry(), EndpointRegistry(), DNSTransportRegistry(), ServiceRegistry())`.

**register.go — регистри:**
- Inbound: tun, redirect(+TPROXY), direct, socks, http, mixed, shadowsocks, vmess, trojan, naive(+quic), shadowtls, vless, anytls, hysteria, tuic, hysteria2
- Outbound: direct, block, selector+urltest, socks, http, ss, vmess, trojan, naive(cond), tor, ssh, shadowtls, vless, anytls, hysteria, tuic, hysteria2
- Endpoint: wireguard + tailscale
- DNSTransport: TCP/UDP/TLS/HTTPS/hosts/local/fakeip/quic(+H3)/dhcp + tailscale
- Service: resolved, ssmapi + DERP, ccm, ocm, oomkiller

**Build-теги:**
- `with_naive_outbound` — реальный naive.RegisterOutbound, иначе no-op+error
- `with_tailscale` — tailscale/DERP, иначе заглушки с понятной ошибкой (не паника)

- **outbound_check.go** — `CheckOutbound(ctx, tag, link)` через `urltest.URLTest` (15s). Используется `service.Doctor` и `cronjob.FailoverJob`.
- **validate.go** — `ValidateConfig` строит Box без старта (проверка перед commit).
- **tracker_stats.go** — `adapter.ConnectionTracker`; счётчики `atomic.Int64` per `(inbound,outbound,user)`; **вызывает `ipmonitor.Allow`** до accept (deny→закрытый conn) и `ipmonitor.Record` при allow; `GetStats()` → `[]model.Stats`.
- **tracker_conn.go** — трекинг всех соединений по UUID; `CloseConnByInbound`; epoch-based reset.
- **tracker_policy.go** — пин версии `TrackerValidatedSingBoxVersion = "v1.13.13"` + 7 инвариантов (информативно).
- **tracker_wait.go** — `trackerWaitGroup`, `waitForTrackerIdle(5s)` на shutdown.

## config/ (6 файлов)
- **config.go** — `//go:embed version, name`. `GetVersion/GetName/GetLogLevel` (`SUI_LOG_LEVEL`), `IsDebug` (`SUI_DEBUG`), `IsSafeLogOutputPath`, `GetDBFolderPath` (`SUI_DB_FOLDER` или `dir/db`), `GetDBPath`, `GetSecret` (`SUI_SECRET`), `GetForceCookieSecureEnv`.
- **version_policy.go** — Semver 2.0: `ValidateReleaseVersion`, `ParseSemver`, `CompareVersions`, `VersionIsNewer`, `NormalizeVersion`.
- **update_channel.go** — `main` / `beta`, `NormalizeUpdateChannel`→`main` (SR-004 allowlist).
- **artifact_platform.go** — `ArtifactPlatform` инжектится ldflags; `ResolveArtifactPlatform()` (ARM без ldflags → `""` = self-update недоступен).

## cronjob/ (23 файла) — плановые задачи

`cronJob.go` — один `robfig/cron/v3` с `WithLocation(loc)`, `WithSeconds()`, `Recover`+`SkipIfStillRunning`.

| Расписание | Задача | Что делает |
|---|---|---|
| @every 10s | StatsJob | `SaveStats` — flush счётчиков в БД |
| @every 1m | DepleteJob | `DepleteClients()` → disable → `RestartInbounds` |
| @daily | DelStatsJob | `DelOldStats(trafficAge)` (если >0) |
| @every 5s | CheckCoreJob | `StartCore()` если не запущен; publish `TopicCoreState` на edge |
| @every 12s | CPUHysteresisJob | окно 5 сэмплов (60s); `cpu_high/cpu_normal` Telegram (threshold def 90) |
| @every 2s | ObservabilitySamplerJob | 2s buckets host+core; агрегация 30s/1m/5m |
| @every 5s | FailoverJob | per-group по `group.Interval`; пробы+переключение |
| @every 1m | TelegramReportScheduler | реплан отчёта по settings |
| @every 1m | TelegramBackupScheduler | реплан encrypted backup |
| @every 10m | WALCheckpointJob | `PRAGMA wal_checkpoint(FULL)` |
| @every 1h | AuditGCJob | prune `audit_events`+`client_ips` по retention |
| @every 20s | PaidSubPollJob | `paidsub.PollOnce` (gate `paidSubEnabled`) |
| @every 12h | CertRenewJob | `RenewIfNeeded` (gate внутри) |

**failoverJob.go/failoverProbe.go:** загрузка групп, prune live-статуса, early return если ядро не запущено. Пробы members конкурентно (`failoverProbeConcurrency=4`), `ConsecutiveUp/Down`, `SelectFailoverMember` (чистое решение), switch через `core.SelectGroupMember` только при `ShouldSwitch`. Edge `failover_all_down` audit+realtime.

## paidsub/ (28 файлов) — платные подписки (экспериментально)
Самодостаточный модуль: `EnsureSchema()`, `StartBot()`/`StopBot()`, `PollOnce()` (крон @20s), `RegisterRoutes()` в api. Gate по `paidSubEnabled`. UI: `views/paidsub/PaidSubscriptions.vue`.

## ipmonitor/ (5 файлов)
IP-лимиты по клиентам. `WarmUp()`, `Allow(user, ip)` / `Record(user, ip)` (зовутся из `core/tracker_stats.go`), `InvalidateAllCache`, `SecurityEventAuditHook` (debounced, только аудит). Salted hashes, enforce отклоняет только новые over-limit.

## network/ (6 файлов)
Сетевые guardrails: валидация URL/IP для внешних фетчей, блок private/loopback, re-check resolved IP на dial, cap ответов. Trusted-proxy логика для `X-Forwarded-For`.

## logger/ (5 файлов)
Уровни (Debug/Info/Warn/Error), bounded логи, безопасный output path.

## util/ (34 файла)
Утилиты: `LinkGenerator` (генерация подписочных ссылок), `redact` (скрабление секретов), crypto/secretbox-хелперы, random, QR (`skip2/go-qrcode`), прочее.
