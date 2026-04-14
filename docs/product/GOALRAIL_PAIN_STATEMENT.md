# Goalrail — Pain Statement

## 1. Primary business pain

**AI ускорил исполнение, но не решил главную проблему: как перевести бизнес-задачу в общий, контролируемый и проверяемый инженерный контур.**

Сегодня у большинства команд:
- задача приходит в виде vague request
- ограничения и non-goals зафиксированы слабо или вообще не зафиксированы
- часть контекста живёт в локальных prompt'ах, IDE, чатах и заметках на машинах разработчиков
- AI и инженерия начинают execution быстрее, чем команда успевает согласовать цель, границы и проверку
- на выходе есть код, diff или PR, но нет одного общего объекта, в котором видно, **что именно делали, почему именно так и как это проверили**

## 2. Why this is not a made-up pain

### Adoption is already here
GitLab в 2024 Global DevSecOps Report пишет, что **78%** респондентов уже используют AI в software development или планируют сделать это в ближайшие два года. Там же **64%** респондентов говорят, что хотят консолидировать toolchain. Это важный сигнал: AI уже внутри delivery, а окружающий контур слишком фрагментирован.

### Business / tech misalignment is still expensive
BCG в 2024 пишет, что почти половина C-suite респондентов сказала: **больше 30%** технологических проектов у них идут с перерасходом и задержкой. Среди главных причин отдельно названа **lack of alignment between the technology and business sides** по поводу operational objectives программы.

### Trust remains weak
Stack Overflow в 2024 зафиксировал, что только **43%** респондентов доверяют точности AI tools, а почти половина профессиональных разработчиков считает, что AI плохо справляется со сложными задачами. То есть скорость растёт быстрее, чем доверие к результату.

### Toolchain and governance are still fragmented
GitLab отдельно отмечает рассинхрон между CxO и individual contributors по AI, рискам и обучению, а также реальный toolchain sprawl. Это значит, что компании уже не ищут ещё один isolated AI tool — им нужен более управляемый слой.

## 3. Product response

**Goalrail — это единый server-side источник истины для AI-assisted delivery.**

Он нужен, чтобы:
- бизнес задавал цель и ограничения не в пустоту, а в общий рабочий контур
- команда превращала vague request в shared working contract
- задачи, запуски, проверки и proof жили не на отдельных ноутбуках, а в общем управляемом слое
- бизнес и инженерия видели один и тот же объект до начала execution, во время него и после него

## 4. Short positioning version

### One-line pain
**Проблема не в том, как быстрее генерировать код. Проблема в том, как не потерять управляемость между постановкой задачи и изменением в проде.**

### One-line answer
**Goalrail даёт общий контракт и общий server-side контур между intent и delivery.**

## 5. What not to say

Не говорить:
- “мы даём ещё одного AI-агента”
- “мы делаем prompt tool”
- “мы заменим вам Jira / Linear”
- “мы гарантируем детерминированный AI delivery навсегда”

Говорить:
- “мы убираем разрыв между бизнес-задачей и инженерным результатом”
- “мы даём единый источник истины для AI-assisted delivery”
- “мы делаем intent, contract, execution и proof видимыми и управляемыми”
- “мы дополняем provider tools там, где у них сейчас нет общего team/business control plane”

## 6. Sources

1. GitLab — 2024 Global DevSecOps Report: https://about.gitlab.com/resources/developer-survey/2024/
2. GitLab — Survey Reveals Tension Around AI, Security, and Developer Productivity: https://about.gitlab.com/press/releases/2024-06-25-gitlab-survey-reveals-tension-around-ai-security-and-developer-productivity-within-organizations/
3. BCG — Software Projects Don’t Have to Be Late, Costly, and Irrelevant: https://www.bcg.com/publications/2024/software-projects-dont-have-to-be-late-costly-and-irrelevant
4. Stack Overflow — 2024 Developer Survey gap between AI use and trust: https://stackoverflow.co/company/press/archive/stack-overflow-2024-developer-survey-gap-between-ai-use-trust/
