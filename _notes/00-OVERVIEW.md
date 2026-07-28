# s-ui-x — Обзор проекта

> Заметки для внутренней работы. Собраны при первичном изучении репозитория 2026-07-27.
> Репозиторий: https://github.com/DiP4RiP/s-ui-x (upstream/canonical: `deposist/s-ui-x`).
> Локальный путь: `/home/node/.openclaw/workspace/s-ui-x`

## Что это

**s-ui-x** — веб-панель управления прокси на базе `SagerNet/Sing-Box`.
Форк `alireza0/s-ui` начиная с `v1.4.1`, поверх которого добавлены харденинг
безопасности, надёжности, наблюдаемость и переработанный UI.

- Go-модуль: `github.com/deposist/s-ui-x`
- Go **1.26.4**, sing-box встроен как **библиотека in-process** (не subprocess): `sagernet/sing-box v1.13.13`
- Один бинарник `sui` (backend + встроенный собранный фронтенд через `//go:embed`)
- Хранилище: **SQLite** через **GORM** (`mattn/go-sqlite3`, WAL)
- Фронтенд: **Vue 3 + Vuetify 4 + Vite 8 (rolldown)**, две темы UI: **Nexus (по умолчанию)** и classic

## Текущая версия

- `config/version` = **`1.5.10-beta7`** (нота релиза `docs/releases/v1.5.10-beta7.md`, 2026-06-25)
- frontend `package.json` тоже `1.5.10-beta7`
- Текущий stable по README: **v1.5.9**
- git HEAD: `c79b96e` (change "final": "proxy" → "final": "direct")
- `[Unreleased]` в CHANGELOG-{EN,RU,ZH}.md пуст

## Порты и пути по умолчанию

| Параметр | Значение |
|---|---|
| Порт панели | **2095** |
| Путь панели | `/app/` |
| Порт подписок | **2096** |
| Путь подписок | `/sub/` |
| Таймзона по умолчанию | **Europe/Moscow** (в `service/setting.go`, `defaultValueMap["timeLocation"]`) |

## Платформы

- **Linux**: amd64, arm64, armv7/6/5, 386, s390x — Supported
- **Windows**: amd64, 386, arm64 — Supported
- **macOS**: amd64, arm64 — Experimental

## Локализация

- **Панель (UI)**: 6 локалей — en, fa, vi, zhcn, zhtw, ru (по умолчанию en)
- **Установщик/меню (bash)**: только EN/RU/ZH (`SUI_LANG=en|ru|zh`)
- ⚠️ Рассинхрон: FA/VI есть только в панели, не в bash-скриптах.

## Ключевые отличия от upstream `alireza0/s-ui`

- **Строгая аутентификация**: случайный первый admin-пароль, bcrypt, hardened cookies, CSRF на мутирующих браузерных вызовах. API-токены хэшируются и скоупятся: `admin | read | write | observability`.
- **Секреты как секреты**: Telegram-креды, прокси-креды, install salt и др. шифруются at-rest через **secretbox** (`SUI_SECRETBOX_KEY`). Редакция в Telegram-сообщениях, аудите, captions.
- **Жёсткие guardrails для сети**: `X-Forwarded-For` игнорируется без trusted proxies; внешние фетчи подписок — URL/IP-проверки, блок private/loopback, лимит 4 MiB, re-check IP на dial.
- **Безопасные подписки**: per-client sub-secrets (link/json/clash), rate-limit по IP, gzip, короткий кэш успешных ответов, no-store.
- **Ядро in-process**: sing-box как Go-библиотека. Hot-apply изменений где возможно, полный рестарт только когда нужно.
- **failover outbound**: упорядоченный список members, HTTPS-пробы, переключение и fail-back.
- **Self-update из UI**: проверка stable/beta, верификация SHA-256, применение, авто-rollback при повторном фейле старта. Требует повторного ввода пароля, всё аудируется.
- **Защитные бэкапы/импорты**: cap 64 MiB, SQLite magic check, staging, integrity check, миграции, rollback.
- **Аудит и наблюдаемость**: audit events + retention, scoped/paginated audit API, bounded логи, observability buckets, hardened WebSocket (single-use токены, Origin checks).
- **IP-мониторинг**: приватность по умолчанию (salted hashes), raw IP opt-in, enforce отклоняет только новые over-limit соединения.
- **Безопасные дефолты сервера**: таймауты read/write/header/idle, TLS MinVersion 1.2, security headers.
- **Nexus UI** по умолчанию, classic сохранён.

## Структура каталогов (Go-файлов)

| Каталог | Файлов | Назначение |
|---|---:|---|
| `api/` | 58 | HTTP API (session + token v2), CSRF, rate-limit, импорт x-ui |
| `app/` | 1 | Жизненный цикл приложения (`app.go`) |
| `cmd/` | 24 | CLI: admin, import-xui, decrypt-backup, migration/*, setting, ip-cert |
| `config/` | 6 | Версии, каналы обновлений, artifact platform |
| `core/` | 26 | Встроенный sing-box, трекеры, failover, регистри |
| `cronjob/` | 23 | Плановые задачи (stats, deplete, failover, backups…) |
| `database/` | 65 | GORM-модели, миграции, backup, importxui/ |
| `logger/` | 5 | Логирование |
| `middleware/` | 4 | Gin middleware |
| `network/` | 6 | Сетевые утилиты/guardrails |
| `paidsub/` | 28 | Платные подписки (экспериментальный модуль) |
| `realtime/` | 5 | WebSocket realtime |
| `service/` | 122 | Бизнес-логика (CRUD, config assembly, audit, telegram, cert…) |
| `sub/` | 18 | Отдача подписок (link/json/clash) |
| `util/` | 34 | Утилиты, генерация ссылок, redact, crypto |
| `ipmonitor/` | 5 | IP-лимиты по клиентам |
| `web/` | 6 | HTTP-сервер панели, TLS, embed фронтенда |
| `frontend/` | — | Vue 3 SPA |

## Файлы этих заметок
- `00-OVERVIEW.md` — этот файл.
- `01-backend-service-db.md` — `service/`, `database/`.
- `02-api-web-realtime-sub.md` — `api/`, `web/`, `realtime/`, `sub/`.
- `03-core-cmd-config-cron-paidsub.md` — `core/`, `cmd/`, `config/`, `cronjob/`, `paidsub/`.
- `04-frontend-build-ops.md` — `frontend/`, Build, Ops, Docs.
- `05-subpage-landing.md` — форк-фича: `subpage/`, лэндинг «Личный кабинет» подписки с диплинками для sing-box/Hiddify/Clash/mihomo/v2ray.

- `00-OVERVIEW.md` — этот файл
- `01-backend-service-db.md` — service/ + database/ (модели, миграции, data flow)
- `02-api-web-realtime-sub.md` — api/ middleware/ web/ realtime/ sub/
- `03-core-cmd-config-cron-paidsub.md` — core/ cmd/ config/ cronjob/ paidsub/ ipmonitor/ и др.
- `04-frontend-build-ops.md` — frontend/ + install/build/ops + docs
