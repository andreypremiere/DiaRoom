import redis
import uuid
import time

# Если запускаешь С КОМПЬЮТЕРА, используй localhost и порт 6380 (из твоего compose)
# Если запускаешь ВНУТРИ DOCKER, используй redis-queue и порт 6379
REDIS_HOST = 'localhost'
REDIS_PORT = 6380
QUEUE_KEY = "new_posts:post_id"


def fill_queue(count=10000):
    try:
        # Подключаемся к Redis
        r = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)

        print(f"🚀 Начинаем заполнение очереди '{QUEUE_KEY}'...")

        r.lpush(QUEUE_KEY, '1cbae323-b25f-4fb0-89fb-bb81df445fce')

        # for i in range(count):
        #     # Генерируем случайный UUID, как это делает Go
        #     post_id = str(uuid.uuid4())
        #
        #     # Используем LPUSH (как в твоем Go-коде)
        #     r.lpush(QUEUE_KEY, post_id)
        #
        #     print(f"✅ [{i + 1}] Добавлен PostID: {post_id}")
        #     time.sleep(0.001)  # Небольшая пауза для наглядности

        print(f"\n✨ Готово! В очереди сейчас элементов: {r.llen(QUEUE_KEY)}")

    except Exception as e:
        print(f"❌ Ошибка подключения: {e}")


if __name__ == "__main__":
    fill_queue()