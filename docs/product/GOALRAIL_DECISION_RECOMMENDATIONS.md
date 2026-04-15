# Goalrail Decision Recommendations

> Recommended answers to the currently open structural questions.

## 1. ICP location
**Recommendation:** держать `ICP` отдельным файлом: `docs/product/GOALRAIL_ICP.md`.

Почему:
- qualification logic нужен отдельно от GTM copy
- ICP живёт дольше, чем campaign / landing / pitch wording
- deployment и sales используют этот файл как одинаковый reference

## 2. Pilot document split
**Recommendation:** держать `PILOT_MODEL` отдельным файлом: `docs/product/GOALRAIL_PILOT_MODEL.md`.

Почему:
- pilot — это не просто часть deployment
- pilot является главным коммерческим объектом раннего этапа
- его удобно использовать как отдельный sales / delivery reference

## 3. Index language
**Recommendation:** English filenames + Russian content, в духе текущего repo.

Почему:
- repo уже идёт по этому пути
- filenames остаются стабильными и удобными для tooling
- content быстрее писать и уточнять на русском для текущего этапа

## 4. Delivery mode
**Recommendation:** сначала `managed service`, позже `toolkit + guided setup`.

Почему:
- early deployments слишком хрупкие для self-guided motion
- нужен плотный feedback loop
- сначала надо productize playbook на живых пилотах

## 5. Required runtimes
**Recommendation:** обязательный initial support — `Codex` и `Claude Code`; `Gemini` — optional / experimental.

Почему:
- это наиболее полезный начальный coverage
- снижает initial complexity
- не мешает сохранить runtime-neutral posture

## 6. Proof policy
**Recommendation:** proof обязателен с первого дня, но в lightweight form.

Почему:
- иначе теряется ключевой wedge продукта
- proof не должен откладываться “на потом”
- минимальный proof можно сделать дешёвым, но всё равно обязательным

## 7. Pricing posture
**Recommendation:** `free diagnostic / qualification` + `paid pilot`.

Почему:
- qualification убирает no-fit без friction
- основной ценностный объём создаётся внутри pilot
- paid pilot лучше отфильтровывает слабую заинтересованность

## 8. Geography and language
**Recommendation:** RU-first market entry; структура документов сразу готовится к EN-localization later.

## 9. Landing type
**Recommendation:** сначала `lead capture page`, не interactive demo.

Почему:
- early commercial motion founder-led
- главное сейчас — собрать pilot-qualified conversations
- interactive demo можно добавить позже

## 10. Engagement shape
**Recommendation:** не длинный audit, а short fit-check + 2-week pilot baseline.

Почему:
- Goalrail должен оставаться productized deployment, а не bespoke consulting
- длительный аудит противоречит выбранному motion

## 11. Work mode
**Recommendation:** remote-first; onsite optional later.

## 12. Canon structure
**Recommendation:** один `docs/INDEX.md` + несколько канонических файлов, а не один giant master document.
