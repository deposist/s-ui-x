# Release Notes: v1.5.10-beta1
Release date: 2026-06-23

First beta of the 1.5.10 line. This release focuses on performance remediation from the 2026-06-23 audit: faster stats charts, less database work on full panel reloads, safer backup downloads for large databases, safer stats writes on large installations, subscription output caching, and smaller/cache-friendlier frontend chunks.

No manual database migration or configuration change is required. Public API and UI behaviour are preserved.

**EN**

## 1. Faster dashboard stats

The stats chart downsampling path now buckets rows in a single pass after sorting instead of scanning all rows for every bucket and direction. Large stats windows should use substantially less CPU when the dashboard or stats API requests chart data.

Stats writes also use SQLite-safe explicit batching, avoiding oversized multi-row inserts on installations with many clients, inbounds, outbounds, or endpoints.

## 2. Lighter `/api/load` full reloads

The full reload path now reads the panel settings needed for load data through one request-local snapshot instead of several duplicate settings queries. Independent entity reads for clients, TLS, inbounds, outbounds, endpoints, services, and settings are also parallelized, reducing latency when the frontend needs a full refresh.

## 3. Safer database backup downloads

Unencrypted local database export now streams the prepared backup file to the HTTP response instead of reading the whole SQLite backup into memory first. This reduces memory spikes on large databases. Encrypted Telegram-style backup downloads still buffer plaintext internally because the current authenticated envelope format encrypts the complete payload at once.

## 4. Subscription output cache

Base, JSON, and Clash subscription outputs are cached for a short TTL and cleared after a successful config save. This reduces repeated CPU work when many clients poll subscriptions at the same time, for example after a restart. Successful responses only are cached; not-found and error responses are not cached.

Note: subscription headers are cached together with the response for up to 45 seconds, so traffic/userinfo values can lag slightly on repeated requests during that window.

## 5. Frontend bundle and cache improvements

The main app entry is split into stable vendor chunks for Vue, Vuetify, and HTTP dependencies, improving browser cache reuse between releases. The DateTime picker path is lazy-loaded from client modals and no longer imports extra Moment locale files globally.

The full Moment removal was intentionally deferred: the current Persian datetime picker directly depends on `moment-jalaali`, so replacing Moment safely requires replacing the picker component. Chart.js and YAML replacements were evaluated and deferred because they need visual/config compatibility coverage.

## Upgrade

Upgrade normally. There is no manual database or configuration migration. Because this is a beta, publish it as a GitHub pre-release and keep it out of the Latest stable slot.

---

# Примечания к релизу: v1.5.10-beta1
Дата релиза: 2026-06-23

Первая бета линейки 1.5.10. Релиз закрывает performance remediation по аудиту от 2026-06-23: быстрее графики статистики, меньше работы с базой при полном обновлении панели, безопаснее скачивание больших backup-файлов, устойчивее запись статистики на крупных установках, кеширование подписок и более cache-friendly frontend chunks.

Ручная миграция базы и изменение конфигурации не требуются. Публичное поведение API и UI сохранено.

**RU**

## 1. Быстрее статистика на dashboard

Downsampling для графиков статистики теперь раскладывает строки по bucket'ам за один проход после сортировки, вместо повторного сканирования всех строк для каждого bucket и направления. На больших окнах статистики это снижает CPU-нагрузку при запросах dashboard/stats API.

Запись статистики также переведена на явные безопасные SQLite batch-вставки, чтобы не упираться в лимит переменных SQLite на установках с большим числом клиентов, inbounds, outbounds или endpoints.

## 2. Легче полный `/api/load`

Полная загрузка данных панели теперь получает нужные settings через один request-local snapshot вместо нескольких повторных запросов к settings. Независимые чтения clients, TLS, inbounds, outbounds, endpoints, services и settings выполняются параллельно, что уменьшает задержку при полном refresh фронтенда.

## 3. Безопаснее скачивание backup базы

Незашифрованный локальный экспорт базы теперь стримит подготовленный backup-файл в HTTP-ответ, а не читает весь SQLite backup в память целиком. Это уменьшает memory spikes на больших базах. Зашифрованные Telegram-style backup downloads пока по-прежнему буферизуют plaintext, потому что текущий authenticated envelope шифрует весь payload целиком.

## 4. Кеш output подписок

Base, JSON и Clash подписки кешируются на короткий TTL и очищаются после успешного сохранения конфигурации. Это уменьшает повторную CPU-работу, когда много клиентов одновременно запрашивают подписки, например после рестарта. Кешируются только успешные ответы; not-found и ошибки не кешируются.

Замечание: headers подписки кешируются вместе с ответом до 45 секунд, поэтому traffic/userinfo значения могут немного отставать при повторных запросах в этом окне.

## 5. Frontend bundle и кеширование

Главный app entry разделён на стабильные vendor chunks для Vue, Vuetify и HTTP-зависимостей, что улучшает browser cache reuse между релизами. Путь DateTime picker теперь lazy-loaded из client modals и больше не импортирует лишние Moment locale files глобально.

Полное удаление Moment намеренно отложено: текущий Persian datetime picker напрямую зависит от `moment-jalaali`, поэтому безопасная замена требует замены самого picker component. Замены Chart.js и YAML оценены и отложены, потому что требуют visual/config compatibility coverage.

## Обновление

Обновляйтесь обычным образом. Ручной миграции базы или конфигурации нет. Так как это beta, публикуйте её как GitHub pre-release и не помечайте как Latest stable.
