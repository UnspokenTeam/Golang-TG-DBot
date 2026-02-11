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
│   └── pg_dump_worker   # воркер для бэкапов (заготовка)
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
