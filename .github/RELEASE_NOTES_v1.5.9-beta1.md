# Release v1.5.9‑beta1: What’s New

## 1. Addressing critical security issues

### Issue 1: Error in secure connection (TLS) setup

Previously, the system had a bug: when configuring a secure connection, the program could suddenly “freeze” if the client‑side and server‑side settings didn’t match. This caused some settings to be saved while others weren’t, leading to a desynchronisation between the core application and the database.

**What’s been done:**
* the program now handles mismatched settings correctly;
* if an error occurs, all changes are rolled back — data remains consistent.

### Issue 2: Insecure file downloads

Installation and update scripts downloaded files without verifying the connection’s authenticity. An attacker could intercept the traffic, replace the downloaded file, and even tamper with its checksum — the system would still accept the fake file as genuine. This could lead to malicious code execution with administrator privileges.

**What’s been done:**
* TLS certificate verification is now enforced for every file download;
* files are downloaded to a temporary location and then replaced atomically — this eliminates the risk of file substitution during updates.

## 2. Supply‑chain security enhancements

To prevent component substitution during build and installation, the following measures have been implemented:

* **Libcronet library** (used in Docker images and Windows archives) is no longer downloaded from the generic “latest releases” page. Instead, it’s fetched from a specific, immutable link pointing to a particular release. Before use, it’s verified via SHA‑256 checksum for each architecture.
* **Docker images** now use fixed digests instead of mutable tags;
* **GitHub Actions** (automated build and test workflows) are tied to specific code versions (via full commit SHA), not branches that can be updated;
* **The s‑ui.sh script** no longer pipes a remote installer directly into a root shell without security checks. Silent background auto‑upgrade of acme.sh has also been disabled.

## 3. Fixing availability issues

* **Database migration error (from version 1.2 to 1.3).** Previously, migration could fail with a “record not found” error on a panel that never stored a raw config string. This blocked updates and backup restoration. Now, missing strings are handled gracefully — migration continues.
* **Listening address issue.** After restoring from a backup on a new server, the program might try to use an address unavailable on that machine. Previously, it silently switched to all network interfaces — including public ones, which was dangerous. Now, in such cases, it falls back to the loopback interface.

## 4. Correctness and performance improvements

* **CSRF protection.** The auto‑logout endpoint now uses a secure POST method instead of the vulnerable GET method.
* **Admin name change.** Empty and duplicate names are now rejected.
* **Traffic plan refunds.** Refunding an older order no longer resets consumption for the current billing period.
* **Periodic reset.** The risk of misconfiguration that could reset client counters every minute has been limited.
* **Subscription loading.** The external subscription loading path now uses a central SSRF validator that blocks dangerous IP ranges (e.g., CGNAT).
* **Race condition fixes.** Issues with concurrent hot reloads from deplete‑cron and full core restarts have been resolved.
* **Auto‑renewal of IP certificate.** Fixed a bug where the panel restart could be silently skipped.

### Performance improvements:
* a composite `stats` index has been added — it matches dashboard queries;
* WebSocket broadcasts are now serialized once (not per connection);
* the subscription server now compresses responses using gzip;
* the new `stats` index is created automatically at startup — no manual database migration is required.

## 5. Frontend usability improvements (user interface)

* **Settings tooltips.** Every settings field (web server, subscription server, sing‑box core page, Telegram page) now shows:
  * a default or recommended value (placeholder);
  * an info tooltip (i‑tooltip) with field description.
* **Dropdown lists.** Fields with a limited set of values are now implemented as proper lists:
  * time zone — autocomplete with the full IANA list;
  * Clash API mode — dropdown list;
  * payment currencies — combobox.
* **Misleading labels fixed.** The routing label “Invalid IP Ranges” / “Invalid Source IPs” now correctly reflects its function (previously, it did the opposite of what the name implied).
* **Login form.** Errors are now displayed inline (under the input field), not just in a popup toast.
* **Localisation.** A number of hard‑coded English strings (Telegram transport labels, routing action cards, dashboard KPI labels) have been moved to the internationalisation system (i18n).

---

**Important:** no manual changes to the database, API, or configuration are required to use the new version.


**RU**

# Релиз v1.5.9‑beta1: что изменилось

## 1. Устранение серьёзных проблем с безопасностью

### Проблема 1: ошибка при работе с защищённым соединением (TLS)

Раньше в системе была ошибка: при настройке защищённого соединения программа могла внезапно «зависнуть», если настройки на стороне клиента и сервера не совпадали. Из‑за этого часть настроек сохранялась, а часть — нет. В результате основная часть программы работала не так, как записано в базе данных — возникала рассинхронизация.

**Что сделано:**
* программа теперь корректно обрабатывает несовпадающие настройки;
* если возникает ошибка, все изменения отменяются — данные остаются согласованными.

