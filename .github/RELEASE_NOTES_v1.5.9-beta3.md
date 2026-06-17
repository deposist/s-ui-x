# Release v1.5.9-beta3: What's New

This release adds two improvements: a one-click **panel update from the web interface** (no SSH), and **automatic failover between outbounds** so your traffic keeps flowing when a connection drops. No manual changes to the database, API, or configuration are required.

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

## 3. Automatic outbound failover

A new **Failover group** outbound keeps your traffic flowing when an upstream goes down:

* **Set a priority order.** Create a *Failover group* (a new outbound type) and add your outbounds in order — the first is the primary, the rest are backups.
* **Automatic switching.** The panel probes each member over HTTPS (you choose the target host or IP and the interval; 30 seconds by default). When the active member stops responding, traffic moves to the next working backup automatically — no manual action.
* **Automatic return.** When the primary comes back and stays healthy, traffic returns to it. A short confirmation delay keeps a flapping connection from causing constant switching.
* **Always-on fallback.** If every member is down, the group falls back to a direct connection when one exists (otherwise it holds the top member) and shows an *all members down* state.
* **Use it anywhere.** Select the group as **Routing → Default Outbound** or in any routing rule — it behaves like any other outbound. The Outbounds list shows which member is currently active.

---

**Important:** no manual changes to the database, API, or configuration are required. These features apply to the standard Linux service install.

---

# Релиз v1.5.9-beta3: что нового

В этом релизе два улучшения: обновление панели **прямо из веб-интерфейса** в один клик (без SSH) и **автоматическое переключение между исходящими**, чтобы связь сохранялась при падении канала. Ручные изменения в базе данных, API или конфигурации не требуются.

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

## 3. Автоматическое переключение исходящих (failover)

Новый исходящий типа **Группа отказоустойчивости** сохраняет связь, когда внешний канал падает:

* **Задайте порядок приоритета.** Создайте *группу отказоустойчивости* (новый тип исходящего) и добавьте исходящие по порядку — первый основной, остальные резервные.
* **Автоматическое переключение.** Панель проверяет каждого участника по HTTPS (вы выбираете адрес-цель — домен или IP — и интервал; по умолчанию 30 секунд). Когда активный участник перестаёт отвечать, трафик автоматически переходит на следующего рабочего резервного — без ручных действий.
* **Автовозврат.** Когда основной снова доступен и стабилен, трафик возвращается к нему. Небольшая задержка-подтверждение не даёт «мигающему» каналу вызывать постоянные переключения.
* **Резерв на крайний случай.** Если недоступны все участники, группа переходит на прямое соединение (direct), если оно есть (иначе удерживает старшего участника), и показывает состояние *все участники недоступны*.
* **Используйте где угодно.** Выберите группу в **Маршрутизация → Исходящий по умолчанию** или в любом правиле — она ведёт себя как обычный исходящий. В списке исходящих видно, какой участник активен сейчас.

---

**Важно:** для использования новой версии не требуется вносить ручные изменения в базу данных, API или конфигурацию. Эти возможности применимы к стандартной установке как Linux-служба.
