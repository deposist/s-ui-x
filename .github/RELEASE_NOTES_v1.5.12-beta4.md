# Release Notes: v1.5.12-beta4
Release date: 2026-08-15

v1.5.12-beta4 fixes a core restart failure that occurred when a WARP endpoint was selected as the Default outbound in Rules. No manual migration is required.

## WARP endpoint routing

Selecting a WARP endpoint as `route.final` no longer causes the core restart to fail with `default outbound not found: <tag>`. The panel keeps the existing WARP tag and writes the endpoint to sing-box as a WireGuard endpoint.

You do not need to recreate existing WARP endpoints. The fix covers both full configuration rebuilds and endpoint reloads.

## Update

Update the panel normally. You do not need to change the database or configuration.

# Заметки о релизе v1.5.12-beta4
Дата релиза: 2026-08-15

v1.5.12-beta4 исправляет сбой перезапуска ядра при выборе WARP endpoint как Default outbound в Rules. Ручная миграция не нужна.

## Маршрутизация через WARP endpoint

Если выбрать WARP endpoint в `route.final`, перезапуск ядра больше не завершается с ошибкой `default outbound not found: <tag>`. Панель сохраняет существующий WARP tag и передаёт endpoint в sing-box как WireGuard endpoint.

Существующие WARP endpoint не нужно создавать заново. Исправление охватывает полную сборку конфигурации и перезагрузку endpoint.

## Обновление

Обновите панель обычным способом. Менять базу данных или конфигурацию не нужно.

# 更新日志: v1.5.12-beta4
发布日期: 2026-08-15

v1.5.12-beta4 修复了在 Rules 中将 WARP endpoint 设为 Default outbound 后核心重启失败的问题。无需手动迁移。

## WARP 路由

将 WARP endpoint 设为 `route.final` 后，核心重启不再因 `default outbound not found: <tag>` 失败。面板会保留现有的 WARP tag，并将 endpoint 以 WireGuard 类型写入 sing-box 配置。

现有的 WARP endpoint 不需要重新创建。修复覆盖完整配置构建和 endpoint 重载。

## 更新

按正常方式更新面板。不需要修改数据库或配置。
