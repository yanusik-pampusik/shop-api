# BE4: авторизация и Dependency Injection

Два приложения на Go: REST API магазина (`ShopAPI`) и отдельный gRPC-сервис
авторизации (`AuthService`). Реализация в `develop` продолжает магазин из `base`
и сервис авторизации из `cassidal`.

## Соответствие заданию

| Требование | Реализация |
| --- | --- |
| DI через интерфейсы и IoC | `go.uber.org/dig` в `AuthService/internal/app` и `ShopAPI/internal/app`; репозитории, сервисы и gRPC-клиент передаются через интерфейсы |
| Отдельное приложение авторизации | `AuthService/cmd/auth`, только gRPC поверх HTTP/2 |
| Отдельная PostgreSQL-база | `auth_db` для пользователей; `shopapi` для магазина; Compose запускает их в отдельных контейнерах с отдельными томами |
| Регистрация и вход | RPC `Register` и `Login` возвращают подписанный JWT |
| Хеширование с солью | bcrypt с автоматически генерируемой солью; открытый пароль в БД не сохраняется |
| Смена пароля | `ChangePassword` проверяет JWT и старый пароль; пользователь определяется по подписанному `sub` |
| Восстановление | `ResetPassword` генерирует случайный пароль, сохраняет bcrypt-хеш и печатает письмо в консоль AuthService |
| Публичные методы магазина | `POST /register`, `POST /auth`, `POST /reset` вызывают gRPC без предварительной проверки JWT |
| Защита API | `middleware.RequireAuth` подключён к группе `/api/v1` и `/change-password`; каждый запрос проверяется RPC `ValidateToken` |
| Заголовок авторизации | `Authorization: Bearer <JWT>`; отсутствие/ошибка токена или недоступность проверки дают `401` |

В Go нет атрибутов методов как в C#. Собственный `RequireAuth` является явной
пометкой защищённого маршрута/группы при регистрации. Вся логика проверки находится
в middleware, выполняется до обработчика и не дублируется в контроллерах.
Swagger доступен публично: это документация, а не API работы с данными.

JWT содержит `sub`, `email`, `iss`, `aud`, `iat`, `nbf`, `exp`, `jti`. Сервис проверяет
подпись HS256, алгоритм, срок, issuer, audience и идентификатор пользователя.
Магазину не передаётся ключ подписи; локальной проверки или кеширования JWT нет.
Смена пароля не отзывает уже выданные JWT: они действуют до `exp`.

## Быстрый запуск в контейнерах

Требуется Docker Compose v2. Все следующие команды выполняются из `src`.

```sh
cp .env.example .env
# Вписать результат команды в AUTH_JWT_SECRET в .env:
openssl rand -hex 32
make up
```

`make up` выполняет `docker compose up --build -d`. Ключ должен содержать минимум
32 байта и оставаться прежним после перезапуска. По умолчанию:

- REST API: `http://localhost:8080`.
- Swagger: `http://localhost:8080/swagger/index.html`.
- gRPC `auth:50051` и обе базы доступны только внутри сети Compose.
- Данные сохраняются в томах `shop-data` и `auth-data`.

Остановка с сохранением данных: `make down`. Логи сервиса и заглушка письма:

```sh
docker compose logs -f auth
```

Для Podman те же действия доступны через `podman-compose up --build -d`,
`podman-compose logs -f auth`, `podman-compose down`.
Compose предназначен для локальной учебной среды: тестовые пароли БД заданы в
файле, внешний REST-порт привязан к loopback, внутренний gRPC использует HTTP/2
без TLS. Конфигурация Nginx предыдущего проекта сохранена в `deploy`; в неё
добавлена проксификация новых методов авторизации. Nginx не входит в Compose.

## Пример использования

Регистрация:

```sh
curl -i http://localhost:8080/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"ivan@example.com","first_name":"Ivan","last_name":"Ivanov","phone":"+79990000000","password":"initial-password"}'
```

Ответ `201`: `{"token":"..."}`. Скопировать JWT в переменную:

```sh
TOKEN='полученный-JWT'
curl -i http://localhost:8080/api/v1/products -H "Authorization: Bearer $TOKEN"
# Без заголовка — 401:
curl -i http://localhost:8080/api/v1/products
```

В Swagger нажать **Authorize** и вставить `Bearer <JWT>` целиком. Для пустой базы
список товаров будет пустым. Магазин сохраняет модель данных предыдущего проекта:
для создания клиента/поставщика требуется существующий адрес, а товара — категория
и поставщик; адреса и категории можно добавить в PostgreSQL.

Вход, смена пароля, восстановление:

```sh
curl -i http://localhost:8080/auth \
  -H 'Content-Type: application/json' \
  -d '{"email":"ivan@example.com","password":"initial-password"}'

curl -i http://localhost:8080/change-password \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $TOKEN" \
  -d '{"old_password":"initial-password","new_password":"changed-password"}'

curl -i http://localhost:8080/reset \
  -H 'Content-Type: application/json' -d '{"email":"ivan@example.com"}'
```

Новый случайный пароль для `/reset` появляется только в консоли AuthService.
Восстановить исходный пароль из bcrypt-хеша нельзя, поэтому он заменяется новым.
Для неизвестного email `/reset` также возвращает `{"success":true}`.

HTTP-ошибки: `400` — некорректные данные, `401` — неверные учётные данные/токен,
`409` — повторный email или удаление поставщика, на которого ссылается товар. Публичные методы возвращают `503` при недоступном gRPC,
`504` при таймауте, `500` при внутренней ошибке; внутренние сообщения БД не
раскрываются. Защищённые методы при любой неудаче проверки отвечают `401`.

