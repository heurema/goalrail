# Первый вопрос был не про то, как заставить AI писать больше кода.

- ID: 2026-04-24-01
- Date: 2026-04-24
- Status: ready-for-review
- Channel: LinkedIn
- Campaign / narrative: Goalrail Origin Series
- Pillar: founder-journey
- Format: text
- Hook type: claim
- Canon refs:
  - `docs/product/GOALRAIL_PUBLIC_NARRATIVE.md`
  - `docs/product/GOALRAIL_PUBLIC_LANGUAGE.md`
  - `docs/product/GOALRAIL_OFFER.md`
  - `docs/product/GOALRAIL_PRICING_MODEL.md`
  - `docs/brand/SHORT_FORM_CONTENT_SYSTEM.md`
- Source refs:
  - `https://punks.run/journal`
  - `/Users/vi/personal/heurema/punk/site/src/data/journal.ts`
- First comment: `Источник journal: https://punks.run/journal`

## Draft

Первый вопрос был не про то, как заставить AI писать больше кода.

Он был про то, как доверять результату, если ты не читаешь каждую строку.

В 2025 Q1 мой процесс еще жил в IDE. Модель помогала искать решения. Я сам правил код, смотрел diff, запускал проверки и решал, можно ли мержить.

AI был ассистентом. Не отдельной моделью исполнения.

Потом появилась неприятная мысль.

Что если редактор перестанет быть главным местом проверки?

Сначала это звучало странно. Читать код казалось единственным ответственным способом. Но с coding agents объем изменений растет быстрее, чем внимание человека.

Значит, вопрос меняется.

Не "может ли модель написать код?"
А "может ли процесс произвести доверие?"

Первым ответом были тесты. Потом планирование. Потом specs. Потом contracts. Дальше появились gates, proof artifacts и идея runtime surface вокруг работы.

Из этой линии вырос Goalrail: берем бизнес-цель и не пускаем ее в исполнение, пока она не встала на rails. Не потому что процесс красивый. Потому что расплывчатая задача плюс автономное исполнение почти всегда дают drift.

Хочу разобрать эту историю серией постов. Первый кусок совсем базовый: AI как помощь внутри IDE.

Дальше будет интереснее, потому что вопрос уйдет из личного workflow в инфраструктуру команды.

Сейчас мы начинаем с русскоязычных небольших команд. Там проще честно проверить один реальный repo, быстро увидеть боль и повлиять на будущие defaults продукта.

Что вам нужно увидеть, чтобы доверить агентному workflow часть разработки без чтения каждой строки?

#AIDevTools #Разработка #BuildInPublic

## QA

- Character count: 1577 / 1200-1800
- Word count: 231
- AI typography: pass
- Banned phrases: pass
- Link check: pass, no URL in body
- Hashtag check: pass, 3 hashtags at end
- Voice drift: pass, mean sentence length 8.44 words
- Opening check: pass, strong claim
- Closing check: pass, open question
- No self-promotion: pass

## Pending ledger entry

```json
{
  "id": "2026-04-24-01",
  "date": "2026-04-24",
  "posted_at": null,
  "timezone": "Europe/Moscow",
  "title": "Первый вопрос был не про то, как заставить AI писать больше кода.",
  "type": "dev",
  "pillar": "founder-journey",
  "format": "text",
  "hook_type": "claim",
  "source": "original",
  "source_ref": "https://punks.run/journal",
  "char_count": 1577,
  "word_count": 231,
  "hashtags": ["AIDevTools", "Разработка", "BuildInPublic"],
  "first_comment": "Источник journal: https://punks.run/journal",
  "url": null,
  "engagement": {
    "d2_impressions": null,
    "d2_likes": null,
    "impressions": null,
    "likes": null,
    "comments": null,
    "reposts": null,
    "clicks": null
  },
  "dms_from_post": 0,
  "metrics_updated": null
}
```
