## Nginx Pro

Этот документ описывает шаги, необходимые для развертывания бэкенда ShopAPI и настройки обратного прокси Nginx в соответствии с заданием.

Вся конфигурация вынесена в папку deploy:

- nginx.conf — итоговый конфигурационный файл Nginx.
- init_db.sql — SQL-скрипт для настройки прав пользователя.
- static/ — файлы статики (index.html и image.png).
- hosts_example.txt — пример записи для локального DNS.

### 1. Настройка базы данных (PostgreSQL)
Для того чтобы два инстанса бэкенда работали в режиме read-only (только чтение), необходимо создать отдельного пользователя в базе данных.

Открываем pgAdmin или psql.

Выполняем скрипт из файла deploy/init_db.sql:

### 2. Запуск трех инстансов бэкенда
Бэкенд настроен на чтение порта из переменной окружения PORT. Для запуска трех инстансов открываем три терминала и выполняем команды:

**Терминал 1 (Основной, полные права):**\
powershell
cd src/ShopAPI
$env:PORT="8080"
$env:DB_USER="postgres"
$env:DB_PASSWORD="ваш_пароль_от_postgres"
go run cmd/main/main.go

**Терминал 2 (Read-Only):**\
powershell
cd src/ShopAPI
$env:PORT="8081"
$env:DB_USER="readonly_user"
$env:DB_PASSWORD="readonly_pass"
go run cmd/main/main.go

**Терминал 3 (Read-Only):**\
powershell
cd src/ShopAPI
$env:PORT="8082"
$env:DB_USER="readonly_user"
$env:DB_PASSWORD="readonly_pass"
go run cmd/main/main.go

### 3. Настройка локального DNS (Hosts)
Для того чтобы домен shop.local работал и браузер мог открыть HTTPS-сертификат, добавляем запись в файл hosts нашей операционной системы:

Открываем C:\Windows\System32\drivers\etc\hosts от имени администратора.

Добавляем в конец файла строку (см. deploy/hosts_example.txt):\
127.0.0.1 shop.local

### 4. Генерация SSL-сертификата
Самоподписанный сертификат должен лежать в папке C:\nginx\ssl\. Для его генерации выполняем в терминале (требуется установленный OpenSSL):

bash
mkdir C:\nginx\ssl
cd C:\nginx\ssl
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout shop.key -out shop.crt -subj "/C=RU/ST=Moscow/L=Moscow/O=ShopAPI/CN=shop.local"

### 5. Настройка Nginx
Копируем файл nginx.conf из этой папки в директорию C:\nginx\conf\nginx.conf.

Копируем файлы из папки static/ в директорию C:\nginx\html\.

Проверяем конфигурацию и перезапускаем Nginx:

powershell
cd C:\nginx
.\nginx.exe -t
.\nginx.exe -s reload

### Что настроено в Nginx:

1. Балансировка: GET-запросы к /api/v1/ распределяются между тремя бэкендами в пропорции 2:1:1 (8080 — 2 части, 8081 и 8082 — по 1 части). Остальные методы (POST, PUT, DELETE) идут только на основной бэкенд (8080).

2. Маршрутизация: /api/v1/ проксируется на бэкенд, /api/v1 (без слеша) отдает Swagger UI.

3. Статика: По пути / раздаются файлы из html.

4. Статус: По пути /status доступна статистика Nginx (stub_status).

5. pgAdmin: По пути /admin настроен редирект на локальный pgAdmin. (Примечание: из-за особенностей аутентификации pgAdmin Desktop используется редирект с ключом PGADMIN_INIT_KEY).

6. Gzip: Настроено сжатие текстовых данных (HTML, CSS, JSON), при этом медиафайлы (png, jpeg) не сжимаются.

7. Кэширование: Для статики настроен заголовок expires 30d.

8. HTTPS: Настроен редирект с HTTP (80) на HTTPS (443) с использованием самоподписанного сертификата.

### Как проверить работу

Открываем https://shop.local — должна загрузиться статическая страница магазина.

Открываем https://shop.local/api/v1/ — должен открыться Swagger UI.

Открываем https://shop.local/status — должна отобразиться статистика Nginx.

Открываем https://shop.local/admin — произойдет переход в pgAdmin.

Выполняем запрос https://shop.local/api/v1/clients — должен вернуться JSON. При обновлении страницы запросы будут распределяться между тремя инстансами (можно увидеть в логах терминалов).