## Локальный запуск без контейнеров

Требуются Go 1.26+, PostgreSQL 14+ и две базы с кодировкой UTF-8.
Сгенерированный protobuf-код включён в репозиторий; для сборки `protoc` не нужен.

```sh
createdb -h localhost -U postgres --encoding=UTF8 --template=template0 auth_db
createdb -h localhost -U postgres --encoding=UTF8 --template=template0 shopapi
make build
```

Первый терминал:

```sh
export AUTH_DB_DSN='postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable'
export AUTH_JWT_SECRET='ваш-сохранённый-ключ-не-короче-32-байт'
export AUTH_GRPC_ADDR='127.0.0.1:50051'
./bin/auth
```

Второй терминал:

```sh
export DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=shopapi
export AUTH_GRPC_ADDR='127.0.0.1:50051'
export PORT=8080 DB_MIGRATE=true
./bin/shop
```

Таблицы создаются автоматически. Оба сервиса корректно завершаются по SIGINT/SIGTERM.
Приложения читают переменные окружения; `.env` автоматически читает только Compose.

| Переменная | Значение по умолчанию / назначение |
| --- | --- |
| `AUTH_DB_DSN` | Обязательная строка подключения AuthService |
| `AUTH_JWT_SECRET` | Обязательный секрет от 32 байт, только в AuthService |
| `AUTH_GRPC_ADDR` | `127.0.0.1:50051`; адрес прослушивания для AuthService, адрес подключения для ShopAPI |
| `AUTH_JWT_ISSUER` / `AUTH_JWT_AUDIENCE` | `authservice` / `shopapi` |
| `AUTH_TOKEN_TTL` | `1h`, минимум `1s` |
| `DB_HOST` / `DB_PORT` | `localhost` / `5432` |
| `DB_USER` / `DB_PASSWORD` / `DB_NAME` | `postgres` / `postgres` / `shopapi` |
| `DB_SSLMODE` | `disable` |
| `PORT` | `8080` |
| `DB_MIGRATE` | `true` на порту 8080; `false` на остальных для совместимости с read-only инстансами предыдущего проекта. Можно явно переопределить |

Таймаут одного обращения магазина к gRPC — 3 секунды.

## Проверки

```sh
make test       # тесты обоих модулей с race detector
make vet        # статическая проверка обоих модулей
make integration
```

`make integration` собирает два приложения и запускает `scripts/integration.py`.
Нужны Python 3 и утилиты PostgreSQL (`initdb`, `pg_ctl`, `createdb`, `psql`) в PATH.
На macOS с Homebrew при необходимости:

```sh
PG_BINDIR="$(brew --prefix postgresql@17)/bin" make integration
```

Сценарий сам создаёт временный кластер с двумя UTF-8 базами, выбирает свободные
локальные порты, запускает приложения и удаляет свои процессы и данные после
завершения. Существующие базы не используются. Проверяет:

- регистрацию, дубликаты, валидацию, вход и bcrypt с разными солями;
- разделение таблиц между базами;
- защиту всех 22 закрытых маршрутов без JWT и с неверным JWT, а также границы JWT claims и алгоритмов подписи;
- полный CRUD клиентов, поставщиков, товаров и изображений, включая сохранение новых адресов и содержимого файлов;
- нулевую цену и отклонение отсутствующей, null и отрицательной цены;
- одновременную регистрацию, списание остатков и смену пароля;
- конфликт удаления поставщика, связанного с товаром;
- смену/сброс пароля и работоспособность пароля из заглушки письма;
- сохранение пользователя после перезапуска AuthService;
- отказ в доступе при остановке AuthService и восстановление связи после запуска.

Дополнительно тесты ShopAPI поднимают реальный gRPC-сервер в памяти, собирают IoC
и проверяют все 22 защищённых маршрута, отсутствие кеширования проверки JWT,
таймауты и преобразование ошибок. Во время `make integration` тесты обоих Go-модулей
также запускаются с `-race` на временных базах. Они проверяют атомарное списание
остатков, привязку адреса и откат транзакции при ошибке, а также запрет устаревшей
смены пароля после сброса. Для JSON-отчёта можно задать `E2E_REPORT=/tmp/be4-e2e.json`.

Тесты с PostgreSQL можно запускать отдельно, указав отдельную тестовую базу
через `AUTH_TEST_DB_DSN` для AuthService или `SHOP_TEST_DB_DSN` для ShopAPI. Например:

```sh
cd AuthService
AUTH_TEST_DB_DSN='postgres://postgres:postgres@localhost:5432/auth_test?sslmode=disable' go test -race ./...
```

## Обновление контрактов

Единый исходный контракт — `proto/auth/auth.proto`. Из него генерируется код обоих
приложений. После изменения контракта нужны `protoc`, `protoc-gen-go` и
`protoc-gen-go-grpc`:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
export PATH="$(go env GOPATH)/bin:$PATH"
make proto
make swagger
```

`make swagger` обновляет JSON/YAML и встроенную документацию по комментариям
контроллеров. Секреты, локальные `.env` и собранные бинарники исключены из Git.

## Выполненная проверка

21 октября 2026 года после исправления замечаний ревью выполнены: тесты обоих
модулей с `-race` на PostgreSQL 17, `go vet` и полный сценарий
`scripts/integration.py`: 112 проверок, 0 ошибок. Сценарий покрывает все 25 операций
REST API, включая конкурентные запросы и восстановление после остановки AuthService.
Сборка и запуск Compose дополнительно проверены через Podman. В контейнерах
проверены регистрация, вход, защищённый API, смена/сброс пароля и Swagger.
Тестовые процессы, контейнеры и тома остановлены и удалены.
