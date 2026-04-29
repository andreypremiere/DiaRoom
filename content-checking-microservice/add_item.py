import time

import redis

REDIS_HOST = 'localhost'
REDIS_PORT = 6380
QUEUE_KEY = "new_posts:post_id"


def fill_queue():
    r = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    print(f"Начинаем заполнение тестирование...")
    try:
        # while True:
        #     for i in range(32):
        r.lpush(QUEUE_KEY, '12a1cf4a-1dc4-4be5-904a-65ef35e9b165')
        r.lpush(QUEUE_KEY, 'ddcd8e07-1deb-4439-b9a4-0ea01631a8aa')
            # time.sleep(1)
        print(f"\nГотово! В очереди сейчас элементов: {r.llen(QUEUE_KEY)}")
            # break

    except Exception as e:
        print(f"Ошибка подключения: {e}")


if __name__ == "__main__":
    fill_queue()