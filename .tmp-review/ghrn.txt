# Release Notes: v1.5.10
Release date: 2026-07-03

Stable v1.5.10 consolidates v1.5.10-beta1..beta9. The release focuses on faster large installations, safer update paths, Nexus dashboard and Settings polish, exact traffic totals, Settings compatibility with stale database rows, and the final RU/ZH country-direct preset behavior.

No manual database or configuration migration is required. The stats index added in this line is applied automatically.

## Performance and large installations

Stats charts now prepare large result sets with one sort and one bucket assignment pass instead of scanning the full result for every bucket and traffic direction. Stats writes use SQLite-safe batches, which avoids oversized insert statements on installations with many clients, inbounds, outbounds, or endpoints.

`/api/load` now reads Settings through one request-local snapshot and loads independent entities in parallel. Full panel reloads do less repeated database work.

Unencrypted local database export streams the prepared SQLite backup file to the browser instead of reading the whole file into memory. Encrypted backup downloads keep the existing sealed whole-payload format.

Base, JSON, and Clash subscription responses use a short cache for successful output and clear it after a successful config save. Subscription headers are cached with the response for up to 45 seconds, so repeated requests in that window can show slightly older traffic counters.

The frontend build now splits stable vendor chunks for Vue, Vuetify, and HTTP dependencies. The DateTime picker is lazy-loaded from client modals, and unused Moment locale imports were removed.

## RU/ZH regional presets

The old flat preset gallery was replaced with a side drawer. The drawer has a preview step, marks preset-managed items in both Classic and Nexus, uses deterministic preset tags, and avoids writing unknown metadata fields into sing-box config.

The final stable behavior is country-direct only. Matched RU and ZH country traffic goes through the selected direct outbound. Other traffic keeps the existing final route. The earlier beta proxy-direction choices and per-domain exceptions were removed from the drawer.

RU domain routing uses `wastrel-g/geosite-ru-smart` category `direct-ru`. The panel downloads `geosite.dat`, converts that category into a local sing-box `.srs` rule set, and stores it under the panel data directory as:

```text
rulesets/geosite-ru-smart/direct-ru.srs
```

Saved config keeps a relative managed rule-set path. Runtime config receives the absolute path before sing-box starts. The managed RU smart rule set refreshes after 24 hours and can keep using a valid cached `.srs` file if GitHub is temporarily unavailable. If the file is missing or invalid, the panel reports an error instead of starting with a broken rule set.

RU/ZH DNS rules use `preset-dns-direct` only for matched country domains. They do not replace global `dns.final`. The direct DNS server is still written without `detour: "direct"` when the selected direct outbound is the empty built-in `direct` outbound.

## Nexus dashboard and Settings

The Nexus dashboard layout was tightened across KPI cards, overview panels, protocol summaries, Recent events, System status, and dense tables. Top clients now shows real summary totals, online and total client counts, compact traffic columns, and a link to the Clients page.

Traffic widgets were reworked. Live traffic gained a local time-window selector, and the first dashboard KPI now shows Traffic statistics. The new `GET /api/stats/traffic` endpoint returns exact download and upload totals using `SUM(traffic)` for the selected period. Long ranges no longer depend on average-downsampled `/api/stats` rows. The dashboard includes historical inbound traffic even when the inbound was later disabled or removed.

Recent events and Top clients now scroll inside fixed-height cards, and System status is sized around its current values. The Nexus sidebar shows S-UI-X branding and CPU/RAM percentages when the status API returns them. Emerald and Dracula palettes were added for dark and light Nexus themes with improved light-theme contrast.

Nexus Settings now uses card-based sections. sing-box Basics moved into Settings as the Basics (Singbox) tab, and `/basics` opens that tab. The old Basics menu item was removed from both Classic and Nexus navigation. The moved Basics save button now clears its loading state after failed config saves.

## Security, updates, and API access

Expired sessions are handled more cleanly in the frontend. `Invalid login` clears local auth and CSRF state, returns the browser to the login page, and avoids calling the CSRF-protected logout endpoint. Repeated invalid-session responses are shown once until the next successful login. CSRF token loading now preserves backend errors instead of replacing them with a generic missing-token message.

