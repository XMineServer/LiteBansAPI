# LiteBans REST API — описание для фронтенда

Базовый префикс всех путей: `/api/v1`. Это read-only API (нет методов на создание/изменение данных).

Введены три уровня доступа: **аноним** (`/public/*`), **игрок** (`/player/*`, требует JWT) и
**модератор** (`/mod/*`, требует JWT + право `web.litebans.view.all`). Контекстный эндпоинт
`GET /punishments/{type}/{id}` доступен всем, но авторизуется по содержимому записи (см. ниже).

## Общие положения

- Все ответы — `application/json`.
- Все временные метки (`issuedAt`, `expiresAt`, `at`, `before`/`after`) — Unix-время в **миллисекундах**, целое число. Форматирование/локализация — на фронте.
- `reason`/`removed.reason` приходят уже очищенными от цветовых кодов Minecraft (`§x`, `&x`, `#RRGGBB`) — выводить как обычный текст. Перенос строки — обычный `\n` внутри строки.
- `id` наказания — либо число, либо (если на сервисе включена обфускация ID) непрозрачная строка-токен. **Фронт не должен полагаться на тип/формат `id`** — просто хранить как есть и передавать обратно в `GET /punishments/{type}/{id}` без изменений. Тип токена связан с конкретным `{type}` — нельзя подставить id, полученный из `bans`, в `mutes`.
- uuid принимается и в виде с тире, и без — но в ответах сервис всегда возвращает канонiчный вид с тире, в нижнем регистре.
- Все списочные эндпоинты используют **офсетную** пагинацию (`page`/`pageSize`) и возвращают `PunishmentList`.
- Авторизация — заголовком `Authorization: Bearer <JWT>`. JWT выпускается Identity Service (RS256), проверяются `iss` и `exp`; uuid игрока берётся из `sub`.

## Модели данных

### Moderator
```ts
type Moderator = {
  uuid: string | null;   // null, если выдано консолью
  name: string | null;
  isConsole: boolean;
};
```

### Removed
Присутствует только когда наказание было явно снято модератором/консолью (не при автоматическом истечении срока — в этом случае `removed` будет `null`, а `expired: true`).
```ts
type Removed = {
  by: Moderator;
  at: number;
  reason: string | null;
  expiredAutomatically: boolean; // всегда false в текущей реализации (см. ниже)
};
```

### Punishment (ban / mute / warning)
```ts
type Punishment = {
  id: number | string;
  type: "ban" | "mute" | "warning";
  playerUuid: string;
  reason: string;
  moderator: Moderator;
  issuedAt: number;
  expiresAt?: number;     // отсутствует, если permanent === true
  permanent: boolean;
  active: boolean;
  expired: boolean;
  ipBan: boolean;
  silent: boolean;
  serverOrigin: string | null;
  serverScope: string | null;
  removed?: Removed;       // отсутствует, если не было явного снятия
  acknowledged?: boolean;  // только для type === "warning"
};
```

### Punishment (kick)
У kick **нет** длительности и снятия — соответствующие поля в ответе вообще отсутствуют (не `null`, а ключ отсутствует в JSON):
```ts
type KickPunishment = {
  id: number | string;
  type: "kick";
  playerUuid: string;
  reason: string;
  moderator: Moderator;
  issuedAt: number;
  serverOrigin: string | null;
  // НЕТ: expiresAt, permanent, active, expired, ipBan, silent, serverScope, removed, acknowledged
};
```

Рекомендация для фронта: проверять `type` и использовать union-тип, не рассчитывать на присутствие полей длительности у kick. В списках, где `type` не указан в запросе (`/player/punishments/me`, `/mod/punishments/list`), `items` — это union всех 4 типов вперемешку, отличать по полю `type`.

### Player (только для `/public/lookup`)
```ts
type Player = {
  uuid: string | null;   // null для записей консоли
  name: string | null;   // null, если имя не резолвится
  isConsole: boolean;
  offlineMode: boolean;
};
```

### Списки и пагинация
```ts
type PunishmentList = {
  items: Punishment[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
};

type Stats = { bans: number; mutes: number; warnings: number; kicks: number };
```

### Ошибки
Любая ошибка — единый формат с соответствующим HTTP-кодом:
```ts
type ApiError = { error: string; message: string };
```
| HTTP | `error` | Когда |
|---|---|---|
| 400 | `INVALID_UUID` | некорректный формат uuid (не 32/36 hex-символов) |
| 400 | `INVALID_TYPE` | `{type}` не из набора `bans/mutes/warnings/kicks` |
| 400 | `INVALID_PARAMETER` | некорректная пагинация, булево значение, имя игрока, `type` вне разрешённого набора, или некорректная комбинация параметров `/public/lookup` |
| 401 | `UNAUTHORIZED` | требуется JWT, но он отсутствует/невалиден/истёк (`/player/*`, `/mod/*`) |
| 403 | `FORBIDDEN` | JWT валиден, но нет права `web.litebans.view.all` (`/mod/*`) |
| 404 | `NOT_FOUND` | наказание/игрок не найден; также используется для чужого непубличного наказания в `/punishments/{type}/{id}`, чтобы не раскрывать его существование |
| 503 | `SERVICE_UNAVAILABLE` | проблема с БД или с Authority Service (при проверке права на `/mod/*`) |
| 500 | `INTERNAL_ERROR` | непредвиденная ошибка сервиса |

## Эндпоинты

### `GET /api/v1/public/punishments`
Без авторизации. Публичные наказания — только тех типов, что разрешены конфигурацией сервиса (по умолчанию только `ban`).

