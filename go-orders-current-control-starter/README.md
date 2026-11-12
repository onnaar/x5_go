# Сервис заказов

реализован сервис для работы с заказами. Данные хранятся в памяти приложения, поэтому после перезапуска сервера созданные заказы сбрасываются.

## Что сделано

- разделил приложение на `handler`, `service` и `repository`;
- добавил создание заказа, получение одного заказа и списка заказов;
- реализовал проверку позиций и расчёт итоговой стоимости;
- добавил изменение статусов и историю изменений;
- добавил единый формат JSON-ошибок;
- подключил структурированные логи, request ID и recovery middleware;
- добавил Swagger-документацию;
- написал table-driven unit-тесты для бизнес-логики.

Допустимые статусы: `new`, `processing`, `completed`, `cancelled`. Статус нового заказа — `new`. Разрешённые переходы: `new → processing`, `new → cancelled`, `processing → completed` и `processing → cancelled`.

```bash
go mod tidy
go run ./cmd/api
```

Проверка доступности:

```bash
curl http://localhost:8080/health
```

Swagger UI доступен по адресу [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html).

## API

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/health` | Проверка сервиса |
| `POST` | `/api/v1/orders` | Создание заказа |
| `GET` | `/api/v1/orders` | Список заказов |
| `GET` | `/api/v1/orders/{id}` | Получение заказа |
| `PATCH` | `/api/v1/orders/{id}/status` | Изменение статуса |
| `GET` | `/api/v1/orders/{id}/history` | История статусов |

Пример создания заказа:

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"name":"Keyboard","quantity":1,"price":500000}]}'
```

## Проверка

```bash
go test ./...
go vet ./...
go build ./...
```