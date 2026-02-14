[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/UnspokenTeam/Golang-TG-DBot)
# GoLang TG D-Bot — Production Compose

Проект поднимается через Docker Compose с использованием `.env` файла и профилей (`prod`).

## Технологии

- Go 1.25.6
- Telegram Bot API через библиотеку telego.
- PostgreSQL
- sqlc (генерация типобезопасного кода для SQL).
- Redis (rate limiting и кэш/состояния).
- ClickHouse (хранилище данных observability).
- OpenTelemetry (инструментирование, трассировка, метрики, логи).
- Uptrace (UI и сбор телеметрии).
- Viper (управление конфигурацией).
- Caddy (reverse proxy и TLS).
- slog + otelpgx (структурированные логи и трейсинг запросов в Postgres).
- Docker + Docker Compose (локальный и прод запуск).

## Возможности

- Высокая надежность: graceful shutdown, health checks и устойчивое завершение процессов.
- Масштабирование: Redis rate limiting, очереди сообщений и контроль нагрузки.
- Observability: OpenTelemetry-инструментирование, распределенные трейсы в Uptrace, метрики и логи в ClickHouse.
- Безопасность: TLS через Caddy, role-based access control, webhook secret tokens.
- Командная система бота с ролями пользователей (owner/admin/user) и приватной статистикой.
- Воркеры и очередь отправки для рассылок/анонсов с ограничением скорости.
- Middleware-пайплайн для фильтрации пользователей и анти-спам логики.

## Требования

- Docker с включенным Docker Compose v^2.

## Структура проекта

```
.
├── cmd
│   ├── bot              # входная точка бота
│   └── pg_dump_worker   # воркер для бэкапов
├── internal
│   ├── bot              # приложение бота, обработчики, воркеры, graceful shutdown
│   ├── configs          # загрузка и кеширование конфигурации
│   ├── db               # доступ к Postgres и инструментирование запросов
│   ├── logger           # логирование и связка с OpenTelemetry/Uptrace
│   └── middlewares      # фильтры/ограничители для входящих апдейтов
├── pkg
│   └── utils            # общие утилиты
├── configs              # конфиги сервисов окружения (postgres/observability и др.)
├── deployments          # compose-файлы для local/prod
├── Dockerfile
├── docker-compose.yml
└── example.env
```

## Запуск
dev:
```bash
cd deployments
docker compose -f docker-compose.local.yml --env-file ./../example.env up -d
```

prod:
```bash
cd deployments
docker compose -f docker-compose.prod.yml --env-file ./../.env --profile prod up -d
```

## Production deploy
Для запуска требуется собрать образ проекта в какой-то registry через CI/CD

### Server requirements:
- Docker Compose v2 and above
- Go 1.25.6 and above (job build)
- Cron (запуск job)

### Backups 

В проекте присутствует джоба предназначенная для бэкапов, делается pgDump и грузится на внешний S3 через SFTP
Требуется настроить окружение в котором будет выполняться джоба:
- POSTGRES_INTERNAL_HOST - имя контейнера Postgres
- POSTGRES_PGUSER - юзер от лица, которого будет dump
- POSTGRES_PGDB - база, которую экспортирует джоба
- SFTP_HOST - хост, на котором располагается ваше S3-хранилище (Ceph/MinIO)
- SFTP_PORT - SSH PORT 22
- SFTP_USER - SSH User
- SFTP_PASSWORD - SSH Password
- SFTP_PATH - название бакета

<blockquote>
Джобу требуется забилдить и получить независимый go-бинарник:

```shell
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -a -installsuffix cgo -o go_pg_dump ./cmd/pg_dump_worker/main.go
chmod +x go_pg_dump
```

Джоба запускается с cron через shell с подгрузкой окружения из .env файла на сервере:
```shell
chmod +x run-backup.sh
```

Добавляем cron job через crontab (требуется задать адрес)
```shell
( crontab -l 2>/dev/null; echo '0 23 * * * /PATH-TO-SHELL-SCRIPT/run-backup.sh >> /var/www/golang-tg-dbot/backup-cron.log 2>&1' ) | crontab -
```
</blockquote>

Рестор c бэкапа:
```shell
docker exec -i $POSTGRES_INTERNAL_HOST pg_restore \
  -U postgres \
  --verbose \
  --clean \
  --no-acl \
  --no-owner \
  -d bot_db \         
  < dump.dump 
```