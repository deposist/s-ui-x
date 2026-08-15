# Release Notes: v1.5.12-beta5
Release date: 2026-08-15

v1.5.12-beta5 fixes a Doctor diagnostic that flagged WARP endpoints as unsupported by the official core. It also fixes a related check that rejected edits to WARP endpoints. No manual migration is required.

## WARP endpoint diagnostics

A WARP endpoint is stored in the panel under the type `warp` and written to sing-box as a WireGuard endpoint. The Doctor did not know this mapping, so it reported every WARP endpoint as an unsupported historical entity. The same mapping was missing from the save validation, so editing a WARP endpoint failed with an unsupported type error.

Both checks now resolve `warp` to `wireguard` before they consult the official capability list. Existing WARP endpoints no longer appear in the Doctor, and they can be edited normally.

## Update

Update the panel normally. You do not need to change the database or configuration.

# Заметки о релизе v1.5.12-beta5
Дата релиза: 2026-08-15

v1.5.12-beta5 исправляет диагностику Doctor, которая помечала WARP endpoint как неподдерживаемый официальным ядром. Исправлена и связанная проверка, которая отклоняла изменение WARP endpoint. Ручная миграция не нужна.

## Диагностика WARP endpoint

WARP endpoint хранится в панели с типом `warp`, а в sing-box передаётся как WireGuard endpoint. Doctor не знал об этом сопоставлении и отмечал каждый WARP endpoint как неподдерживаемую историческую сущность. Того же сопоставления не хватало в проверке сохранения, поэтому изменение WARP endpoint завершалось ошибкой неподдерживаемого типа.

Теперь обе проверки преобразуют `warp` в `wireguard` перед обращением к списку официальных возможностей. Существующие WARP endpoint больше не появляются в Doctor, и их можно менять обычным способом.

## Обновление

Обновите панель обычным способом. Менять базу данных или конфигурацию не нужно.

# 更新日志: v1.5.12-beta5
发布日期: 2026-08-15

v1.5.12-beta5 修复了 Doctor 诊断将 WARP endpoint 误报为官方核心不支持的问题，同时修复了另一个会拒绝修改 WARP endpoint 的检查。无需手动迁移。

## WARP endpoint 诊断

WARP endpoint 在面板中以 `warp` 类型保存，并作为 WireGuard endpoint 写入 sing-box。Doctor 不知道这个映射，因此把每个 WARP endpoint 都报成不受支持的历史实体。保存校验里也缺少同样的映射，导致修改 WARP endpoint 时报类型不支持的错误。

现在两个检查都会在查询官方能力列表前把 `warp` 解析为 `wireguard`。现有的 WARP endpoint 不再出现在 Doctor 中，并且可以正常修改。

## 更新

按正常方式更新面板。不需要修改数据库或配置。
