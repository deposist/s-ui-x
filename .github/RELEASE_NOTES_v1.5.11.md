# Release Notes: v1.5.11
Release date: 2026-07-07

v1.5.11 is a stable patch release for Telegram notification setup. No manual database or configuration migration is required.

## Telegram Chat ID setup

Telegram settings now include a Detect Chat ID action. After you enter a bot token and send `/start` or any message to the bot, the panel can ask Telegram for recent updates and fill the Chat ID field for you.

Detection works with a newly typed token or with a token already saved in the panel. Saved bot tokens remain encrypted at rest and are not sent back to the browser.

If Telegram has no updates for the bot yet, the panel shows a clear message. Open the bot, send `/start`, then run detection again. For groups, add the bot to the group and send a message in that group first.

## Test button behavior

The Telegram Test button now saves changed Telegram settings before sending the test message. This prevents the common `missing_chat` result after filling or detecting Chat ID but forgetting to press Save.

The backend also exposes an admin-only `POST /api/telegram/detect-chat` route for the panel. Detection results are audited without logging the bot token.

## Upgrade notes

Upgrade normally. Existing Telegram settings are kept. If Test still returns `missing_chat`, check that the Chat ID field is filled and saved, or use Detect Chat ID after the bot receives a message.

# Заметки о релизе v1.5.11
Дата релиза: 2026-07-07

v1.5.11 - стабильный patch-релиз для настройки Telegram-уведомлений. Ручная миграция базы данных или конфигурации не требуется.

## Настройка Telegram Chat ID

В настройках Telegram появилась кнопка «Определить Chat ID». Введите токен бота, отправьте боту `/start` или любое сообщение, после этого панель сможет запросить у Telegram последние updates и заполнить поле Chat ID.

Определение работает с только что введённым токеном и с токеном, который уже сохранён в панели. Сохранённые bot tokens остаются зашифрованными at rest и не возвращаются в браузер.

Если у бота ещё нет updates, панель покажет понятное сообщение. Откройте бота, отправьте `/start` и повторите определение. Для группы добавьте бота в группу и сначала отправьте сообщение в этой группе.

## Поведение кнопки Test

Кнопка Telegram Test теперь сохраняет изменённые Telegram-настройки перед отправкой тестового сообщения. Это убирает частый сценарий с `missing_chat`, когда Chat ID уже введён или найден, но Save ещё не нажали.

На backend добавлен admin-only route `POST /api/telegram/detect-chat` для панели. Результат определения попадает в audit, токен бота в audit не записывается.

## Обновление

Обновляйтесь обычным способом. Существующие Telegram-настройки сохраняются. Если Test всё ещё возвращает `missing_chat`, проверьте, что поле Chat ID заполнено и сохранено, или используйте «Определить Chat ID» после сообщения боту.