Scoped API tokens can now access the routes they are meant to use: `observability` tokens can read observability and core history, and `telegram` tokens can run manual Telegram database backups. Telegram test remains admin-only.

The panel self-update path now uses owner-only permissions for pending markers and staging files, handles invalid markers explicitly, logs cleanup errors, and has regression coverage for traversal-looking tar entries, symlink entries, marker permissions, and invalid marker recovery. Managed IP certificate files also use owner-only permissions for both certificate chains and private keys.

The release line also includes small code cleanup and release-pipeline maintenance, including GitHub Actions updates for current Node-backed actions and e2e coverage aligned with the Regional presets drawer.

## Settings compatibility

Settings saves now tolerate old or third-party rows that may still exist in the `settings` table. `GET /api/settings` returns only user-editable keys, so stale rows such as `globalReset` are not sent back to the browser. `POST /api/save` still rejects unknown setting keys, so the save-side allowlist remains strict.

## Upgrade

Upgrade normally. There is no manual database or configuration migration.

If you used an earlier 1.5.10 beta only to test RU/ZH proxy-direction presets, review the Regional presets drawer after upgrading. The stable UI keeps the country-direct preset behavior only.

Full per-beta history is in `CHANGELOG-EN.md`, `CHANGELOG-RU.md`, and `CHANGELOG-ZH.md`.

---

# Примечания к релизу: v1.5.10
Дата релиза: 2026-07-03

Стабильная v1.5.10 объединяет v1.5.10-beta1..beta9. В релиз вошли ускорения для больших установок, более безопасный путь обновления, доработки Nexus Dashboard и Settings, точные totals трафика, совместимость Settings со старыми строками в базе и финальное поведение RU/ZH country-direct пресетов.

Ручная миграция базы данных или конфигурации не требуется. Индекс статистики, добавленный в этой линейке, применяется автоматически.

## Производительность и большие установки

Графики статистики теперь готовят большие наборы данных через одну сортировку и один проход по bucket'ам вместо повторного сканирования всего результата для каждого bucket и направления трафика. Запись статистики использует SQLite-safe batches, чтобы не создавать слишком большие insert statements на установках с большим числом клиентов, inbounds, outbounds или endpoints.

`/api/load` читает Settings через один request-local snapshot и загружает независимые сущности параллельно. Полная перезагрузка данных панели делает меньше повторных запросов к базе.

Незашифрованный локальный экспорт базы стримит подготовленный SQLite backup-файл в браузер, а не читает весь файл в память. Зашифрованные backup downloads сохраняют прежний формат с шифрованием всего payload целиком.

Base, JSON и Clash подписки используют короткий cache для успешных ответов и очищают его после успешного сохранения конфигурации. Headers подписки кешируются вместе с ответом до 45 секунд, поэтому повторные запросы в этом окне могут показывать слегка устаревшие traffic counters.

Frontend build теперь отделяет стабильные vendor chunks для Vue, Vuetify и HTTP-зависимостей. DateTime picker загружается lazy из client modals, лишние Moment locale imports удалены.

## RU/ZH региональные пресеты

Старый flat preset gallery заменён side drawer. В drawer есть preview перед применением, preset-managed элементы помечаются в Classic и Nexus, используются детерминированные preset tags, а в sing-box config не записываются неизвестные metadata-поля.

Финальное стабильное поведение теперь только country-direct. Найденный RU и ZH страновой трафик идёт через выбранный direct outbound. Остальной трафик сохраняет текущий final-маршрут. Более ранние beta-настройки proxy direction и исключения по доменам удалены из drawer.

RU-домены берутся из категории `direct-ru` репозитория `wastrel-g/geosite-ru-smart`. Панель скачивает `geosite.dat`, конвертирует эту категорию в локальный sing-box `.srs` rule set и сохраняет его в каталоге данных панели:

```text
rulesets/geosite-ru-smart/direct-ru.srs
```

