import pytest
import psycopg2


@pytest.fixture
def auth_url():
    return "http://localhost:8081"

# function	Выполняется после каждого отдельного теста (функции). Самый чистый, но медленный способ.
# module	Выполняется один раз в конце каждого файла с тестами. Если в файле 10 тестов, очистка сработает 1 раз после 10-го.
# package	Выполняется один раз после завершения всех тестов в конкретной папке.
# session	Выполняется всего один раз, когда вообще все тесты во всех папках завершены.


# autouse=True значит, что фикстуру не надо прописывать в каждом тесте вручную
@pytest.fixture(scope="module", autouse=True)
def clear_sql_db():
    # Код до yield (выполняется перед первым тестом)
    print("\n[Start] Тесты запущены...")

    yield

    # Код после yield (выполняется после всех тестов в файле)
    print("\n[Cleanup] Начинаю очистку баз данных...")

    # Очистка db_users
    try:
        conn = psycopg2.connect(dbname="db_users", user="postgres", password="1111", host="localhost", port=5434)
        with conn.cursor() as cursor:
            cursor.execute("TRUNCATE TABLE users RESTART IDENTITY CASCADE;")
            conn.commit()
        conn.close()
    except Exception as e:
        print(f"Ошибка при чистке db_users: {e}")

    # Очистка db_rooms
    try:
        conn = psycopg2.connect(dbname="db_rooms", user="postgres", password="1111", host="localhost", port=5433)
        with conn.cursor() as cursor:
            cursor.execute("TRUNCATE TABLE rooms RESTART IDENTITY CASCADE;")
            conn.commit()
        conn.close()
    except Exception as e:
        print(f"Ошибка при чистке db_rooms: {e}")

    print('База данных успешно очищена')
