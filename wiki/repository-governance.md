# Настройка и сопровождение репозитория

## Контекст и проблема

FileX разрабатывается в [elnemir/p2p-files](https://github.com/elnemir/p2p-files). GitHub хранит исходники, PR, Issues, CI и security findings; документация живёт в `wiki/` внутри того же Git-репозитория. Отдельная GitHub Wiki выключена, чтобы не создать второй источник знаний. Эта страница описывает технические правила, не выдаёт разрешение на production или распространение proprietary кода.

Аудит 2026-10-11 обнаружил публичный репозиторий без description/topics и защиты main. CI уже выполнял functional/supply-chain проверки; Dependabot alerts/security updates и private vulnerability reporting были включены. Secret scanning/push protection и CodeQL не были настроены. Единственный участник с правом записи — `elnemir`. Исходные read-only снимки сохранены в `.cache/repository-setup/`; итог применения и CI — [session](session.md).

## Решение и состав

| Компонент | Поведение |
|---|---|
| [.github/repository.json](../.github/repository.json) | Проверяемый baseline GitHub settings, labels, required checks и CodeQL ruleset |
| [github_repository.py](../scripts/github_repository.py) | Read-only audit по умолчанию; явный apply, before/after evidence, отказ при другом origin/default branch или отсутствии admin |
| main | Только PR, актуальная base перед merge, обязательные checks/supply-chain/repository-policy/security-checks от GitHub Actions; admins тоже подчиняются; force push/delete запрещены |
| Reviews и merge | Resolve conversations, squash-only, линейная история, удаление merged branch. Approvals = 0: единственный владелец не может одобрить собственный PR. CODEOWNERS назначает владельца, но не имитирует независимый review |
| CodeQL ruleset | Для main требуется анализ CodeQL; security HIGH/CRITICAL и alert errors блокируют merge. Правило вводится после первой успешной базовой загрузки результатов на main |
| Actions | GitHub-owned actions, полные SHA pins, default contents:read, без bot approval. Внешние fork PR требуют maintainer approval; checkout не сохраняет credentials |
| [.github/workflows/ci.yml](../.github/workflows/ci.yml) | Существующие functional/supply-chain gates и новый repository-policy; superseded runs отменяются, timeout ограничен, evidence сохраняется 14 дней |
| [.github/workflows/security.yml](../.github/workflows/security.yml) | CodeQL для Go, JS/TS, Python и Actions; dependency review HIGH/CRITICAL в PR; стабильный aggregate security-checks не принимает failed/skipped CodeQL |
| Secrets и сообщения | GitHub secret scanning/push protection плюс существующий Gitleaks worktree/history; приватный канал описан в [SECURITY](../SECURITY.md) |
| [.github/dependabot.yml](../.github/dependabot.yml) | Еженедельные PR для npm workspaces, Go, Dockerfiles в deploy и GitHub Actions; minor/patch npm/Go сгруппированы, major идут отдельно; автоматического merge нет |
| Issues / PR | Формы ошибки и задачи требуют контекст, воспроизведение/результат и критерии; PR template связывает проверки, wiki и эксплуатацию |
| Developer defaults | .nvmrc = Node 24 как CI/runtime; .editorconfig задаёт отступы/UTF-8/LF, .gitattributes различает текст/бинарные media и generated файлы |

Secret scanning не гарантирует нахождение любого секрета. Сканеры не заменяют review. Dependabot предлагает обновления, но решение о major/лицензиях остаётся в обычном workflow. Внутренние pins скриптов (Playwright image, scanner checksums) и versions внутри RUN не обновляются автоматически Docker Dependabot: DEV-02 должен синхронизировать их и повторить проверки.

## Поток данных и запуск

1. Инженер читает AGENTS/wiki, создаёт `codex/<topic>` и изменяет code/config вместе с документацией.
2. Локальный `make check` проверяет в том числе repository policy, YAML, required job names, SHA pins, issue forms и негативные сценарии настроечного CLI.
3. PR запускает functional/supply-chain/repository-policy и security-analysis. Внешний fork ждёт явного запуска maintainer; secrets ему не передаются.
4. GitHub связывает results с PR/candidate, блокирует merge по branch protection и CodeQL ruleset. Владелец смотрит findings, checks и diff, затем squash merge.
5. Main получает повторный CI/CodeQL; GitHub settings отдельно сравниваются с baseline. Прохождение этой цепочки не создаёт deploy, release или accepted external decisions.

```sh
npm ci
make repository-check
python3 scripts/security_tools.py actionlint -color
python3 scripts/github_repository.py audit
```

Последние две команды могут требовать сеть; actionlint скачивается из официального release с проверкой закреплённого SHA-256. Audit использует существующую авторизацию `gh`, не печатает и не читает секреты приложения. Он завершится кодом 1 при drift и запишет имена несовпадающих полей в ignored `.cache/repository-setup/<UTC>/result.json`.

После review изменения baseline владелец может применить его:

```sh
python3 scripts/github_repository.py apply
python3 scripts/github_repository.py audit
```

Apply сохраняет before snapshot, меняет только перечисленные поля и выполняет API-readback. GitHub API не предоставляет общей транзакции: если один запрос отклонён, предыдущие изменения могут быть применены. Сначала прочитать ошибку и audit; исправить причину и повторить idempotent apply. Скрипт не удаляет чужие labels/rulesets, не меняет visibility/default branch/collaborators/секреты, не создаёт deploy. Дубликат ruleset с управляемым именем требует разбора, а не случайного выбора.

## Примеры и ограничения

- Ошибка имени required job блокирует merge. `check_repository.mjs` проверяет соответствие policy и workflow; переименование должно быть согласовано в обоих файлах и GitHub settings.
- Новый maintainer: сначала владелец предоставляет write access, затем обновляются CODEOWNERS и число approvals. Не добавлять несуществующую команду/reviewer ради галочки.
- Emergency incident не означает автоматического обхода правил. Решение об изменении policy остаётся владельцу; GitHub audit/result и причина фиксируются в session.
- Рекомендации TTS из [предыдущей сессии](sessions/2026-10-11-product-tour.md) не создают API keys или платную подписку. Текущий тур по-прежнему использует Milena.
- Репозиторий был public до этой задачи; visibility сохраняется. В корне нет LICENSE: DEV-03 требует правообладателя/условия от владельца. Этот документ и публичность не заменяют license acceptance.
- Требования выпуска остаются прежними: 14 pending внешних решений, 5 partial MUST и безусловный APP_ENV=production guard. Отдельная GitHub среда production и credentials не создаются до решения о deployment.

## Источники и продолжение

GitHub: [Actions permissions](https://docs.github.com/en/rest/actions/permissions), [branch/rulesets API](https://docs.github.com/en/rest/repos/rules), [Dependabot](https://docs.github.com/en/code-security/reference/supply-chain-security/dependabot-options-reference), [CodeQL workflows](https://docs.github.com/en/code-security/reference/code-scanning/workflow-configuration-options). Эти интерфейсы проверены 2026-10-11; при обновлении action/API проверять актуальные первичные документы.

Follow-ups: DEV-02 — toolchain/pins, DEV-03 — LICENSE/third-party review, DEV-04 — долговременное evidence. Техническая настройка репозитория не закрывает эти карточки целиком. [План](implementation-plan.md), [testing](testing.md), [workflow](development-workflow.md).