### Проблема 2: небезопасная загрузка файлов

Скрипты установки и обновления загружали файлы без проверки подлинности соединения. Злоумышленник мог перехватить трафик, подменить загружаемый файл и даже его контрольную сумму — система всё равно приняла бы поддельный файл за настоящий. Это могло привести к запуску вредоносного кода с правами администратора.

**Что сделано:**
* теперь при загрузке файлов всегда проверяется подлинность соединения (TLS);
* файлы скачиваются во временный файл, а затем заменяются атомарно — это исключает подмену в процессе обновления.

## 2. Защита цепочки поставок (supply‑chain security)

Чтобы исключить подмену компонентов при сборке и установке, были приняты следующие меры:

* **Библиотека libcronet** (используется в Docker‑образах и Windows‑архивах) теперь загружается не с общей страницы последних релизов, а по конкретной, неизменной ссылке на определённый релиз. Перед использованием её проверяют по контрольной сумме (SHA‑256) для каждой архитектуры.
* **Docker‑образы** теперь используют фиксированные идентификаторы (digest), а не теги, которые могут меняться.
* **GitHub Actions** (автоматизированные процессы сборки и тестирования) привязаны к конкретным версиям кода (по полному идентификатору коммита), а не к веткам, которые могут обновляться.
* **Скрипт s‑ui.sh** больше не передаёт удалённый установщик напрямую в командную оболочку с правами администратора без проверки безопасности. Также отключён автоматический фоновый апгрейд acme.sh.

## 3. Исправление ошибок доступности

* **Ошибка при обновлении базы данных (с версии 1.2 до 1.3).** Раньше миграция могла прерваться с ошибкой «запись не найдена» на панели, которая никогда не сохраняла сырую строку конфигурации. Это блокировало обновления и восстановление из резервной копии. Теперь отсутствие строки обрабатывается корректно — миграция продолжается.
* **Проблема с адресом прослушивания.** После восстановления из бэкапа на новом сервере программа могла пытаться использовать адрес, который недоступен на этой машине. Раньше она молча переключалась на все сетевые интерфейсы, включая публичные, что опасно. Теперь в таком случае она переключается на локальный интерфейс (loopback).

## 4. Улучшения корректности и производительности

* **Защита от CSRF‑атак.** Эндпоинт автоматического выхода теперь использует безопасный метод POST вместо уязвимого GET.
* **Смена имени администратора.** Запрещены пустые и дублирующиеся имена.
* **Возвраты трафиковых тарифов.** При возврате старого заказа больше не обнуляется потребление текущего расчётного периода.
* **Периодический сброс.** Ограничена возможность ошибочной настройки, которая могла обнулять счётчики клиента каждую минуту.
* **Загрузка подписок.** Путь загрузки внешних подписок теперь использует центральный валидатор, который блокирует опасные диапазоны IP‑адресов (например, CGNAT).
* **Исправление гонок.** Устранены проблемы с одновременной перезагрузкой и перезапуском ядра.
* **Автоперевыпуск IP‑сертификата.** Исправлена ошибка, из‑за которой мог пропускаться перезапуск панели.

### Улучшения производительности:
* добавлен композитный индекс `stats`, который соответствует запросам дашборда;
* WebSocket‑рассылки теперь сериализуются один раз (а не для каждого соединения);
* сервер подписок сжимает ответы с помощью gzip;
* новый индекс `stats` создаётся автоматически при старте — ручная миграция базы не требуется.

## 5. Улучшения удобства фронтенда (пользовательского интерфейса)

* **Подсказки в настройках.** Каждое поле настроек панели (веб‑сервер, сервер подписок, страница ядра sing‑box, страница Telegram) теперь показывает:
  * значение по умолчанию или рекомендуемое значение (placeholder);
  * подсказку (i‑тултип) с описанием поля.
* **Выпадающие списки.** Поля с ограниченным набором значений теперь реализованы как полноценные списки:
  * часовой пояс — автозаполнение по полному списку IANA;
  * режим Clash API — выпадающий список;
  * валюты платежей — комбинированный список (combobox).
* **Исправлены вводящие в заблуждение надписи.** Лейбл маршрутизации «Invalid IP Ranges» / «Invalid Source IPs» теперь правильно отражает свою функцию (раньше он делал противоположное тому, что подразумевалось в названии).
* **Форма входа.** Ошибки теперь отображаются прямо под полем ввода (inline), а не только во всплывающем уведомлении.
* **Локализация.** Ряд жёстко прописанных английских строк (лейблы Telegram, карточки действий маршрутизации, подписи KPI дашборда) перенесены в систему многоязычной поддержки (i18n).

---

**Важно:** для использования новой версии не требуется вносить ручные изменения в базу данных, API или конфигурацию.
