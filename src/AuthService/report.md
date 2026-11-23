# AuthService: gRPC, JWT и запуск

Это часть 2 проекта BE4. Сервис построен на пользовательской логике ветки
`isabelsc`; RPC-контракт совместим с клиентом из `mozik`. Сервис принимает
только gRPC поверх HTTP/2, по умолчанию на `127.0.0.1:50051`.

## Что реализовано

- Отдельное Go-приложение с PostgreSQL и миграцией модели `User` при запуске.
- `Register`, `Login`, `ResetPassword`, `ValidateToken`, `ChangePassword`.
- Подписанные HS256 JWT с `sub`, `email`, `iss`, `aud`, `iat`, `nbf`, `exp`, `jti`.
- Проверка подписи, разрешённого алгоритма, срока действия, issuer, audience
  и идентификатора пользователя. Некорректный JWT не даёт доступ.
- IoC-контейнер `dig`: репозиторий → пользовательский сервис → gRPC-обработчик;
  JWT-менеджер и пользовательская логика внедряются через интерфейсы.
- Проверка входных данных и перевод ошибок в gRPC status codes.
- Обработка SIGINT/SIGTERM: завершение текущих RPC, ограниченное пятью секундами,
  и закрытие соединения с БД.

Для подключения части 1 реализована её заглушка `ChangePassword`, а пароль
восстановления теперь генерируется через `crypto/rand` вместо общей константы.
Смена требует действующий JWT и старый пароль. Пользователь определяется из
подписанного токена. При восстановлении новый пароль сохраняется как bcrypt-хеш
и выводится в консоль сервиса — это требуемая заданием заглушка отправки письма.

## Запуск

Требуются Go 1.26+ и PostgreSQL. Команды ниже выполняются из `src/AuthService`.
Сгенерированные protobuf-файлы включены в репозиторий: для обычной сборки
`protoc` и Git LFS не нужны.

1. Создать **отдельную** базу авторизации (магазин использует `shopapi`):

   ```sh
   psql -h localhost -U postgres -d postgres -v ON_ERROR_STOP=1 -f init_auth_db.sql
   ```

   Скрипт выполняется один раз. Таблица `users` мигрируется самим сервисом
   через GORM `AutoMigrate` в базе из `AUTH_DB_DSN`.

2. Задать конфигурацию в окружении:

   ```sh
   export AUTH_DB_DSN='host=localhost user=postgres password=postgres dbname=shopapi_auth port=5432 sslmode=disable'
   export AUTH_GRPC_ADDR='127.0.0.1:50051'
   export AUTH_JWT_SECRET="$(openssl rand -hex 32)"
   export AUTH_JWT_ISSUER='authservice'
   export AUTH_JWT_AUDIENCE='shopapi'
   export AUTH_TOKEN_TTL='1h'
   ```

   Для своей БД подставьте её пользователя и пароль. DSN и JWT-секрет обязательны.
   Секрет должен содержать минимум 32 байта; сохраните его между перезапусками,
   чтобы ранее выпущенные токены оставались проверяемыми. Остальные параметры
   имеют значения по умолчанию, показанные выше. TTL — длительность Go, минимум
   `1s`. Файл `.env.example` служит образцом; приложение не читает `.env` автоматически.

3. Собрать и запустить:

   ```sh
   go mod download
   go build -o authservice ./cmd/auth
   ./authservice
   ```

   Или: `go run ./cmd/auth`. Остановка — Ctrl+C.

Сервис использует незашифрованный HTTP/2 на loopback, совместимый с
`insecure.NewCredentials()` в клиенте `mozik`. При размещении сервисов на разных
машинах используйте закрытую доверенную сеть или настройте TLS на транспортном
контуре. REST-ручек у AuthService нет: `/register`, `/auth`, `/reset` реализуются
в магазине, который обращается к этому сервису.

## RPC-контракт и клиент магазина

Единый источник контракта: `../proto/auth/auth.proto`.
В нём сохранены package `auth`, имена исходных четырёх методов и номера полей
из ветки `mozik`. Добавлен только метод `ChangePassword` и его сообщения.

| Метод | Вход | Результат |
|---|---|---|
| `Register` | email, first_name, last_name, phone, password | Подписанный JWT |
| `Login` | email, password | Подписанный JWT |
| `ValidateToken` | token без префикса Bearer | is_valid, user_id, email |
| `ChangePassword` | token, old_password, new_password | success |
| `ResetPassword` | email | success; новый пароль в консоли |

