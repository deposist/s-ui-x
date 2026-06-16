# Release v1.5.9‑beta2: What’s New

A small follow‑up to v1.5.9‑beta1. No manual changes to the database, API, or configuration are required.

## 1. Frontend appearance fixes (CSS)

Two visual regressions from the v1.5.9‑beta1 settings‑UX update have been fixed so the interface renders correctly:

* **Readable hint tooltips.** The new (i) field‑hint tooltips were almost unreadable in the dark theme — the text was too faint. Tooltips now use a solid dark background with light text and read well in both light and dark themes.
* **Field labels no longer cut off.** Some input labels were truncated to fragments like “Add…”, “P…”, “Do…”. Floating field labels are now shown in full.

## 2. Faster, more reliable install and update

* The installation and self‑update scripts could hang for up to ~15 minutes if a download mirror stalled. They now use a 20‑second timeout with automatic retries, so a stuck mirror is abandoned quickly and the download is retried against a working node.

---

**Important:** no manual changes to the database, API, or configuration are required to use the new version.


**RU**

# Релиз v1.5.9‑beta2: что изменилось

Небольшое дополнение к v1.5.9‑beta1. Ручные изменения в базе данных, API или конфигурации не требуются.

## 1. Исправления внешнего вида фронтенда (CSS)

Исправлены две визуальные регрессии из обновления UX настроек в v1.5.9‑beta1, чтобы интерфейс отображался корректно:

* **Читаемые подсказки‑тултипы.** Новые (i)‑подсказки к полям были почти нечитаемы в тёмной теме — текст был слишком блёклым. Теперь тултипы используют плотный тёмный фон со светлым текстом и хорошо читаются и в светлой, и в тёмной теме.
* **Подписи полей больше не обрезаются.** Некоторые подписи полей обрезались до фрагментов вроде «Add…», «P…», «Do…». Теперь плавающие подписи полей отображаются полностью.

## 2. Быстрее и надёжнее установка и обновление

* Скрипты установки и самообновления могли зависать до ~15 минут, если зеркало загрузки застревало. Теперь они используют таймаут 20 секунд с автоматическими повторами, поэтому застрявшее зеркало быстро отбрасывается, а загрузка повторяется с рабочего узла.

---

**Важно:** для использования новой версии не требуется вносить ручные изменения в базу данных, API или конфигурацию.
