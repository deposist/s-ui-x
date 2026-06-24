# Release Notes: v1.5.10-beta4
Release date: 2026-06-24

This beta fixes session-expiry handling in the frontend and tightens several backend security checks found during release review. It also keeps the small cleanup work from the previous beta branch.

No manual database or configuration migration is required.

## Session expiry handling

When a session expired or disappeared, the frontend could show `Invalid login` and then replace it with `Error: CSRF token was not returned`. It could also try to call the CSRF-protected logout endpoint while already logged out.

The panel now treats `Invalid login` as a local logout: it clears local CSRF/auth state and returns to the login page without making a logout request. Repeated invalid-session responses from background polling are collapsed into one notification until the next successful login.

CSRF token loading also preserves backend messages such as `Invalid login` instead of replacing them with a missing-token error.

## API scope fixes

Scoped API tokens now match the documented route matrix:

- `observability` tokens can read observability history and core history.
- `telegram` tokens can run manual Telegram database backups.
- Telegram test remains admin-only.

This fixes 403 responses on routes that already had tests and scope-matrix entries for those narrowed token scopes.

## Self-update hardening

The panel self-update path now uses owner-only permissions for the pending-update marker and downloaded staging file. Invalid pending markers are handled explicitly and logged before being reset.

Cleanup after failed update steps now logs unexpected remove errors instead of ignoring them. Security scanner exceptions around controlled self-update file paths now include comments explaining why the paths are safe in this flow.

New regression tests cover:

- tar entries with traversal-looking names being written only to the fixed destination path
- symlink entries being ignored during binary extraction
- owner-only pending marker permissions
- invalid pending marker recovery

## Certificate file permissions

Managed IP certificate files are now written as owner-readable only. The private key was already `0600`; the certificate chain now uses the same mode because both files live together in the managed certificate directory.

## Cleanup

The release also includes small code cleanup from the beta branch:

- removed unnecessary `else` in `main.go`
- removed obsolete loop variable capture in `api/apiHandler.go`
- removed a commented-out settings helper
- simplified tautological map allocations in `api/apiService.go`

## Verification

The following checks pass on the release tree:

- `go test ./... -count=1`
- `golangci-lint run ./...`
- `gosec -quiet ./...`
- `cd frontend && npm run lint`
- `cd frontend && npm test`
- `cd frontend && npm run build`

This is a beta release. Publish it as a GitHub pre-release and keep it out of the Latest stable slot.

---

# Примечания к релизу: v1.5.10-beta4
Дата релиза: 2026-06-24

Эта бета исправляет обработку истёкшей сессии во фронтенде и закрывает несколько замечаний по безопасности, найденных во время релизной проверки. Также в релиз вошла небольшая очистка кода из текущей beta-ветки.

Ручная миграция базы данных или конфигурации не требуется.

## Обработка истёкшей сессии

После истечения или потери сессии фронтенд мог показать `Invalid login`, а затем заменить это сообщение на `Error: CSRF token was not returned`. Также он мог попытаться вызвать CSRF-защищённый logout endpoint уже после потери сессии.

Теперь панель обрабатывает `Invalid login` как локальный выход: очищает локальное CSRF/auth-состояние и возвращает пользователя на страницу входа без запроса logout. Повторные invalid-session ответы от фонового polling сворачиваются в одно уведомление до следующего успешного входа.

Загрузка CSRF-токена теперь сохраняет backend-сообщения вроде `Invalid login`, а не заменяет их ошибкой об отсутствующем токене.

## Исправления API scopes

Scoped API tokens теперь соответствуют зафиксированной матрице маршрутов:

- токены `observability` могут читать observability history и core history
- токены `telegram` могут запускать ручной Telegram backup базы данных
- Telegram test остаётся доступен только admin-токенам

Это исправляет 403 на маршрутах, для которых уже были тесты и записи в scope matrix.

## Усиление self-update

Путь self-update теперь использует owner-only права для pending-update marker и staging-файла загрузки. Некорректный pending marker обрабатывается явно: ошибка логируется, счётчик попыток сбрасывается.

Очистка после неудачных шагов обновления теперь логирует неожиданные ошибки удаления, а не игнорирует их. Исключения security scanner для контролируемых путей self-update теперь снабжены пояснениями.

Добавлены regression tests для следующих случаев:

- tar entry с traversal-похожим именем записывается только в фиксированный destination path
- symlink entry игнорируется при извлечении бинаря
- pending marker создаётся с owner-only правами
- некорректный pending marker восстанавливается безопасно

## Права файлов сертификатов

Файлы managed IP certificate теперь записываются только для владельца. Private key уже был `0600`; certificate chain теперь использует тот же режим, потому что оба файла лежат рядом в managed certificate directory.

## Очистка кода

В релиз также вошли небольшие упрощения из beta-ветки:

- убран лишний `else` в `main.go`
- убран устаревший захват переменной цикла в `api/apiHandler.go`
- удалён закомментированный settings helper
- упрощены тавтологические map allocation в `api/apiService.go`

## Проверка

На релизном дереве проходят:

- `go test ./... -count=1`
- `golangci-lint run ./...`
- `gosec -quiet ./...`
- `cd frontend && npm run lint`
- `cd frontend && npm test`
- `cd frontend && npm run build`

Это beta-релиз. Публикуйте его как GitHub pre-release и не помечайте как Latest stable.
