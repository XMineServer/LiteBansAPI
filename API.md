# LiteBans REST API — описание для фронтенда

Базовый префикс всех путей: `/api/v1`. Это read-only API (нет методов на создание/изменение данных).

## Общие положения

- Все ответы — `application/json`.
- Все временные метки (`issuedAt`, `expiresAt`, `at`, `before`/`after`/курсоры) — Unix-время в **миллисекундах**, целое число. Форматирование/локализация — на фронте.
- `reason`/`removed.reason` приходят уже очищенными от цветовых кодов Minecraft (`§x`, `&x`, `#RRGGBB`) — выводить как обычный текст. Перенос строки — обычный `\n` внутри строки.
- `id` наказания — либо число, либо (если на сервисе включена обфускация ID) непрозрачная строка-токен. **Фронт не должен полагаться на тип/формат `id`** — просто хранить как есть и передавать обратно в `GET /punishments/{type}/{id}` без изменений. Тип токена связан с конкретным `{type}` — нельзя подставить id, полученный из `bans`, в `mutes`.
- uuid принимается и в виде с тире, и без — но в ответах сервис всегда возвращает канонiчный вид с тире, в нижнем регистре.

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

Рекомендация для фронта: проверять `type` и использовать union-тип, не рассчитывать на присутствие полей длительности у kick.

### Player (только для `/players/lookup`)
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

type HistoryPage = {
  items: Punishment[];
  totalItems: number;
  cursors: { before: number | null; after: number | null };
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
| 400 | `INVALID_PARAMETER` | некорректная пагинация, булево значение, имя игрока или комбинация параметров `/players/lookup` |
| 404 | `NOT_FOUND` | наказание/игрок не найден |
| 503 | `SERVICE_UNAVAILABLE` | проблема с БД |
| 500 | `INTERNAL_ERROR` | непредвиденная ошибка сервиса |

## Эндпоинты

### `GET /api/v1/punishments/{type}`
`{type}` ∈ `bans | mutes | warnings | kicks`. Список с офсетной пагинацией, сортировка всегда по `issuedAt` убывающе.

Query-параметры (все опциональны):
| Параметр | Тип | Описание |
|---|---|---|
| `page` | int ≥ 1 | по умолчанию 1 |
| `pageSize` | int ≥ 1 | по умолчанию из конфига сервиса, обрезается верхним лимитом сервиса |
| `active` | bool | переопределяет дефолт видимости активных записей; **не действует** для `kicks` |
| `silent` | bool | переопределяет дефолт видимости тихих записей; **не действует** для `kicks` |
| `playerUuid` | uuid | фильтр по игроку |
| `moderatorUuid` | uuid | фильтр по модератору |

→ `200 PunishmentList`

```
GET /api/v1/punishments/bans?page=1&pageSize=20&active=true
```

### `GET /api/v1/punishments/{type}/{id}`
Деталь одной записи по `id` (тому самому значению, что в `id` объекта Punishment). Фильтры видимости **не применяются** — доступна всегда, включая снятые/тихие/неактивные.

→ `200 Punishment` | `404 ApiError`

### `GET /api/v1/punishments/stats`
→ `200 Stats`. Учитывает те же правила видимости (active/silent), что и дефолты списков; query-параметров не принимает.

### `GET /api/v1/players/{uuid}/history`
Вся история наказаний, **полученных** игроком (по всем 4 типам), без фильтров видимости — полный аудит. Курсорная пагинация.

| Параметр | Тип | Описание |
|---|---|---|
| `before` | long (ms) | вернуть записи строго раньше этого времени |
| `after` | long (ms) | вернуть записи строго позже этого времени |
| `pageSize` | int | размер страницы |

→ `200 HistoryPage`

Для подгрузки "глубже в историю" — следующий запрос с `before = cursors.before` текущей страницы. Для возврата к более новым записям — `after = cursors.after`.

### `GET /api/v1/players/{uuid}/issued-history`
То же самое, но наказания, **выданные** этим uuid как модератором. Те же параметры и формат ответа.

### `GET /api/v1/players/lookup`
Резолвинг uuid↔имя по истории имён. Нужно передать **ровно один** из параметров:

| Параметр | Тип |
|---|---|
| `name` | string, 1–16 символов `[A-Za-z0-9_]` |
| `uuid` | uuid |
| `moderatorName` | string, тот же формат, что `name` |
| `moderatorUuid` | uuid |

> ⚠️ В отличие от исходного ТЗ (где допускалась одновременная передача `name`/`uuid` и `moderatorName`/`moderatorUuid`), текущая реализация требует **ровно один** параметр за запрос и вернёт `400 INVALID_PARAMETER`, если передано 0 или больше 1. `moderatorName`/`moderatorUuid` резолвятся идентично `name`/`uuid` — разница чисто смысловая (это два независимых, но одинаково устроенных критерия поиска).

→ `200 Player` | `404 ApiError` (если ничего не резолвилось)

## Примечания по интеграции

- **Видимость по умолчанию.** Если фронту нужно явно показать «все, включая снятые/тихие» — передавайте `active=false`/`silent=false`/оба параметра по необходимости, не полагайтесь на дефолт сервиса (он настраивается на деплое и может отличаться между средами).
- **`playerUuid` в Punishment — это просто строка**, без резолвленного имени. Чтобы показать имя игрока, фронту нужно отдельно вызвать `/players/lookup?uuid=...` (имя там может быть `null`, обрабатывать как "Unknown player").
- **`expiresAt` отсутствует у permanent-записей** — проверяйте `permanent === true`, не `expiresAt == null`, как основной признак "бессрочно" (оба варианта эквивалентны, но `permanent` — явный).
- **`removed` отсутствует и при активном наказании, и при истёкшем автоматически** — чтобы понять причину закрытия записи, используйте пару `(removed, expired)`: `removed` есть → снято явно; `removed` нет и `expired === true` → истекло само; `removed` нет и `expired === false` → ещё действует (или активный флаг `active === false` без даты снятия, что обычно означает рассинхрон данных на стороне плагина — на это стоит не полагаться при отображении).
- **404 на `/players/lookup` — это нормальный сценарий**, а не ошибка интеграции: означает, что для переданного имени/uuid в истории имён нет записей.
