# Release v1.5.9-beta3: What's New

This release lets you update the panel from the web interface. Until now an update required the server console (SSH). No manual changes to the database, API, or configuration are required.

## 1. Update the panel from the web UI

A new **Panel updates** card is available in **Settings → Maintenance**:

* **Pick a channel.** *Main* tracks stable releases; *Beta* tracks the newest release, including pre-releases. On the Beta channel a freshly-published stable supersedes the matching beta — once `1.5.9` is out it is offered over `1.5.9-beta2`.
* **Check for updates.** Click **Check updates** to see the available version, whether it is stable or beta, and its release notes.
* **Update in one click.** Click **Update** and the panel downloads the new version, verifies it, replaces itself, and restarts on the new version — no console needed. Only newer versions are offered (no downgrades).
* **Nothing else changes.** Unlike the console updater, the web update never asks whether to change your admin login, password, or settings — your credentials and configuration are kept exactly as they were, and any required database migration runs automatically on restart.

## 2. Built to be safe

Applying an update runs freshly-downloaded code with the panel's privileges, so it is guarded carefully:

* **Admin-only, with confirmation.** Updating is available only to a signed-in administrator and additionally requires re-entering your password before it runs. You are warned that the panel and proxying will restart briefly.
* **Verified downloads.** The release artifact is fetched over verified HTTPS and checked against the release's published SHA-256 checksum before anything is replaced. A mismatch aborts the update and keeps the running version.
* **Safe to fail.** If an update cannot complete, the previous version keeps running. The replaced binary is backed up, and a freshly-installed version that repeatedly fails to start is rolled back automatically.
* **Audited.** Every check and update attempt — including rejected ones — is written to the audit log. Your password is never logged.

---

**Important:** no manual changes to the database, API, or configuration are required. The in-panel update applies to the standard Linux service install.

---

# Релиз v1.5.9-beta3: что нового

В этом релизе появилась возможность обновлять панель прямо из веб-интерфейса. Раньше обновление требовало консоли сервера (SSH). Ручные изменения в базе данных, API или конфигурации не требуются.

## 1. Обновление панели из веб-интерфейса

В разделе **Настройки → Обслуживание** появилась карточка **Обновления панели**:

* **Выбор канала.** *Main* отслеживает стабильные релизы; *Beta* — новейший релиз, включая предварительные. На канале Beta вышедшая стабильная версия имеет приоритет над соответствующей бетой — после выхода `1.5.9` она предлагается вместо `1.5.9-beta2`.
* **Проверка обновлений.** Нажмите **Check updates**, чтобы увидеть доступную версию, её тип (стабильная/предварительная) и описание изменений.
* **Обновление в один клик.** Нажмите **Update** — панель скачает новую версию, проверит её, заменит себя и перезапустится уже на новой версии, без консоли. Предлагаются только более новые версии (понижение невозможно).
* **Больше ничего не меняется.** В отличие от консольного обновления, веб-обновление не спрашивает, менять ли логин, пароль или настройки — учётные данные и конфигурация сохраняются, а необходимые миграции базы данных применяются автоматически при перезапуске.

## 2. Сделано безопасно

* **Только для администратора, с подтверждением.** Обновление доступно только аутентифицированному администратору и требует повторного ввода пароля перед запуском. Вы предупреждаетесь, что панель и проксирование кратковременно перезапустятся.
* **Проверенная загрузка.** Артефакт скачивается по защищённому HTTPS и сверяется с опубликованной контрольной суммой SHA-256 до подмены. Несовпадение отменяет обновление и сохраняет работающую версию.
* **Безопасно при сбое.** Если обновление не удалось, продолжает работать прежняя версия. Заменяемый бинарь резервируется, а повторно не стартующая новая версия откатывается автоматически.
* **С аудитом.** Каждая попытка проверки и обновления — включая отклонённые — записывается в журнал аудита. Пароль никогда не логируется.

---

**Важно:** для использования новой версии не требуется вносить ручные изменения в базу данных, API или конфигурацию. Обновление из панели применимо к стандартной установке как Linux-служба.