Невалидный токен в `ValidateToken` возвращает `is_valid=false`, без идентичности
пользователя. Магазин переводит это в HTTP 401. Неизвестный email при сбросе
возвращает тот же `success=true`, чтобы публичный метод не раскрывал список
пользователей. Выпущенные JWT действуют до `exp`; отзыв токенов при смене/сбросе
пароля в этой реализации не предусмотрен.

Ошибки: `InvalidArgument` для некорректных полей, `AlreadyExists` для повторной
регистрации, `Unauthenticated` для неверного пароля или токена смены пароля,
`Internal` для внутренних сбоев без передачи деталей БД клиенту. Пароль должен
содержать 1–72 байта: это ограничение bcrypt, а не число символов.

Магазину нужно задать `AUTH_SERVICE_ADDR=127.0.0.1:50051`. Интеграция middleware
с существующими маршрутами `ShopAPI` остаётся частью 3. При объединении с `mozik`
сохраните расширенную версию `auth.proto` из этой ветки.

Пример ручной проверки установленным `grpcurl` (из `src/AuthService`):

```sh
grpcurl -plaintext -import-path ../proto/auth -proto auth.proto \
  -d '{"email":"demo@example.com","first_name":"Ivan","last_name":"Ivanov","phone":"+79991234567","password":"secret123"}' \
  localhost:50051 auth.AuthService/Register

grpcurl -plaintext -import-path ../proto/auth -proto auth.proto \
  -d '{"email":"demo@example.com","password":"secret123"}' \
  localhost:50051 auth.AuthService/Login

grpcurl -plaintext -import-path ../proto/auth -proto auth.proto \
  -d '{"token":"ВСТАВЬТЕ_ТОКЕН"}' \
  localhost:50051 auth.AuthService/ValidateToken

grpcurl -plaintext -import-path ../proto/auth -proto auth.proto \
  -d '{"token":"ВСТАВЬТЕ_ТОКЕН","old_password":"secret123","new_password":"another-password"}' \
  localhost:50051 auth.AuthService/ChangePassword

grpcurl -plaintext -import-path ../proto/auth -proto auth.proto \
  -d '{"email":"demo@example.com"}' \
  localhost:50051 auth.AuthService/ResetPassword
```

## Генерация protobuf

В репозитории сохранён результат генерации. При изменении контракта установите
`protoc` 33.0 и плагины:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
export PATH="$(go env GOPATH)/bin:$PATH"
sh scripts/generate.sh
```

Скрипт переопределяет `go_package` только для сервера на `authservice/proto/auth`.
Для магазина исходный `go_package` остаётся `shopapi/proto/auth`; после переноса
его кода в модуль `src/ShopAPI` генерация выполняется из этого каталога:

```sh
protoc -I ../proto/auth \
  --go_out=. --go_opt=module=shopapi \
  --go-grpc_out=. --go-grpc_opt=module=shopapi \
  auth.proto
```

## Проверки

Из `src/AuthService`:

```sh
go test -race ./...
go vet ./...
go build ./...
```

Обычные тесты не требуют БД: gRPC использует реальный HTTP/2-транспорт через
`bufconn`, пользовательскую логику и JWT с репозиторием в памяти. Проверяются
регистрация/вход, дубли email, смена/сброс пароля, ошибки RPC, подпись и алгоритм
JWT, issuer/audience, exp/nbf/iat, отсутствующие обязательные claims, подделка
токенов и конфигурация.

Для полного теста запуска, миграции, сохранения данных и повторного запуска
создайте отдельную тестовую БД:

```sh
createdb -h localhost -U postgres authservice_test
AUTH_TEST_DB_DSN='host=localhost user=postgres password=postgres dbname=authservice_test port=5432 sslmode=disable' \
  go test -race ./...
```

`TestPostgresApplication` запускает приложение на локальном TCP-порту, выполняет
RPC, проверяет bcrypt-хеш в PostgreSQL, меняет пароль, перезапускает сервис и
проверяет сохранение учётных данных и валидность JWT. Затем проверяет сброс
пароля и завершение сервера. Тест удаляет своего пользователя; схема `users`
остаётся. Без `AUTH_TEST_DB_DSN` этот тест пропускается.
