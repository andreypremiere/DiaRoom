from datetime import datetime, timezone

from sqlalchemy import text
from sqlalchemy.ext.asyncio import create_async_engine

from database import async_engine


async def get_canvas_payload_by_post(post_id: str):
    """
    Асинхронно получает payload холста, связанного с конкретным постом.
    """
    try:
        # В асинхронном SQLAlchemy используем async with и .connect() или .begin()
        async with async_engine.connect() as conn:
            query = text("""
                SELECT c.payload 
                FROM canvases c
                JOIN posts p ON p.canvas_id = c.id
                WHERE p.id = CAST(:post_id AS UUID)
            """)

            # Выполняем запрос асинхронно с await
            result = await conn.execute(query, {"post_id": post_id})

            # .mappings().one_or_none() также работает в асинхронном режиме
            row = result.mappings().one_or_none()

            if row:
                return row['payload']

            print(f"⚠️ Payload не найден для post_id: {post_id}")
            return None

    except Exception as e:
        print(f"❌ Ошибка БД при получении payload: {e}")
        return None


async def process_results_async(results):
    print('Началась обработка в бд', flush=True)
    if not results:
        return

    now = datetime.now(timezone.utc)

    # Готовим список параметров для каждого поста
    data_to_update = [
        {
            "status": 'warning' if res['is_flagged'] else 'passed',
            "now": now,
            "reason": res['verdict'] if res['is_flagged'] else None,
            "post_id": res['post_id']
        }
        for res in results
    ]

    try:
        async with async_engine.begin() as conn:
            query = text("""
                    UPDATE posts 
                    SET ai_check_status = :status,
                        ai_check_at = :now,
                        ai_check_reason = :reason,
                        updated_at = :now
                    WHERE id = :post_id AND is_deleted = FALSE
                """)

            await conn.execute(query, data_to_update)

        print(f"✅ База данных обновлена: {len(results)} постов.")
    except Exception as e:
        print(f"❌ Ошибка массового обновления БД: {e}")