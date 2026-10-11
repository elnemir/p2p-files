# Решения, исключения, потенциальные конфликты

- Позднее решение имеет приоритет: после publish payload полностью immutable; раннее разрешение добавлять файлы отменено.
- QR bundle содержит секретные URLs в QR и mapping email; это чувствительный временный encrypted artifact, инвалидируется при изменении links.
- `integrity_failed`: при обычном TTL объект удерживается для диагностики; при revoke/block/delete аккаунта удаляется; при transfer/file freeze сохраняется до решения ADMIN.
- Restore: API tokens и recipient links восстанавливаются по backup (риск resurrection), но Web/recipient sessions инвалидируются, data plane блокируется до reconciliation и явного ADMIN approval.
- Доступ ADMIN к чужим exports/forensic не уведомляет владельца; действия фиксируются в tamper-evident audit.
- CLI имеет host/container trust boundary без дополнительного admin password и без обязательных интерактивных destructive confirmation flags.
- Промежуточный export snapshot должен сохранять **значения**, а не только IDs; гибридная модель должна гарантировать consistent view и crash safety.
- Обязательный backup PostgreSQL перед restore, но не автоматически перед startup migrations; restore-test вручную, без напоминаний.
- Отчёты имеют canonical report hash, document content hash (без собственного поля) и отдельный artifact SHA-256; не смешивать эти понятия.
- Ed25519 только SHA256SUMS; подписанный manifest хранится в ZIP без отдельного meta.json; ключ в PEM/SPKI, fingerprint SHA-256(DER/SPKI), сверка через независимый канал.
- Публичная `/verify-report` принимает reportId/revision/canonical hash, не подтверждает admin reports, при trash/purge — neutral not found.
- Нет IPv6 requirement, нет PWA, нет сторонней аналитики, нет внешнего proxy, нет auto production deploy.

## Требующие design review, не новых вопросов по умолчанию
1. Реальная гарантия «download completed» при обрывах TCP, reverse proxy buffering и Range — описать честно и протестировать.
2. Точный snapshot больших exports: атомарное materialization данных с bounded DB transaction; при невозможности предложить ADR, не симулировать согласованность.
3. Atomic publish + асинхронный SMTP health: publish не гарантирует доставку письма; явная модель partial delivery.
4. Строгий fail-closed crypto startup и изоляция crypto-context: разграничить глобальные критические и локальные ошибки.
5. «Бессрочный» security audit и key metadata: storage growth, operator legal obligations и external backup trust model.


## Зафиксированные design decisions 2026-10-10
По поручению владельца автономно уточнены [completion/Range/ZIP](../wiki/adr/001-download-completion.md), [freeze/expiry](../wiki/adr/002-retention-precedence.md), [export snapshot](../wiki/adr/003-export-snapshot.md), [restore backup](../wiki/adr/004-restore-gate.md), [crypto contexts](../wiki/adr/005-cryptography.md). Это решения текущего цикла, а не восстановленные ответы интервью. Обзор перечисленных выше design-review вопросов: [review](../wiki/design-review.md). Production/остаточные риски не одобрены автоматически.

## Реализация эксплуатационных решений 2026-10-11
[ADR-006](../wiki/adr/006-operations-recovery.md) фиксирует encrypted logical archive, отдельный неоткатываемый recovery control plane, обязательный pre-backup, runtime generation и explicit reopen. KEK rewrap сохраняет audit identity, restore в свежую БД требует новый master KEK; исторические backups сохраняют исторический KEK. ADMIN доступ к чужим exports/TTL реализован с reason/audit и непрерывным role/reauth guard. Локальные fault/concurrency/browser evidence и границы release приёмки — [development](../wiki/development.md).

## Начало фазы 7: ingress
[ADR-007](../wiki/adr/007-ingress-and-network-policy.md) реализует один доверенный Traefik hop, versioned network policy, generation fencing при её смене, host allowlist reset и offline generation production config. Это не принятие residual risks и не разрешение deployment.
