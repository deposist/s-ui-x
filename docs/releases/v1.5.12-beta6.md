# Release Notes: v1.5.12-beta6
Release date: 2026-08-17

v1.5.12-beta6 contains no feature changes. It removes a leftover reference to a sing-box fork from the protocol capability notes for WireGuard. No action is required.

## Protocol notes cleanup

The documentation matrix of supported protocols is generated from a manifest kept in the panel. The WireGuard entry carried a remark about a forked core with extended WireGuard support. The panel uses only the official sing-box core, so the remark pointed at software this project does not distribute. The note now says only that the WireGuard endpoint requires the standard wireguard build tag.

## Update

Update the panel normally. You do not need to change the database or configuration.

# Заметки о релизе v1.5.12-beta6
Дата релиза: 2026-08-17

В v1.5.12-beta6 нет новых возможностей. Из описания WireGuard в матрице протоколов убрана отсылка к форку sing-box, которым эта панель не пользуется. Никаких действий не требуется.

## Чистка описаний протоколов

Матрица протоколов в документации генерируется из манифеста, который хранится в панели. У записи WireGuard была пометка о форке ядра с расширенной поддержкой WireGuard. Панель работает только на официальном sing-box, поэтому пометка указывала на программное обеспечение, которое этот проект не распространяет. Теперь в заметке сказано лишь, что для WireGuard endpoint нужен стандартный build tag wireguard.

## Обновление

Обновите панель обычным способом. Менять базу данных или конфигурацию не нужно.

# 更新日志: v1.5.12-beta6
发布日期: 2026-08-17

v1.5.12-beta6 不包含功能变化。协议矩阵中 WireGuard 的说明里残留了一条指向本面板并未使用的 sing-box 分支的引用，现已移除。无需任何操作。

## 协议说明清理

文档中的协议矩阵由面板维护的清单生成。WireGuard 条目带有一条关于扩展 WireGuard 支持的内核分支的备注。面板只使用官方 sing-box 内核，这条备注指向的是本项目并不分发的软件。现在该说明只指出 WireGuard endpoint 需要标准的 wireguard build tag。

## 更新

按正常方式更新面板。不需要修改数据库或配置。
