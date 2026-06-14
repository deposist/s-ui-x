# Release Notes: Unreleased

## Fix: badCSR for IP-address TLS certificates; issuance moved to terminal menu

This patch fixes a critical issue that prevented Let's Encrypt from issuing
TLS certificates for bare IP addresses, and moves the issuance workflow from
the Settings web UI into the terminal management menu (item 20). The in-panel
auto-renewal cron is unchanged and now actually works with the CSR fix applied.
No database migration. No configuration changes.

## What changed

### Fix: badCSR — IP address must not appear in CSR Common Name

Production logs showed every IP-certificate issuance rejected by Let's Encrypt:

```
urn:ietf:params:acme:error:badCSR :: CSR contains IP address in Common Name
```

**Root cause.** The previous code called `client.Certificate.Obtain(ObtainRequest{Domains:[]string{ip}})`.
Internally, lego copies the first "domain" string verbatim into the CSR
`Subject.CommonName`. RFC 8738 (TLS certificates for IP addresses) requires the
IP to appear **only** in the `SubjectAltName` extension as an `iPAddress` entry,
with an **empty** Common Name. Let's Encrypt enforces this strictly.

**Fix.** A new `buildIpCSR(ip)` helper generates a fresh EC256 leaf key and
calls `certcrypto.CreateCSR(leafKey, CSROptions{Domain:"", SAN:[]string{ip}})`.
An empty `Domain` produces an empty CN; a parseable IP in `SAN` is routed to
`IPAddresses` by the lego library. The resulting CSR is parsed and passed to
`client.Certificate.ObtainForCSR(ObtainForCSRRequest{CSR, PrivateKey,
Profile:"shortlived", Bundle:true})`. Lego's `ExtractDomainsCSR` skips the
empty CN and picks up the IP from `IPAddresses`, then `createIdentifiers` emits
a `Type:"ip"` ACME identifier. The finalization step sends the custom CSR DER
to Let's Encrypt and returns the signed chain with the leaf private key PEM.
Verified against lego v4.35.2 source.

### IP cert issuance moved to terminal (s-ui.sh, item 20)

The **Settings → Maintenance** "IP certificate" card is removed. Issuance is
now a privileged terminal operation that integrates with the panel lifecycle:

