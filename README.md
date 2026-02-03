# GoLang TG D-Bot — Production Compose

Этот проект поднимается через Docker Compose с использованием `.env` файла и профилей (`prod`).

## Требования

- Docker с включенным Docker Compose v^2.

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