Query-параметры (все опциональны):
| Параметр | Тип | Описание |
|---|---|---|
| `type` | string | один из разрешённых публичных типов; по умолчанию `ban`. Тип вне списка → `400 INVALID_PARAMETER` |
| `page` | int ≥ 1 | по умолчанию 1 |
| `pageSize` | int ≥ 1 | по умолчанию из конфига сервиса, обрезается верхним лимитом сервиса |
| `active` | bool | переопределяет дефолт видимости активных записей |
| `silent` | bool | переопределяет дефолт видимости тихих записей |
| `playerUuid` | uuid | фильтр по игроку |
| `moderatorUuid` | uuid | фильтр по модератору |

→ `200 PunishmentList`

```
GET /api/v1/public/punishments?page=1&pageSize=20&active=true
```

### `GET /api/v1/public/punishments/stats`
Без авторизации. Глобальная статистика (не персонализированная).

→ `200 Stats`. Query-параметров не принимает.

### `GET /api/v1/public/lookup`
Без авторизации. Резолвинг uuid↔имя по истории имён. Нужно передать **ровно один** из параметров:

| Параметр | Тип |
|---|---|
| `name` | string, 1–16 символов `[A-Za-z0-9_]` |
| `uuid` | uuid |
| `moderatorName` | string, тот же формат, что `name` |
| `moderatorUuid` | uuid |

`moderatorName`/`moderatorUuid` резолвятся идентично `name`/`uuid` — разница чисто смысловая (это два независимых, но одинаково устроенных критерия поиска). 0 или >1 параметров → `400 INVALID_PARAMETER`.

→ `200 Player` | `404 ApiError` (если ничего не резолвилось)

### `GET /api/v1/player/punishments/me`
Требует `Authorization: Bearer <JWT>`. Наказания **текущего** игрока (все 4 типа), uuid берётся только из JWT.

| Параметр | Тип | Описание |
|---|---|---|
| `type` | string | `ban`/`mute`/`warning`/`kick`; без него — все типы вперемешку |
| `moderatorUuid` | uuid | фильтр «кто выдал» |
| `before`/`after` | long (ms) | фильтр по времени выдачи (не курсор — пагинация всё равно офсетная) |
| `page`, `pageSize` | int | офсетная пагинация |

Параметр `playerUuid`, если передан, игнорируется — uuid всегда берётся из токена. `active`/`silent` неприменимы к kick-записям и игнорируются для них.

→ `200 PunishmentList` | `401 ApiError`

### `GET /api/v1/mod/punishments/list`
Требует `Authorization: Bearer <JWT>` и право `web.litebans.view.all`. Полный список наказаний всех игроков.

| Параметр | Тип | Описание |
|---|---|---|
| `type` | string | `ban`/`mute`/`warning`/`kick`; без него — все типы вперемешку |
| `playerUuid` | uuid | фильтр по игроку |
| `moderatorUuid` | uuid | фильтр по модератору (заменяет старый `issued-history`) |
| `active`, `silent` | bool | фильтры видимости; неприменимы к kick |
| `page`, `pageSize` | int | офсетная пагинация |

→ `200 PunishmentList` | `401 ApiError` | `403 ApiError` | `503 ApiError`

### `GET /api/v1/punishments/{type}/{id}`
`{type}` ∈ `bans | mutes | warnings | kicks`. Деталь одной записи по `id` (тому самому значению, что в `id` объекта Punishment). Авторизация — по содержимому, **после** загрузки записи:

- если `{type}` относится к публично разрешённым (например `bans`) — отдаётся без JWT;
- иначе (например `mutes`/`warnings`/`kicks`): нужен валидный JWT, и либо `playerUuid` записи совпадает с uuid из токена (своё наказание), либо у вызывающего есть право `web.litebans.view.all` (модератор); во всех остальных случаях — `404`, а не `403`, чтобы не подтверждать существование записи.

Фильтры видимости (`active`/`silent`) **не применяются** к детали — доступна запись в любом состоянии (снятая/тихая/неактивная), если прошла авторизацию.

→ `200 Punishment` | `404 ApiError`

## Примечания по интеграции

- **Видимость по умолчанию.** Если фронту нужно явно показать «все, включая снятые/тихие» — передавайте `active=false`/`silent=false`/оба параметра по необходимости, не полагайтесь на дефолт сервиса (он настраивается на деплое и может отличаться между средами).
- **`playerUuid` в Punishment — это просто строка**, без резолвленного имени. Чтобы показать имя игрока, фронту нужно отдельно вызвать `/public/lookup?uuid=...` (имя там может быть `null`, обрабатывать как "Unknown player").
- **`expiresAt` отсутствует у permanent-записей** — проверяйте `permanent === true`, не `expiresAt == null`, как основной признак "бессрочно" (оба варианта эквивалентны, но `permanent` — явный).
- **`removed` отсутствует и при активном наказании, и при истёкшем автоматически** — чтобы понять причину закрытия записи, используйте пару `(removed, expired)`: `removed` есть → снято явно; `removed` нет и `expired === true` → истекло само; `removed` нет и `expired === false` → ещё действует (или активный флаг `active === false` без даты снятия, что обычно означает рассинхрон данных на стороне плагина — на это стоит не полагаться при отображении).
- **404 на `/public/lookup` — это нормальный сценарий**, а не ошибка интеграции: означает, что для переданного имени/uuid в истории имён нет записей.
- **404 на `/punishments/{type}/{id}` для непубличного типа** может означать либо что записи действительно не существует, либо что она существует, но принадлежит другому игроку и у вас нет права её видеть — фронт не должен пытаться различать эти случаи.
