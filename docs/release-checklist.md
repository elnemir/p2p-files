# Release checklist

> Нормативный production/release scope фазы 7. Локальные recovery drills фазы 6 не означают production DR acceptance. Этот документ не является отчётом о выполнении; текущее evidence и ограничения — [development](../wiki/development.md), порядок работы — [roadmap](../wiki/roadmap.md).

- [ ] Requirements ↔ implementation ↔ test matrix без необъяснённых gaps
- [ ] Architecture/threat/crypto reviews и ADR актуальны
- [ ] Go vet/staticcheck/security scan, TS strict/lint, formatting, secret scan
- [ ] Unit/integration/E2E, go test -race, fuzz/property, fault injection
- [ ] Browser/mobile/a11y/visual regression
- [ ] DB fresh+upgrade migration tests, OpenAPI/AsyncAPI/schema breaking diff
- [ ] Backup/restore, audit/crypto/storage checks и streaming smoke tests
- [ ] Docker build/container scan, SBOM, third-party license inventory
- [ ] CHANGELOG, release notes, docs links, CLI verifier checksums
- [ ] Нет открытых critical/high findings; residual risks согласованы владельцем
- [ ] Tag/GitHub Release только после зелёных gates; production deploy только по отдельной команде
