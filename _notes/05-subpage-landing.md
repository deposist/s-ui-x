# 05 — subpage/ — лэндинг «Личный кабинет» подписки

## Зачем
Стандартные эндпоинты `/sub/{subid}` и `/sub/json/{subid}`, `/sub/clash/{subid}` отдают **сырой конфиг** (base64-links / sing-box JSON / Clash YAML). Конечному пользователю неудобно: нужно копировать ссылку, вставлять в клиент руками, гадать, какой формат у какого приложения. `subpage/` превращает это в **мини-личный-кабинет** с кнопками-диплинками и сводкой по трафику/сроку.

## Расположение
- Пакет `subpage/` в корне модуля (`github.com/deposist/s-ui-x/subpage`).
- **Изолирован от upstream `alireza0/s-ui`**: всё новое лежит здесь, в апстрим-коде ровно одна точка касания — `sub/sub.go` (см. ниже). Цель — минимизировать merge-конфликты при подтягивании upstream-обновлений.

## Эндпоинт
- По умолчанию: `GET/HEAD /cabinet/{subid}`.
- Настройки: `subPageEnabled` (bool, default `false`) и `subPagePath` (string, default `/cabinet`). Меняются как обычные sub-настройки через админ-панель — БД не нуждается в миграции (key/value `Setting`).
- Порт: тот же, что у `/sub/` — **2096**. Никакого нового порта, никакого нового firewall-правила.

## Что показывает страница
- **Шапка:** имя клиента, последняя активность (человек читаемо: «12 мин назад»).
- **Три плитки:** использовано трафика / остаток / срок действия (на русском, с правильным склонением «1 день / 2 дня / 5 дней»).
- **Прогресс-бар** использования трафика (вычисляется в JS из форматированных строк, чтобы не плодить парсер на бэке).
- **Кнопки-диплинки:**
  - `sing-box://import-remote-profile?url=...` — sing-box ≥ 1.11 (iOS/Android/macOS/Windows/Linux).
  - `hiddify://import-sub?url=...` — Hiddify Next.
  - `clash://install-config?url=...` — Clash Verge / Clash Meta for Android / mihomo (форк Clash Meta, тот же диплинк).
  - `v2rayn://install-config?url=...` — v2rayN (Windows), v2rayNG / Nekoray на Android/iOS/macOS/Linux (получает raw base64-links, rule-sets игнорируются — это осознанное упрощение, см. раздел «v2ray-нюанс»).
- **Прямые ссылки:** вкладки «Универсальная / Sing-box JSON / Clash+mihomo» с кнопкой «Копировать» (clipboard API + fallback на `execCommand`).
- **Баннер предупреждения**, если на сервере выключен `subSecretRequired` — «ваш subId можно подобрать по имени, попросите админа включить обязательные секреты».
- **Баннер объявления** из `subAnnounce`.
- **Поддержка** (если заполнен `subSupportUrl`).
- **Подвал** с названием панели из `subTitle`.

## Дизайн-решения
- **Локализация: только RU** — никакого i18n-обвязки. Если потом потребуется EN/ZH — добавим `Accept-Language` парсинг.
- **CSS встроенный** (через `//go:embed` в `embed.go`), без CDN-зависимости. Размер ~4 KB. Шрифты — системные, иконок нет (эмодзи Unicode).
- **JS минимальный**: переключение вкладок и копирование. Без Alpine.js / React / сборщиков.
- **HTML рендерится через `html/template`** — XSS-защита из коробки.
- **HEAD на `/cabinet/{subid}`** отдаёт ту же страницу, не SIP-заголовки (защита от misroute, если кто-то случайно скормит этот URL в SIP-клиент).

## Файлы

| Файл | Назначение |
|---|---|
| `subpage/page.go` | `Mount()`, handler `renderCabinet`, lookup клиента, сборка `pageData`, deep-link генератор, форматирование. |
| `subpage/page_middleware.go` | Локальный per-IP rate-limit (30/min дефолт, читает `subRateLimitPerIP`). Не зависит от приватных символов `sub/`. |
| `subpage/page.tmpl.html` | Шаблон страницы + CSS + JS. ~10 KB, embed. |
| `subpage/embed.go` | `//go:embed page.tmpl.html`. |
| `subpage/page_test.go` | Unit-тесты без БД: форматирование, deep-link builder, валидация subid, заголовки. |

## Upstream-точки касания (всё, что меняется в `alireza0/s-ui`)

1. `sub/sub.go` — добавлен импорт `github.com/deposist/s-ui-x/subpage` и вызов `subpage.Mount(engine, subPath)` после `registerCustomFormatRoutes`. **5 строк.**
2. `service/setting.go` — добавлены ключи `"subPageEnabled": "false"`, `"subPagePath": "/cabinet"` в `defaultValueMap` и геттеры `GetSubPageEnabled()`, `GetSubPagePath()`. **~30 строк.**

Больше нигде upstream-файлы не меняются. При merge из `alireza0/s-ui` вероятен конфликт только в этих двух местах.

## Безопасность
- subId в URL — единственный секрет. С UUID-форматом (включён по дефолту через `subSecretRequired=true`) подобрать невозможно.
- Если subSecretRequired выключен — **явный баннер** на странице с просьбой включить.
- HTML auto-escape.
- Rate-limit на /cabinet — наследуется от middleware в пакете.
- Заголовок `X-Content-Type-Options: nosniff` — защита от sniffing, если страница случайно окажется на пути подписки.

## Чего пока НЕ делает
- **Не хостит rule-sets.** Если захочешь подменять URL rule-sets в sing-box JSON на свой домен — это отдельный патч в `sub/jsonService.go` (плагин-подмена URL в `route.rule_set[*].url`).
- **Нет авторизации пользователя** по паролю — это же лк, юзер знает только subId. Если потребуется login/password — отдельная фича.
- **Нет мультиязычности.** RU только.
- **Нет брендинга/темы через настройки.** Тёмная тема = `prefers-color-scheme`, без конфига.

## v2ray-нюанс
- v2ray-клиенты получают raw base64 (`vmess://`, `vless://`, `ss://`, `trojan://`) — это совместимо с v2rayN/v2rayNG/Nekoray.
- sing-box JSON **содержит `rule_set` с URL** (по дефолту на MetaCubeX/meta-rules-dat). v2ray-клиенты это игнорируют — для них это просто ссылка на конфиг, который они не парсят.
- Если позже запилишь свой хостинг rule-sets — добавим шаблон подмены URL в `sub/jsonService.go`. Это не блокирует текущий subpage.

## Как включить
1. Залить код (форк уже с этим коммитом).
2. Открыть админку → Настройки → `subPageEnabled = true` (по дефолту `false`, фича-флаг).
3. Опционально: `subPagePath = /cabinet` (или другой путь).
4. Перезапустить панель. **Никаких миграций БД не нужно.**
5. Проверить: открыть `https://<sub-host>:<sub-port>/cabinet/<subSecret>`.