# Release Notes: v1.5.10-beta8
Release date: 2026-07-01

This beta fixes Settings saves for databases that contain stale or third-party setting rows such as `globalReset`.

No manual migration is required. Existing stale rows may remain in the `settings` table, but they are no longer returned to the Settings page and are no longer sent back in the next save request.

## Settings save fix

`GET /api/settings` now returns only user-editable setting keys. This keeps old or unknown rows out of the browser payload before the user clicks Save.

`POST /api/save` still rejects unknown setting keys. The fix does not weaken the save-side allowlist. It only prevents backend-returned stale rows from being round-tripped by the UI.

This fixes the error:

```text
save: invalid setting key: globalReset
```

## Tests

Added a regression test that inserts a stale `globalReset` row, verifies that `GetAllSetting` filters it out, and confirms that the returned settings payload can be saved again.

---

# Примечания к релизу: v1.5.10-beta8
Дата релиза: 2026-07-01

Эта beta исправляет сохранение Settings для баз данных, где в таблице `settings` остались старые или сторонние строки, например `globalReset`.

Ручная миграция не требуется. Такие строки могут остаться в таблице `settings`, но Settings page больше не получает их от backend и не отправляет обратно при следующем сохранении.

## Исправление сохранения Settings

`GET /api/settings` теперь возвращает только ключи, которые разрешено редактировать пользователю. Старые или неизвестные строки не попадают в payload браузера перед нажатием Save.

`POST /api/save` по-прежнему отклоняет неизвестные ключи. Исправление не ослабляет allowlist на сохранении. Оно только не даёт UI повторно отправлять устаревшие строки, которые пришли из backend.

Исправлена ошибка:

```text
save: invalid setting key: globalReset
```

## Тесты

Добавлен регрессионный тест: он вставляет устаревшую строку `globalReset`, проверяет фильтрацию в `GetAllSetting` и подтверждает, что полученный settings payload снова сохраняется без ошибки.