- **Menu entry:** item 20 → option 5 — *Issue IP certificate (Let's Encrypt)*
- **Flow:** the menu stops the panel (freeing port 80 and giving the binary
  exclusive database access), runs `sui ip-cert issue`, then restarts the panel,
  which loads the new certificate from the paths written to `webCertFile`/
  `webKeyFile`.
- **CLI subcommand:** `sui ip-cert <issue|renew|status|disable>` with flags:
  - `issue -ip <ip> -email <email> [-port <port>] [-no-renew]`
  - `renew` — re-issue immediately regardless of expiry
  - `status` — print target IP, enabled flag, expiry, last-issued date
  - `disable` — turn off auto-renewal
- **Encryption key.** The management script sources
  `/etc/s-ui/secretbox.env` (which holds `SUI_SECRETBOX_KEY`) before calling
  the binary, so the CLI decrypts and re-encrypts the ACME account key
  identically to the running panel. The DB-derived fallback also works if the
  env file is absent.

### Web API and frontend removed

- API endpoints `POST /api/ip-cert/issue` and `GET /api/ip-cert/status` deleted.
- `IpCertificateService` removed from the API layer (`apiService.go`,
  `apiHandler.go`, `api/ip_certificate.go` deleted).
- `components/settings/IpCertificateCard.vue` and `types/ipcert.ts` deleted.
- `ipCert` i18n block removed from `en.ts` and `ru.ts`.
- `shield-check` Lucide icon removed from `plugins/lucideIcons.ts`.
- `MaintenanceTab.vue` no longer imports or renders the card.

### Auto-renewal preserved

The in-panel `@every 12h` cron job (`cronjob/certRenewJob.go`) is unchanged.
With the CSR fix applied it now correctly re-issues shortlived certificates
(~6.7 days) before the 72-hour expiry threshold and applies the result to the
panel HTTPS listener with a scheduled restart.

## Verification

- Backend: `go build ./...`, `go vet ./...`, and `go test ./... -p 1 -count=1`
  all green. New tests in `service/ip_certificate_acme_test.go` verify that
  `buildIpCSR` produces an empty Subject.CommonName, exactly one `IPAddresses`
  SAN entry, and a valid ECDSA self-signature; whitespace in the IP argument is
  trimmed. `TestIssueForCLIAppliesToPanelWithoutRuntime` confirms the nil-runtime
  CLI path writes `webCertFile`/`webKeyFile` without panicking.
- Frontend: `vue-tsc --noEmit`, `eslint` and `vitest` (locale parity + icon map
  scan) all green. `npm run build` clean.
- Shell: `bash -n s-ui.sh` — no syntax errors.

No database migration. The management script handles the panel stop/start cycle
around issuance. Auto-renewal binds port 80 while the panel is running — use a
custom challenge port if the panel occupies port 80. Let's Encrypt production is
used by default; set `SUI_ACME_DIR_URL` to point at staging/Pebble for testing.

---

# Примечания к релизу: Не выпущено

## Исправление: badCSR для TLS-сертификатов IP-адресов; выпуск перенесён в терминальное меню

Этот патч исправляет критическую ошибку, из-за которой Let's Encrypt отклонял
каждую попытку выпустить TLS-сертификат для голого IP-адреса, и переносит
процесс выпуска из веб-интерфейса Settings в терминальное меню управления
(пункт 20). Фоновое задание автоперевыпуска в панели не изменено и теперь
фактически работает с применённым исправлением CSR. Миграций базы нет.
Изменений конфигурации нет.

## Что изменилось

### Исправление: badCSR — IP-адрес не должен быть в Common Name сертификата

Журналы эксплуатации показывали, что Let's Encrypt отклонял каждую попытку
выпуска IP-сертификата:

```
urn:ietf:params:acme:error:badCSR :: CSR contains IP address in Common Name
```

**Причина.** Предыдущий код вызывал
`client.Certificate.Obtain(ObtainRequest{Domains:[]string{ip}})`. Внутри lego
копирует первую строку "domain" дословно в `Subject.CommonName` CSR. RFC 8738
(TLS-сертификаты для IP-адресов) требует, чтобы IP был **только** в расширении
`SubjectAltName` как запись `iPAddress`, а Common Name был **пустым**. Let's
Encrypt строго это проверяет.

**Исправление.** Новый вспомогательный метод `buildIpCSR(ip)` генерирует
свежий ключ EC256 и вызывает
`certcrypto.CreateCSR(leafKey, CSROptions{Domain:"", SAN:[]string{ip}})`.
Пустой `Domain` даёт пустой CN; распознаваемый IP в `SAN` направляется в
`IPAddresses` библиотекой lego. Полученный CSR парсится и передаётся в
`client.Certificate.ObtainForCSR(ObtainForCSRRequest{CSR, PrivateKey,
Profile:"shortlived", Bundle:true})`. `ExtractDomainsCSR` от lego пропускает
пустой CN и подхватывает IP из `IPAddresses`, затем `createIdentifiers` выдаёт
ACME-идентификатор `Type:"ip"`. Шаг finalization отправляет DER CSR в
Let's Encrypt и получает подписанную цепочку с PEM приватного ключа листа.
Проверено по исходникам lego v4.35.2.

### Выпуск IP-сертификата перенесён в терминал (s-ui.sh, пункт 20)

Карточка «IP certificate» в **Settings → Maintenance** удалена. Выпуск теперь
является привилегированной терминальной операцией, интегрированной с жизненным
циклом панели:

- **Пункт меню:** 20 → опция 5 — *Выпустить сертификат для IP (Let's Encrypt)*
- **Порядок работы:** меню останавливает панель (освобождает порт 80 и даёт
  бинарнику монопольный доступ к базе данных), запускает `sui ip-cert issue`,
  затем перезапускает панель, которая загружает новый сертификат по путям,
  записанным в `webCertFile`/`webKeyFile`.
- **Подкоманда CLI:** `sui ip-cert <issue|renew|status|disable>` с флагами:
  - `issue -ip <ip> -email <email> [-port <порт>] [-no-renew]`
  - `renew` — немедленный перевыпуск вне зависимости от срока действия
  - `status` — вывести целевой IP, флаг включения, срок, дату последнего выпуска
  - `disable` — отключить автоперевыпуск
- **Ключ шифрования.** Скрипт управления выполняет `source /etc/s-ui/secretbox.env`
  (содержит `SUI_SECRETBOX_KEY`) перед вызовом бинарника, поэтому CLI
  расшифровывает и перешифровывает ключ ACME-аккаунта точно так же, как и
  работающая панель. Запасной вариант на основе БД также работает, если файл
  env отсутствует.

### Удалён веб-API и фронтенд

- API-эндпоинты `POST /api/ip-cert/issue` и `GET /api/ip-cert/status` удалены.
- `IpCertificateService` удалён из API-слоя (`apiService.go`, `apiHandler.go`,
  `api/ip_certificate.go` удалён).
- `components/settings/IpCertificateCard.vue` и `types/ipcert.ts` удалены.
- Блок i18n `ipCert` удалён из `en.ts` и `ru.ts`.
- Иконка `shield-check` Lucide удалена из `plugins/lucideIcons.ts`.
- `MaintenanceTab.vue` больше не импортирует и не рендерит карточку.

### Автоперевыпуск сохранён

Фоновое задание в панели `@every 12h` (`cronjob/certRenewJob.go`) не изменено.
С применённым исправлением CSR оно теперь корректно перевыпускает shortlived-
сертификаты (~6.7 дня) до порога истечения в 72 часа и применяет результат к
HTTPS-листенеру панели с запланированным рестартом.

## Проверка

- Backend: `go build ./...`, `go vet ./...` и `go test ./... -p 1 -count=1` —
  всё зелёное. Новые тесты в `service/ip_certificate_acme_test.go` проверяют,
  что `buildIpCSR` даёт пустой `Subject.CommonName`, ровно одну запись
  `IPAddresses` в SAN и валидную ECDSA-самоподпись; пробелы в аргументе IP
  обрезаются. `TestIssueForCLIAppliesToPanelWithoutRuntime` подтверждает, что
  путь CLI без Runtime записывает `webCertFile`/`webKeyFile` без паники.
- Frontend: `vue-tsc --noEmit`, `eslint` и `vitest` (locale parity + скан
  карты иконок) — всё зелёное. `npm run build` чистый.
- Shell: `bash -n s-ui.sh` — синтаксических ошибок нет.

Миграций базы нет. Скрипт управления сам останавливает и запускает панель
вокруг выпуска. Автоперевыпуск занимает порт 80, пока панель работает —
используйте другой порт challenge, если панель занимает порт 80. По умолчанию
используется Let's Encrypt production; установите `SUI_ACME_DIR_URL` для
staging/Pebble при тестировании.
