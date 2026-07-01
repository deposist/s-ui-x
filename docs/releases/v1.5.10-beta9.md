# Release Notes: v1.5.10-beta9
Release date: 2026-07-01

This beta changes the regional routing presets. RU and ZH presets now send matched country traffic through the selected direct outbound and leave the existing final route for all other traffic unchanged.

No manual database migration is required.

## Regional presets

RU now uses `wastrel-g/geosite-ru-smart` category `direct-ru` for domain routing. The panel downloads `geosite.dat`, converts the `direct-ru` category into a local sing-box `.srs` rule set, and stores it under the panel data directory:

```text
rulesets/geosite-ru-smart/direct-ru.srs
```

The saved config keeps a relative managed path. The runtime config receives the absolute path before sing-box starts. This keeps backups portable while still giving sing-box a valid local rule-set file.

ZH keeps the existing SagerNet CN geosite and GeoIP `.srs` sources.

## DNS behavior

For matched RU and ZH country domains, presets add DNS rules that use `preset-dns-direct`. They do not set or replace global `dns.final`.

The direct DNS server is still written without `detour: "direct"` when the selected direct outbound is the empty built-in `direct` outbound. This avoids the sing-box error:

```text
detour to an empty direct outbound makes no sense
```

## Updates and fallback

The managed RU smart rule set is refreshed after 24 hours. If the cached `.srs` file is valid and GitHub is temporarily unavailable, the panel keeps using the cached file. If the file is missing or invalid, saving or starting a config that references the managed RU rule set reports an error instead of starting with a broken rule set.

## UI changes

The Regional presets drawer now shows only the direct outbound selector. The old proxy direction choice and per-domain exceptions were removed from the drawer because the current presets are country-direct only.

---

# Примечания к релизу: v1.5.10-beta9
Дата релиза: 2026-07-01

Эта beta меняет региональные пресеты маршрутизации. RU и ZH пресеты теперь отправляют найденный страновой трафик через выбранный direct outbound и не меняют существующий final-маршрут для остального трафика.

Ручная миграция базы данных не требуется.

## Региональные пресеты

RU теперь использует категорию `direct-ru` из `wastrel-g/geosite-ru-smart` для доменной маршрутизации. Панель скачивает `geosite.dat`, конвертирует категорию `direct-ru` в локальный sing-box `.srs` rule set и сохраняет его в каталоге данных панели:

```text
rulesets/geosite-ru-smart/direct-ru.srs
```

В сохранённой конфигурации остаётся относительный managed path. Перед запуском sing-box runtime config получает абсолютный путь. Так backup остаётся переносимым, а sing-box получает корректный local rule set.

ZH продолжает использовать существующие SagerNet CN geosite и GeoIP `.srs` источники.

## Поведение DNS

Для найденных RU и ZH страновых доменов пресеты добавляют DNS-правила через `preset-dns-direct`. Они не задают и не заменяют глобальный `dns.final`.

Direct DNS server по-прежнему записывается без `detour: "direct"`, если выбран пустой встроенный outbound `direct`. Это предотвращает ошибку sing-box:

```text
detour to an empty direct outbound makes no sense
```

## Обновление и fallback

Managed RU smart rule set обновляется после 24 часов. Если локальный `.srs` валиден, а GitHub временно недоступен, панель продолжает использовать кешированный файл. Если файла нет или он повреждён, сохранение или запуск конфигурации с этим managed RU rule set возвращает ошибку вместо запуска с нерабочим rule set.

## Изменения UI

В Regional presets drawer остался только выбор direct outbound. Старый выбор proxy-направления и исключения по доменам удалены из drawer, потому что текущие пресеты работают только в режиме country-direct.
