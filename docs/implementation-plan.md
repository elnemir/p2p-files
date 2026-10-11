# План реализации

Канонические фазы 0–7 и условия закрытия: [roadmap](../wiki/roadmap.md). **Фазы 1–6 закрыты по локальным development gates.** Фаза 6 включает backup → verified pre-restore backup → restore/session fencing → reconciliation/reopen → KEK recovery/incidents/checkpoints/UI/CLI. Реальное исполнение, проверки и следующие действия: [session](../wiki/session.md).

Фаза 7 сохраняет production TLS/ACME/trusted-proxy/CIDR, release security/DR/performance/browser/a11y/SBOM и независимую оценку рисков. Все 32 MUST сохранены. Прогресс учитывается по [traceability](test-matrix.md); локальные gates не заменяют инфраструктурную приёмку. Production deploy выполняется только по отдельному поручению.

Фаза 7 начата с [7.1 HTTPS ingress и network policy](../wiki/production-ingress.md): configuration generator, proxy/CIDR guards, settings/host recovery CLI и isolated TLS acceptance harness. Также реализован [controlled upgrade/downtime](../wiki/controlled-upgrade.md); его точные проверки и оставшиеся release gates — в session.

Завершающий технический набор фазы 7 реализован в [release engineering](../wiki/release-engineering.md): staticcheck/ESLint/generated types, fuzz, source/container/secret/license/SBOM, five-profile browser matrix, bounded multi-GB/export/fault/DR harness, compatibility и unpublished verifier bundle. Текущие результаты — session. Владелец подтвердил 2026-10-11: внешнее окружение не подготовлено, его приёмка заблокирована до подготовки.
