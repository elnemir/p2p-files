# ADR-009: доказательства релиза и исправленные runtime images

## Контекст
Фаза 7 должна закрывать проверяемые требования релиза, а не только наличие команд проверки. Исходные upstream images оказались уязвимыми даже при зелёных npm audit/govulncheck приложения. При этом независимый review, лицензии, реальные окружения и residual risks требуют внешнего подтверждения.

## Решение
1. Проверять source dependencies и каждый из пяти итоговых runtime images отдельно. Source/image gates fail closed на HIGH/CRITICAL, включая уязвимости без доступного исправления.
2. Сохранить стек и major versions. PostgreSQL entrypoint получает совместимый privilege-drop helper su-exec вместо gosu со старой Go stdlib. Redis получает исправленные OpenSSL APK packages. Traefik собирается из полного официального tree v3.7.14, закреплённого commit/archive SHA, с Go 1.26.9 и x/net 0.60.0. Web runtime не содержит package managers. Полные SBOM и tests требуются для итоговых образов, а не только баз.
3. Передавать production generator пять проверенных image digests. Генерация профиля, сборка и сканирование не означают публикацию/deploy.
4. `scripts/acceptance.py run <gate>` записывает exit status, время, профиль, commit и SHA-256 исполняемых исходников/config/contracts/CI/visual baselines. Изменение источников во время gate запрещает зачесть его. Публикация результатов в wiki/docs/README/CHANGELOG/MEMORY не меняет fingerprint; Markdown продолжает проверяться на каждом push.
5. `make release-check` требует свежие совпадающие gates, полный browser matrix, multi-GB profile, все 32 verified MUST, чистое дерево и все внешние решения. `contracts/release-acceptance.json` исходно содержит только pending. Наличие reviewer/evidence — проверка полноты записи, не цифровая подпись или независимое доказательство согласия; менять статус можно только после реального подтверждения владельца/reviewer.

## Поток и примеры
`python3 scripts/acceptance.py run images` → build exact Dockerfiles → Trivy JSON/CycloneDX → HIGH/CRITICAL check → source-bound gate result. `make release-check` агрегирует результаты и внешние решения, возвращает nonzero при любом пропуске и ничего не публикует. `python3 scripts/release_bundle.py` собирает unpublished review bundle offline verifier для Linux amd64/arm64, macOS amd64/arm64, Windows amd64 с SHA256SUMS и Go build metadata.

## Ограничения
История Git secret scan redacted; опубликованный deterministic crypto vector разрешён только точным path/value исключением. Scanners не заменяют security review. License inventory не принимает copyleft. Исправленные infrastructure builds становятся частью сопровождаемой поставки и должны пересобираться/сканироваться при advisories; source tree/Go/APK версии нельзя обновлять автоматически без gates. APP_ENV=production остаётся закрытым до принятия релиза. Настоящие ACME/Safari/devices/SMTP/S3/host/DR/escrow и R-01–06 остаются явными условиями, даже если весь локальный CI зелёный.
