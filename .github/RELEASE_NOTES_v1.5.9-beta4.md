# Release v1.5.9-beta4: What's New

A small follow-up to v1.5.9-beta3. No manual changes to the database, API, or configuration are required.

## 1. Failover group editor now works in the default interface

The new **Failover group** outbound added in v1.5.9-beta3 could not be configured in the default (Nexus) interface — selecting the *Failover* type opened an empty editor and still showed the Server address/port fields. The Failover editor now renders correctly in both the default and the classic interface: the ordered member list, the probe target and interval, the failback setting, and the *Add member* button all appear, and the server/port fields are hidden as they should be for a group.

---

**Important:** no manual changes to the database, API, or configuration are required.

---

# Релиз v1.5.9-beta4: что нового

Небольшое дополнение к v1.5.9-beta3. Ручные изменения в базе данных, API или конфигурации не требуются.

## 1. Редактор группы отказоустойчивости теперь работает в основном интерфейсе

Новый исходящий **Группа отказоустойчивости**, добавленный в v1.5.9-beta3, нельзя было настроить в основном интерфейсе (Nexus) — при выборе типа *Failover* открывался пустой редактор, а поля адреса/порта сервера всё равно показывались. Теперь редактор отображается корректно и в основном, и в классическом интерфейсе: упорядоченный список участников, адрес и интервал проверки, настройка возврата и кнопка *Добавить участника* — на месте, а поля сервера/порта скрыты, как и должно быть для группы.

---

**Важно:** для использования новой версии не требуется вносить ручные изменения в базе данных, API или конфигурации.
