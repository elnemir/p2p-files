# План реализации

Канонические фазы 0–7 и условия закрытия: [roadmap](../wiki/roadmap.md). **Фазы 1–6 закрыты по локальным development gates.** Фаза 6 включает backup → verified pre-restore backup → restore/session fencing → reconciliation/reopen → KEK recovery/incidents/checkpoints/UI/CLI. Реальное исполнение, проверки и следующие действия: [session](../wiki/session.md).

Фаза 7 сохраняет production TLS/ACME/trusted-proxy/CIDR, release security/DR/performance/browser/a11y/SBOM и независимую оценку рисков. Все 32 MUST сохранены. Прогресс учитывается по [traceability](test-matrix.md); локальные gates не заменяют инфраструктурную приёмку. Production deploy выполняется только по отдельному поручению.
