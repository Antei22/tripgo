# TripGo — репозиторий для лабораторных работ

Заготовка курса «Разработка микросервисов на Go». Здесь вы делаете все пять
работ: каждая следующая продолжает предыдущую, переписывать сервис с нуля не
нужно.

## Что делать сразу

1. **Fork** этого репозитория к себе. Форк нужен, чтобы преподаватели видели
   список всех работ курса одной страницей.
2. Заведите модуль:

```bash
git clone git@github.com:<ваш-логин>/<ваш-репозиторий>.git
cd <ваш-репозиторий>
go mod init github.com/<ваш-логин>/<ваш-репозиторий>
```

Путь модуля потом не меняется — иначе придётся править все импорты. Проще всего
взять адрес своего репозитория, каким бы он ни был.

Дальше — [`homework/docs/getting-started.md`](https://github.com/course-go-autumn-2026/course/blob/main/homework/docs/getting-started.md)
в репозитории курса: инструменты, окружение, миграции, вид сданной работы.

## Где что лежит

| Что | Где |
|---|---|
| Задания, документация, контракты | [`course-go-autumn-2026/course`](https://github.com/course-go-autumn-2026/course) |
| Слайды и записи лекций | [`lections/`](https://github.com/course-go-autumn-2026/course/tree/main/lections) |
| Как оценивают, дедлайны, порядок сдачи | [`homework/docs/grading.md`](https://github.com/course-go-autumn-2026/course/blob/main/homework/docs/grading.md) |
| Локальное окружение и утилита `tripgoctl` | [`course-go-autumn-2026/course-infra`](https://github.com/course-go-autumn-2026/course-infra) |

Задания появляются по мере курса, каждое — после своей пары лекций.

## Как сдавать

Ветка `homework/NN` от `main`, pull request в `main` своего форка, ссылка
ментору до дедлайна. Подробно — в `grading.md` репозитория курса.

## Чужие работы

Форки видны всем, включая ваши. Смотреть чужие решения, пока идёт курс, —
плохая идея: одинаковый код виден сразу, а разбираться на защите придётся
самому.

# TripGo Trip Service

Сервис управления поездками для лабораторных работ курса «Разработка микросервисов на Go».

В рамках лабораторной работы №1 реализованы:

- создание поездки;
- получение поездки по идентификатору;
- завершение поездки;
- хранение данных в PostgreSQL;
- журнал изменений статуса;
- HTTP API, сгенерированный из OpenAPI;
- менеджер транзакций;
- защита от конкурентных запросов;
- liveness- и readiness-проверки;
- graceful shutdown.

Задания со звёздочкой — идемпотентность и Dockerfile — не реализованы.

## Требования

Для запуска необходимы:

- Linux или WSL2;
- Docker;
- `tripgoctl`;
- Go с поддержкой toolchain из `go.mod`;
- `make`.

Нативный Windows не поддерживается утилитой `tripgoctl`, поэтому на Windows команды необходимо выполнять внутри WSL2.

## Быстрый запуск

Перейти в каталог проекта:

```bash
cd "/mnt/c/Users/Artem/Yandex.Disk-arigsolovev@edu.hse.ru/GO/labs/tripgo"
```

Проверить окружение:

```bash
tripgoctl doctor
```

Запустить PostgreSQL:

```bash
tripgoctl environment start
tripgoctl connect
```

Применить миграции:

```bash
make migrate
```

Запустить сервис:

```bash
make run
```

По умолчанию HTTP-сервер доступен по адресу:

```text
http://localhost:8080
```

Проверка:

```bash
curl -i localhost:8080/health
curl -i localhost:8080/ready
```

## Команды Makefile

| Команда | Назначение |
|---|---|
| `make generate` | сгенерировать Go-код из OpenAPI |
| `make migrate` | применить все неприменённые миграции |
| `make migrate-down` | откатить одну последнюю миграцию |
| `make migrate-status` | показать состояние миграций |
| `make run` | запустить сервис |
| `make build` | собрать все пакеты |
| `make test` | запустить тесты с детектором гонок |

Полный откат двух миграций:

```bash
make migrate-down
make migrate-down
```

После отката схему можно создать заново:

```bash
make migrate
```

## Конфигурация

Конфигурация читается только из переменных окружения.

`tripgoctl environment start` создаёт локальный файл `.env` с адресом PostgreSQL. Файл `.env` не коммитится.

Безопасные примеры значений хранятся в `.env.example`.

| Переменная | Назначение | Пример |
|---|---|---|
| `HTTP_ADDR` | адрес HTTP-сервера | `:8080` |
| `LOG_LEVEL` | уровень логирования | `info` |
| `SHUTDOWN_TIMEOUT` | бюджет graceful shutdown | `10s` |
| `HTTP_READ_TIMEOUT` | таймаут чтения запроса | `5s` |
| `HTTP_READ_HEADER_TIMEOUT` | таймаут чтения заголовков | `2s` |
| `HTTP_WRITE_TIMEOUT` | таймаут записи ответа | `10s` |
| `HTTP_IDLE_TIMEOUT` | таймаут keep-alive соединения | `60s` |
| `DATABASE_URL` | строка подключения к PostgreSQL | создаётся `tripgoctl` |
| `DATABASE_MAX_CONNS` | максимальный размер пула | `10` |
| `DATABASE_MIN_CONNS` | минимальный размер пула | `2` |
| `DATABASE_MAX_CONN_LIFETIME` | максимальное время жизни соединения | `30m` |
| `DATABASE_CONNECT_TIMEOUT` | таймаут подключения к PostgreSQL | `5s` |
| `DATABASE_QUERY_TIMEOUT` | таймаут SQL-запроса | `3s` |

Если обязательная переменная отсутствует или содержит некорректное значение, сервис завершается при запуске с понятной ошибкой.

## HTTP API

Источник истины для API:

```text
contracts/openapi/trip-service.openapi.yaml
```

Go-типы и серверный интерфейс генерируются командой:

```bash
make generate
```

Сгенерированный файл:

```text
internal/generated/api.gen.go
```

Сгенерированный код не редактируется вручную.

### Создать поездку

```http
POST /api/v1/trips
Content-Type: application/json
```

Пример:

```bash
curl -i -X POST localhost:8080/api/v1/trips \
  -H 'content-type: application/json' \
  -d '{
    "user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
    "driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
    "start_point":{
      "latitude":59.9398,
      "longitude":30.3146
    },
    "end_point":{
      "latitude":59.9290,
      "longitude":30.3626
    },
    "price":1450
  }'
```

Успешный ответ:

```text
201 Created
```

Ответ содержит созданную поездку и заголовок:

```text
Location: /api/v1/trips/{tripId}
```

### Получить поездку

```bash
curl -i \
  localhost:8080/api/v1/trips/{tripId}
```

Успешный ответ:

```text
200 OK
```

### Завершить поездку

```bash
curl -i -X POST \
  localhost:8080/api/v1/trips/{tripId}/finish
```

Первое завершение возвращает:

```text
200 OK
```

Повторное завершение возвращает:

```text
409 Conflict
code = trip_completed
```

## Коды ошибок

Ошибки API возвращаются с заголовком:

```http
Content-Type: application/problem+json
```

Формат соответствует RFC 9457.

| HTTP-код | `code` | Описание |
|---:|---|---|
| `400` | `invalid_request` | некорректный JSON, UUID или значения полей |
| `404` | `trip_not_found` | поездка не найдена |
| `409` | `trip_completed` | поездка уже завершена |
| `409` | `driver_busy` | у водителя уже есть активная поездка |
| `500` | `internal_error` | внутренняя ошибка сервиса |

Внутренние ошибки PostgreSQL, SQL-запросы и конфиденциальные данные клиенту не возвращаются.

## Health и Ready

### Liveness

```bash
curl -i localhost:8080/health
```

`/health` показывает, что процесс работает, и не обращается к PostgreSQL.

Ответ:

```json
{
  "status": "ok"
}
```

### Readiness

```bash
curl -i localhost:8080/ready
```

`/ready` выполняет `Ping` PostgreSQL.

При доступной базе возвращается:

```text
200 OK
```

При недоступной базе:

```text
503 Service Unavailable
```

```json
{
  "status": "unavailable"
}
```

## Схема PostgreSQL

Миграции находятся в каталоге:

```text
migrations/
```

Создаются две таблицы:

- `trips` — поездки;
- `trip_status_history` — история изменений статуса.

При создании поездки сервис атомарно выполняет:

1. `INSERT` в `trips`;
2. `INSERT` начального статуса в `trip_status_history`;
3. `COMMIT`.

Если вторая вставка завершается ошибкой, первая вставка откатывается.

При завершении поездки изменение `trips` и запись в `trip_status_history` также выполняются в одной транзакции.

## Менеджер транзакций

Менеджер транзакций предоставляет интерфейс:

```go
type TxManager interface {
    Do(
        ctx context.Context,
        fn func(ctx context.Context) error,
    ) error
}
```

`Do` выполняет следующие действия:

1. открывает транзакцию;
2. кладёт `pgx.Tx` в `context.Context`;
3. вызывает функцию с новым контекстом;
4. выполняет `COMMIT`, если функция вернула `nil`;
5. выполняет `ROLLBACK`, если функция вернула ошибку;
6. выполняет `ROLLBACK` и повторно вызывает panic, если функция паниковала.

Repository самостоятельно выбирает исполнителя:

- если в контексте есть транзакция — SQL выполняется через неё;
- если транзакции нет — SQL выполняется через `pgxpool.Pool`.

Транзакция не передаётся аргументом в методы repository.

Вложенный вызов `Do` переиспользует существующую транзакцию и не открывает вторую.

## Уровень изоляции

Используется уровень изоляции PostgreSQL:

```text
Read Committed
```

Он выбран потому, что безопасность конкурентных операций обеспечивается не последовательным выполнением всех транзакций, а конкретными ограничениями и атомарными запросами:

- запрет двух активных поездок обеспечивается уникальным индексом;
- конкурентное завершение защищено условием непосредственно в `UPDATE`.

Это позволяет использовать стандартный уровень изоляции PostgreSQL без лишних serialization failures.

## Защита от двух активных поездок

В PostgreSQL создан частичный уникальный индекс:

```sql
CREATE UNIQUE INDEX trips_one_active_per_driver_idx
    ON trips (driver_id)
    WHERE status = 'active';
```

Благодаря индексу два параллельных запроса на создание поездки одному водителю не могут создать две активные поездки.

Один запрос получает `201`, остальные получают:

```text
409 driver_busy
```

Ошибка PostgreSQL с кодом `23505` и именем этого индекса преобразуется в доменную ошибку `ErrDriverBusy`.

## Защита конкурентного завершения

Завершение выполняется условным запросом:

```sql
UPDATE trips
SET
    status = 'completed',
    finished_at = $1,
    updated_at = $1
WHERE id = $2
  AND status = 'active'
RETURNING ...;
```

Если два запроса одновременно завершают одну поездку, только один из них обновляет строку.

Результат:

- один запрос получает `200`;
- второй получает `409 trip_completed`;
- поле `finished_at` не перезаписывается.

## Проверка конкурентного создания

Создать локальный файл `trip.json` с новым `driver_id` и выполнить:

```bash
seq 20 | xargs -P20 -I{} curl -s \
  -o /dev/null \
  -w '%{http_code}\n' \
  -X POST localhost:8080/api/v1/trips \
  -H 'content-type: application/json' \
  -d @trip.json |
  sort |
  uniq -c
```

Ожидаемый результат:

```text
1 201
19 409
```

`trip.json` используется только для локальной проверки и не коммитится.

## Graceful shutdown

Сервис обрабатывает:

- `SIGINT`;
- `SIGTERM`.

После получения сигнала сервис:

1. перестаёт принимать новые HTTP-запросы;
2. ожидает завершения текущих запросов;
3. завершает HTTP-сервер;
4. закрывает пул PostgreSQL;
5. выходит.

Общий бюджет остановки задаётся переменной:

```text
SHUTDOWN_TIMEOUT
```

Локальная проверка выполняется нажатием `Ctrl+C`.

При корректной остановке в логах появляются сообщения:

```text
shutdown signal received
service stopped gracefully
```

При запуске через `go run` команда `make` может напечатать `Error 1` после `Ctrl+C`, потому что сигнал также получает процесс `go run`. Корректность остановки приложения определяется по логам graceful shutdown.

## Структура проекта

```text
cmd/trip-service/          точка входа
internal/config/           чтение и проверка конфигурации
internal/database/         подключение к БД и TxManager
internal/generated/        сгенерированный OpenAPI-код
internal/httpserver/       HTTP-обработчики и роутер
internal/trips/            модель, repository и бизнес-операции
migrations/                goose-миграции
contracts/                 внешние контракты курса
oapi-codegen.yaml          конфигурация генерации OpenAPI
environment.toml           состав локального окружения
Makefile                   команды проекта
.env.example               пример переменных окружения
```

## Финальная проверка

Перед отправкой работы:

```bash
gofmt -w cmd internal
go mod tidy
go vet ./...
make generate
make test
make build
```

Проверить состояние Git:

```bash
git status
```

Файлы `.env`, `.tripgo/`, `bin/` и `trip.json` не должны попадать в коммит.