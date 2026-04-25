import './App.css';

const topChips = ['ПИЛОТ ОТКРЫТ', 'РУЧНОЙ ФОРМАТ', 'РЕАЛЬНЫЙ КЕЙС'];
const primaryNav = ['Пилот', 'Что это', 'Как работает', 'Демо'];
const contextNav = ['Небольшие команды', 'Реальная разработка', 'ИИ в разработке'];
const statusLines = ['Ручное сопровождение', 'Реальный кейс'];
const processSteps = ['ЗАПРОС', 'КОНТРАКТ', 'ИСПОЛНЕНИЕ', 'ПРОВЕРКА', 'ИТОГ'];

const heroBullets = [
  'Работаем с живым кейсом, где могут проявиться реальные ограничения.',
  'Держим под контролем границы задачи, исполнение, проверку и выводы.',
  'В конце фиксируем честный итог: что сработало, что нет и какой следующий шаг.',
];

const pilotFacts = [
  ['Формат', 'ручной пилот'],
  ['Контур', '1 команда / 1 репозиторий / 1 кейс'],
  ['Итог', 'честный вывод + следующий шаг'],
] as const;

const pilotFormat = ['короткий цикл', 'ручное сопровождение', 'реальный кейс'];
const fitList = [
  'уже используете ИИ в разработке',
  'есть живая продуктовая задача',
  'команда 5-30 человек',
  'готовы к эксперименту и честному выводу',
];
const pilotSteps = [
  'разбираем задачу',
  'формируем рабочий контракт',
  'ведём исполнение в заданных рамках',
  'проверяем результат',
  'фиксируем итог пилота',
];
const exclusions = [
  'не обещаем гарантированный результат',
  'не продаём «полный автопилот»',
  'не заменяем трекер задач или среду разработки',
  'не требуем долгого внедрения',
];

function MenuMark() {
  return (
    <span className="menuMark" aria-hidden="true">
      <span />
      <span />
      <span />
    </span>
  );
}

function PeopleIcon() {
  return (
    <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8.5" cy="8" r="3" />
      <circle cx="16" cy="9.5" r="2.3" />
      <path d="M3 18c.8-2.8 3-4.5 5.5-4.5S13.2 15.2 14 18" />
      <path d="M14.5 18c.6-2.2 2.2-3.5 4-3.5s3.4 1.3 4 3.5" />
    </svg>
  );
}

function ClipboardIcon() {
  return (
    <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="5.5" y="4.5" width="13" height="16" rx="2" />
      <rect x="9" y="3" width="6" height="3.5" rx="1" />
      <path d="M9 11h6M9 14h6M9 17h4" />
    </svg>
  );
}

function XCircleIcon() {
  return (
    <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="8.5" />
      <path d="M9 9l6 6M15 9l-6 6" />
    </svg>
  );
}

function MailIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="3.5" y="5.5" width="17" height="13" rx="2" />
      <path d="M4 7l8 6 8-6" />
    </svg>
  );
}

function DottedList({ items }: { items: string[] }) {
  return (
    <div className="dottedList">
      {items.map((item) => (
        <div className="dottedItem" key={item}>
          <span className="dot" aria-hidden="true" />
          <span>{item}</span>
        </div>
      ))}
    </div>
  );
}

