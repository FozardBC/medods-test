## Для запуска:

docker-compose up -d --remove-orphans

#### Настройка переменных окружения:
./build/docker-compose.yml

    - LOG_MODE=debug
      - JWT_SECRET=asdgasgfdgabu3gpf19r3bg08vduhdwpuh;alksdnfads
        # LISTEN
      - SRV_HOST=0.0.0.0
      - SRV_PORT=8080
        # DB
      - DB_CONN_STRING=postgresql://postgres:qwerty@postgres:5432/medods-test-task-db?sslmode=disable
      - WEB_HOOK=https://webhook.site/22f86d6b-7aca-42fd-a208-ac8d316d88b1  # сюда приходят вебхуки
