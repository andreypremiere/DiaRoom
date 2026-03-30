import asyncio
import time

from fastapi import FastAPI
from database import redis_client
from moderator import analyze_text
from tools import get_canvas_payload_by_post, process_results_async

app = FastAPI()


async def redis_worker():
    """Фоновый процесс для обработки очереди"""
    print("🚀 Worker started: watching 'posts_queue'...")

    cache_list = []

    BATCH_TIMEOUT = 0.5
    MAX_BATCH_SIZE = 16
    last_batch_time = time.time()
    while True:
        try:
            raw_data = await redis_client.brpop("new_posts:post_id", timeout=1)

            if raw_data:
                post_id = raw_data[1]
                payload = await get_canvas_payload_by_post(post_id)

                if payload:
                    full_text = " ".join([
                        block['text'].strip()
                        for block in payload
                        if block.get('type') == 'text'
                    ])
                    cache_list.append({
                        'post_id': post_id,
                        'full_text': full_text
                    })

            current_time = time.time()
            time_since_last = current_time - last_batch_time

            if len(cache_list) >= MAX_BATCH_SIZE or (time_since_last > BATCH_TIMEOUT and cache_list):
                print(f"📦 Обработка батча: {len(cache_list)} постов...")

                start_time = time.perf_counter()
                results = analyze_text(cache_list)
                end_time = time.perf_counter()
                duration = end_time - start_time
                print(f"⏱️ Функция analyze_text выполнена за {duration:.4f} сек.", flush=True)

                await process_results_async(results)

                cache_list = []
                last_batch_time = time.time()

        except Exception as e:
            print(f"❌ Worker Error: {e}")
            await asyncio.sleep(5)


@app.on_event("startup")
async def startup_event():
    asyncio.create_task(redis_worker())


@app.get("/")
def read_root():
    return {"status": "worker_running"}
