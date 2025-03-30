## Установка

1. Клонируйте репозиторий:
```bash
git clone https://github.com/sdvaanyaa/mattermost-voting-bot.git
```

2. Настройка .env
   - Запустить mattermost + tarantool
   - Зарегистрируйтесь в Mattermost
   - Разрешите создание ботов: System Console -> Bot Account (Integration)
   - Создайте бота: Integrations -> Bot Accounts
   - Получите BOT_TOKEN
   - Вставьте токен в config

4. Запустить бота

## Запуск: 

1. Запустить mattermost + tarantool
```bash
docker-compose up --build mattermost tarantool -d
```

2. Запустить бота
```bash
docker-compose up --build app -d
```


