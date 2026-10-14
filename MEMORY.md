# Контекст репозитория FileX

Каноническая база — [wiki/index.md](wiki/index.md); текущие [состояние](wiki/current-state.md), [сессия](wiki/session.md), [план продолжения](wiki/implementation-plan.md). Этот файл — навигация и постоянные инварианты, не отдельный журнал результатов.

На 2026-10-14 UX, SAML/OIDC/LDAP, white labeling, Node 26 и native WebDAV доставлены в main `28a6b4a`; [история последней доставки](wiki/sessions/2026-10-14-product-ux-delivered.md). Фазы 1–6 реализованы; технические проверки фазы 7 исторически завершены на кандидате `7dce226`. Внешняя приёмка заблокирована отсутствием окружения и решений reviewer/owner. 32 MUST сохранены, 5 partial, 14 внешних решений pending. Production startup отклоняется безусловно. [Тур](wiki/screen-tour.md) переснят 2026-10-14: 71 кадр, 26 глав, единый MP4 7:56 с ElevenLabs Liza. Актуальные результаты нового diff всегда проверять в session.

## Не менять без согласования

- Go/Next.js/PostgreSQL/Redis/Traefik/Compose/Local/S3, single origin. PG authoritative; Redis ephemeral.
- Immutable publish, TTL от publishedAt, hard deadline, server-observed one-time consume, atomic ZIP, leases/fencing, freeze/forensic.
- USER без recipient correlation/IP/UA; ADMIN audit/reauth отдельно. KEK вне БД; классы ключей; Instance Ed25519 подписывает SHA256SUMS; отдельный vendor Ed25519 для лицензий утверждён в ADR-016.
- Restore с verified pre-backup, сохранением ops control plane, новой generation/restart, reconciliation и явным reopen. Физически удалённый payload backup БД не возвращает.
- R-01–06/лицензии принимаются владельцем; production deployment — только отдельной командой.

Все решения — [decisions](wiki/decisions.md) и ADR в [индексе](wiki/index.md). Исходное интервью 1–978 отсутствует; текущие решения нельзя приписывать отсутствующему интервью. Исторические числа и CI находятся в [истории проверок](wiki/history/development-evidence-2026-10-11.md).

Текущий native WebDAV срез добавляет migration 0017, PG locks/conditions, owner draft чтение и atomic encrypted staging. Статус проверок/туров — только wiki/session.md; Windows ожидает стенда. Исторические числа выше не относятся к текущему diff.