function App() {
  return (
    <div className="landingShell">
      <div className="landingFrame" data-screen-label="Goalrail Pilot Intake">
        <header className="topbar">
          <div className="topbarLeft">
            <MenuMark />
            <div className="brand mono" aria-label="Goalrail pilot">
              <span className="brandName">GOALRAIL</span>
              <span className="brandSlash">/</span>
              <span className="brandSub">ПИЛОТ</span>
            </div>
          </div>
          <div className="topChips" aria-label="Pilot status">
            {topChips.map((chip, index) => (
              <span className={index === 0 ? 'chip statusChip' : 'chip'} key={chip}>
                {chip}
              </span>
            ))}
          </div>
        </header>

        <div className="bodyGrid">
          <aside className="sidebar" aria-label="Разделы лендинга">
            <div>
              <div className="sidebarLabel">РАЗДЕЛЫ</div>
              <nav className="sideNav" aria-label="Основные разделы">
                {primaryNav.map((item, index) => (
                  <a className={index === 0 ? 'navItem active' : 'navItem'} href={`#${index === 0 ? 'pilot' : 'details'}`} key={item}>
                    <span className={index === 0 ? 'openDot' : 'spacer'} aria-hidden="true" />
                    {item}
                  </a>
                ))}
              </nav>
            </div>

            <div>
              <div className="sidebarLabel">КОНТЕКСТ</div>
              <nav className="sideNav" aria-label="Контекст пилота">
                {contextNav.map((item) => (
                  <a className="navItem" href="#fit" key={item}>
                    <span className="spacer" aria-hidden="true" />
                    {item}
                  </a>
                ))}
              </nav>
            </div>

            <section className="statusCard" aria-label="Формат пилота">
              <div className="statusHeader">
                <span className="statusTitle">ФОРМАТ</span>
                <span className="openBadge">ОТКРЫТ</span>
              </div>
              <div className="statusLines">
                {statusLines.map((line) => (
                  <div key={line}>{line}</div>
                ))}
              </div>
            </section>

            <div className="sidebarFoot">
              <div className="modeLabel">РЕЖИМ</div>
              <div className="modeValue">Публичная страница пилота</div>
            </div>
          </aside>

          <main className="mainGrid">
            <section className="card heroCard" id="pilot" aria-labelledby="hero-title">
              <div className="eyebrow">ПИЛОТНЫЙ ФОРМАТ</div>
              <h1 className="heroTitle" id="hero-title">
                Разработка с ИИ<br />под контролем
              </h1>
              <p className="heroSub">
                Goalrail помогает провести реальную задачу через рабочий контракт, исполнение в заданных рамках, проверку и честный итог пилота внутри вашей команды, а не в вакууме.
              </p>

              <div className="processStrip" aria-label="Goalrail pilot process">
                {processSteps.map((step, index) => (
                  <span className={index === processSteps.length - 1 ? 'processEnd' : undefined} key={step}>
                    {step}
                    {index < processSteps.length - 1 ? <span className="processArrow" aria-hidden="true">→</span> : null}
                  </span>
                ))}
              </div>

              <div className="pulseLine">
                <span className="pulseDot" aria-hidden="true" />
                <span>Пилот это управляемый эксперимент: без гарантии автоматического успеха, но с честной проверкой на реальном кейсе.</span>
              </div>

              <div className="divider" />

              <div className="heroBullets">
                {heroBullets.map((bullet) => (
                  <div className="heroBullet" key={bullet}>
                    <span className="smallDot" aria-hidden="true" />
                    <span>{bullet}</span>
                  </div>
                ))}
              </div>
            </section>

            <aside className="rightColumn" aria-label="Детали пилота">
              <section className="card sideCard" id="details">
                <div className="sideTitle">ФОРМАТ ПИЛОТА</div>
                <dl className="keyValue">
                  {pilotFacts.map(([key, value]) => (
                    <div className="keyValueRow" key={key}>
                      <dt>{key}</dt>
                      <dd>{value}</dd>
                    </div>
                  ))}
                </dl>
                <hr />
                <DottedList items={pilotFormat} />
              </section>

              <section className="card sideCard" id="fit">
                <div className="sideTitle">КОМУ ПОДОЙДЁТ</div>
                <DottedList items={fitList} />
              </section>

              <section className="card sideCard" id="demo">
                <div className="sideTitle">ДЕМО</div>
                <p className="demoText">Доступно интерактивное демо</p>
                <button className="outlineButton" type="button">Открыть демо</button>
              </section>
            </aside>

            <section className="triplet" aria-label="Pilot explanation cards">
              <article className="card tripletCard">
                <div className="tripletIcon"><PeopleIcon /></div>
                <h2 className="tripletTitle">Когда нужен Goalrail</h2>
                <div className="tripletSeparator" />
                <p className="tripletBody">
                  Когда ИИ уже помогает писать код, но команде нужен управляемый эксперимент: с ясной постановкой, границами задачи, проверкой и понятным следующим шагом.
                </p>
              </article>

              <article className="card tripletCard">
                <div className="tripletIcon"><ClipboardIcon /></div>
                <h2 className="tripletTitle">Что делаем в пилоте</h2>
                <div className="tripletSeparator" />
                <div className="stepsList">
                  {pilotSteps.map((step) => (
                    <div className="stepItem" key={step}>{step}</div>
                  ))}
                </div>
              </article>

              <article className="card tripletCard">
                <div className="tripletIcon"><XCircleIcon /></div>
                <h2 className="tripletTitle">Что не делаем</h2>
                <div className="tripletSeparator" />
                <div className="exclusionList">
                  {exclusions.map((item) => (
                    <div className="exclusionItem" key={item}>
                      <span aria-hidden="true">×</span>
                      <span>{item}</span>
                    </div>
                  ))}
                </div>
              </article>
            </section>

            <section className="card ctaCard" aria-labelledby="cta-title">
              <div className="ctaIcon"><MailIcon /></div>
              <div>
                <h2 className="ctaTitle" id="cta-title">Если вам это близко, оставьте почту</h2>
                <p className="ctaSub">Без длинной анкеты. Сначала поймём, есть ли хороший кейс для пилота.</p>
              </div>
              <form className="ctaForm" action="mailto:hello@goalrail.dev" method="post" encType="text/plain" aria-label="Форма заявки на пилот">
                <label className="srOnly" htmlFor="pilot-email">Рабочая почта</label>
                <input className="textInput" id="pilot-email" name="email" type="email" placeholder="ваша@компания.ru" autoComplete="email" />
                <button className="amberButton" type="submit">Обсудить пилот</button>
              </form>
              <p className="ctaNote">
                Без рассылок. Только по делу. <span aria-hidden="true">·</span> Прямая почта:{' '}
                <a href="mailto:hello@goalrail.dev">hello@goalrail.dev</a>
              </p>
            </section>
          </main>
        </div>
      </div>
    </div>
  );
}

export default App;
