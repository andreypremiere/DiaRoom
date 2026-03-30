import asyncio

from fastapi import FastAPI
from database import engine, redis_client
from moderator import analyze_long_text
from tools import get_canvas_payload_by_post

app = FastAPI()


async def redis_worker():
    """Фоновый процесс для обработки очереди"""
    print("🚀 Worker started: watching 'posts_queue'...")

    while True:
        try:
            result = redis_client.blpop("new_posts:post_id", timeout=10)

            if result:
                print(result, flush=True)

                payload = get_canvas_payload_by_post(result[1])

                if payload is None:
                    print('Результат обращение к бд вернул none', flush=True)
                    continue
                else:
                    print(payload, flush=True)

                text_blocks = [
                    block['text'].strip()
                    for block in payload
                    if block.get('type') == 'text' and 'text' in block
                ]

                # Объединяем их в одну строку через пробел
                full_text = " ".join(text_blocks)

                print(f"Результат: {full_text}", flush=True)

                if full_text.strip():
                    # Проверка моделью
                    verdict = analyze_long_text(full_text)

                    status = "approved"
                    if verdict['label'] != "non-toxic":
                        status = "rejected"
                        print(f"🚫 Post {result[1]} REJECTED. Reason: {verdict['label']} ({verdict['score']:.2f})",
                              flush=True)
                    else:
                        print(f"✅ Post {result[1]} APPROVED", flush=True)


        except Exception as e:
            print(f"❌ Worker Error: {e}")
            await asyncio.sleep(5)  # Если ошибка (например, БД упала), ждем 5 сек и пробуем снова


@app.on_event("startup")
async def startup_event():
    asyncio.create_task(redis_worker())


@app.get("/")
def read_root():
    return {"status": "worker_running"}