В сохранённой конфигурации остаётся относительный managed rule-set path. Перед запуском sing-box runtime config получает абсолютный путь. Managed RU smart rule set обновляется после 24 часов и может продолжать использовать валидный кешированный `.srs`, если GitHub временно недоступен. Если файла нет или он повреждён, панель возвращает ошибку вместо запуска с нерабочим rule set.

DNS-правила RU/ZH используют `preset-dns-direct` только для найденных страновых доменов. Они не заменяют глобальный `dns.final`. Direct DNS server по-прежнему записывается без `detour: "direct"`, если выбран пустой встроенный outbound `direct`.

## Nexus Dashboard и Settings

Разметка Nexus Dashboard стала плотнее в KPI cards, overview panels, protocol summaries, Recent events, System status и dense tables. Top clients показывает реальные totals, число online и всего клиентов, компактные traffic-колонки и ссылку на страницу Clients.

Traffic widgets переработаны. Live traffic получил локальный выбор временного окна, а первый KPI Dashboard теперь показывает Traffic statistics. Новый endpoint `GET /api/stats/traffic` возвращает точные download и upload totals через `SUM(traffic)` за выбранный период. Длинные диапазоны больше не зависят от average-downsampled строк `/api/stats`. Dashboard учитывает исторический inbound-трафик даже для inbound'ов, которые позже отключили или удалили.

Recent events и Top clients прокручиваются внутри карточек фиксированной высоты, а System status рассчитан под текущий набор значений. Nexus sidebar показывает бренд S-UI-X и проценты CPU/RAM, когда status API возвращает эти метрики. Добавлены палитры Emerald и Dracula для тёмной и светлой темы Nexus, с улучшенным контрастом в светлом варианте.

Nexus Settings переведён на карточные секции. sing-box Basics перенесён в Settings как вкладка Basics (Singbox), а `/basics` открывает эту вкладку. Старый пункт Basics удалён из Classic и Nexus navigation. Перенесённая кнопка Save теперь сбрасывает loading state после неудачного сохранения config.

## Безопасность, обновления и доступ API

Истёкшие сессии обрабатываются аккуратнее во frontend. `Invalid login` очищает локальное auth и CSRF state, возвращает браузер на страницу входа и не вызывает CSRF-защищённый logout endpoint. Повторные invalid-session ответы показываются один раз до следующего успешного входа. Загрузка CSRF token теперь сохраняет backend-ошибки, а не заменяет их общим сообщением про отсутствующий token.

Scoped API tokens получили доступ к нужным маршрутам: `observability` tokens могут читать observability и core history, а `telegram` tokens могут запускать ручной Telegram backup базы данных. Telegram test остаётся admin-only.

Путь self-update в панели теперь использует owner-only permissions для pending markers и staging files, явно обрабатывает invalid markers, логирует cleanup errors и покрыт regression tests для traversal-looking tar entries, symlink entries, marker permissions и invalid marker recovery. Managed IP certificate files также используют owner-only permissions для certificate chains и private keys.

В релиз также вошли небольшая чистка кода и обслуживание release pipeline, включая обновление GitHub Actions до актуальных Node-backed actions и e2e coverage под Regional presets drawer.

## Совместимость Settings

Сохранение Settings теперь нормально работает на базах, где в таблице `settings` остались старые или сторонние строки. `GET /api/settings` возвращает только user-editable keys, поэтому stale rows вроде `globalReset` не уходят в браузер. `POST /api/save` по-прежнему отклоняет неизвестные setting keys, так что save-side allowlist остаётся строгим.

## Обновление

Обновляйтесь обычным способом. Ручная миграция базы данных или конфигурации не требуется.

Если вы использовали ранние beta 1.5.10 только для проверки RU/ZH proxy-direction пресетов, после обновления проверьте Regional presets drawer. Стабильный UI оставляет только country-direct поведение.

Полная история beta-релизов находится в `CHANGELOG-EN.md`, `CHANGELOG-RU.md` и `CHANGELOG-ZH.md`.
