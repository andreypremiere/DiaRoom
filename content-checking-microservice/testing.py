import asyncio
from datetime import datetime, timezone

from sqlalchemy.ext.asyncio import create_async_engine
from sqlalchemy import text


async_engine = create_async_engine("postgresql+asyncpg://postgres:1111@localhost:5435/db_posts")


async def process_results_async(results):
    if not results:
        return

    now = datetime.now(timezone.utc)

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

# 👉 Тестовые данные (подставь свои post_id)
test_results = [
    {
        "post_id": "039cf785-4dac-4ef1-8b32-e56959653c3e",
        "verdict": "insult",
        "confidence": 0.91,
        "all_scores": {
            "non-toxic": 0.1,
            "insult": 0.91,
            "obscenity": 0.02,
            "threat": 0.01,
            "dangerous": 0.0
        },
        "is_flagged": True
    },
    {
        "post_id": "09dc8df1-9b8d-49b8-91fa-f81d72f8fc2b",
        "verdict": "non-toxic",
        "confidence": 0.99,
        "all_scores": {
            "non-toxic": 0.99,
            "insult": 0.01,
            "obscenity": 0.0,
            "threat": 0.0,
            "dangerous": 0.0
        },
        "is_flagged": False
    }
]


def main():
    print("🚀 Запуск теста...")
    asyncio.run(process_results_async(test_results))
    print("✅ Тест завершён")

if __name__ == "__main__":
    main()
