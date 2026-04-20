import time

import redis

REDIS_HOST = 'localhost'
REDIS_PORT = 6380
QUEUE_KEY = "new_posts:post_id"


def fill_queue():
    r = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    print(f"Начинаем заполнение тестирование...")
    try:
        while True:
            for i in range(32):
                r.lpush(QUEUE_KEY, '6132b35b-537c-45e3-866e-6a440f1c2d23')
                r.lpush(QUEUE_KEY, '1cbae323-b25f-4fb0-89fb-bb81df445fce')
            time.sleep(1)
            print(f"\nГотово! В очереди сейчас элементов: {r.llen(QUEUE_KEY)}")
            # break

    except Exception as e:
        print(f"Ошибка подключения: {e}")


if __name__ == "__main__":
    fill_queue()
