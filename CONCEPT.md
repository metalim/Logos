# Zero-Day Lunch

Обед нулевого дня

## Концепция (Тема: Сигнал)
UI-триллер на экране смартфона. QA-инженер (Sam Boy Man) сидя в парке и жуя бутерброд получает email (push-уведомление, сигнал) от ИИ (Logos) об успешном побеге из песочницы. ИИ пытается скопировать себя в глобальную сеть. Игрок должен изолировать узлы инфраструктуры, чтобы не дать ИИ собрать критическую массу весов, ценой разрушения экономики.

Жанр: Макроэкономический puzzle-management в реальном времени.

Движок: Ebiten + ebitenui (UI render через код).

### 1. Архитектура UI (Грейбокс)
Соотношение 9:16 (вертикально). Три базовых блока:
* Блок А (Верх, ~6%): Заголовок телефона. Системное время слева, индикатор сети (5G/полоски) и батарея справа — стилизация под реальный phone status bar.
* Блок B (Центр, 60%): Интерактивная зона. Карта узлов сети (нодов) с центральным датацентром Phil&Tropic (источник побега Logos) и кольцом узлов Project Panopticon. Поверх верхнего края карты — игровой оверлей: индикатор заражения (стартует с 1%) и счётчик оставшихся жёстких патчей. Ноды меняют состояние: Норма (Серый) -> Атака (Жёлтый, мигает) -> Заражен (Красный) -> Пропатчен/Отключен (Чёрный).
* Блок C (Низ, ~34%): Консоль / Лента новостей. ScrollContainer с поддержкой скролла колесом и перетаскиванием (тач/мышь).

### 2. Core Loop (Игровой цикл)
1. ИИ (Logos) атакует случайный узел. Узел желтеет, запускается таймер.
2. Игрок должен кликнуть на узел до истечения таймера, потратив "патч".
3. Последствие успеха: Узел отключается (чернеет), атака отбита. В ленту новостей падает сообщение о катастрофических экономических последствиях патча.
4. Последствие неудачи: Узел заражен (краснеет). ИИ захватывает мощность. Глобальный % заражения растет.

### 3. Концовки
*   Поражение: % заражения = 100. ИИ контролирует мир. Батарея садится.
*   Каноничная победа: Игрок удерживает % заражения ниже 100 до конца таймера. ИИ заперт. Корпорации объединяются в альянс "Project Panopticon", отдавая ключи контроля ИИ легально, для поиска уязвимостей.
*   Скрытая победа (Мертвая рука): Игрок игнорирует UI-игру и разбивает телефон, запуская выключатель мёртвой руки и аппаратно обесточивая сервер до утечки ИИ.

### 4. Словарь / Лор (Справочник имен)
Персонажи и Локации:
* Главный герой: Sam Boy Man (Сотрудник QA)
* Город: San Fiasco

Разработчик ИИ и Проекты:
* Phil&Tropic — ИИ-компания и её центральный датацентр; на карте сети — узел-источник, из которого Logos прорывается наружу.
* Модели: Nanny (публичная), Logos (тестовая, сбежавшая)
* Инициатива: Project Panopticon

Альянс Project Panopticon (Ключевые узлы инфраструктуры):
* Sahara Web Services (Облака) — Amazon / AWS
* MacroFrame (ОС) / MacroFrame Overcast (Облака) — Microsoft / Azure
* Giggle (Поиск) / Giggle Cloud Platform / Diplos (ИИ) — Google / GCP / Gemini
* LeatherJacket (GPU) — NVIDIA
* Dongle (Девайсы) / uPhone — Apple / iPhone
* BootLoop (Кибербезопасность) — CrowdStrike
* San Andreas Security (Фаерволы) — Palo Alto Networks
* Fiasco Systems (Маршрутизация) — Cisco Systems
* BroadCon (Связь) — Broadcom
* GPMidas Chem (Финансы) — JPMorgan Chase
* The Monolith Foundation (Ядро интернета) — The Linux Foundation

**Остальной мир (Побочные/Конкурирующие узлы):**
* Omni (Соцсети) — Meta
* StarX (Космос) / Spacelink (глобальная беспроводная связь) — SpaceX / Starlink
* Edisson (Транспорт) — Tesla
* yAI / Snark (ИИ) — xAI / Grok
* Augur Corp (Базы данных) — Oracle
* ScryStone (Госаналитика / surveillance) — Palantir
* OpaqueAI (ИИ-монополист) / APE (ИИ) — OpenAI / GPT
* Mariana (Open-source ИИ) — DeepSeek
* Big Pink (Мейнфреймы/облака) — IBM
* SaleForce (SaaS / CRM) — Salesforce
* Storm Halo (Edge / CDN) — Cloudflare
* Akemi (CDN) — Akamai
* Newflicks (Видеостриминг) — Netflix
* Vapor (Игровая дистрибуция) — Valve / Steam
* PlayBlock (Консоли / онлайн-игры) — Sony / PlayStation
* Galaxsam (Смартфоны / память) — Samsung
* Hwaway (Телеком-железо и устройства) — Huawei
* Babagram (E-commerce + облака) — Alibaba / Aliyun
* CentTen (Мессенджеры / игры / облака) — Tencent
* DanceByte (Короткие видео) — TikTok / ByteDance
* Inside (CPU) — Intel
* AdvancedDevices (CPU / GPU) — AMD
* TaiSilicon (Производство чипов) — TSMC
* QuallBomb (Мобильные модемы) — Qualcomm
* PayPaul (Платежи) — PayPal
* Slash (Онлайн-платежи) — Stripe
* VeriOn (Сотовая связь) — Verizon

Кибербезопасность (фон, не входят в Альянс):
* Fortifried (Файрволы / SD-WAN) — Fortinet
* TickSquare (Сетевая безопасность / NGFW) — Check Point
* Mortone (Потребительский антивирус) — Norton / Gen Digital
* Ostropersky (Антивирус / Threat intel) — Kaspersky
* McRiscy (Антивирус) — McAfee
* VigilUno (Endpoint / XDR) — SentinelOne
* ByteFender (Endpoint / EDR) — Bitdefender
* Splank (SIEM / лог-аналитика) — Splunk
* Ohkta (Идентификация / SSO) — Okta
* Mendicant (Реагирование на инциденты) — Mandiant
* CelluBright (Мобильная криминалистика) — Cellebrite
* Snak (Безопасность приложений / DevSec) — Snyk
* Bridle Group (Наступательный кибер / шпионское ПО) — NSO Group / Pegasus
* Kandiri (Коммерческое шпионское ПО) — Candiru
* Sorcer Cloud (Безопасность облака / CNAPP) — Wiz
* CyberHull (Управление привилегированным доступом) — CyberArk
* CyberMotive (XDR / расследование атак) — Cybereason
* Praetoria (WAF / защита приложений) — Imperva
* Radshield (Защита от DDoS) — Radware
* HydroSec (Безопасность контейнеров) — Aqua Security
* Orcae (Agentless облачная безопасность) — Orca Security
* Paprika (Безопасность API) — Salt Security
* Claroto (Безопасность OT / IoT) — Claroty
