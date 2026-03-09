import boto3
import os
import time
from botocore.exceptions import NoCredentialsError

# Настройки
ACCESS_KEY = 'Написать свой'
SECRET_KEY = 'Написать свой'
BUCKET_NAME = 'avatars-diaroom-1'
ENDPOINT = 'https://storage.yandexcloud.net'
REGION = 'ru-central1'


def upload_avatar(local_file_path, user_id):
    # Инициализация клиента S3
    session = boto3.session.Session()
    s3 = session.client(
        service_name='s3',
        endpoint_url=ENDPOINT,
        region_name=REGION,
        aws_access_key_id=ACCESS_KEY,
        aws_secret_access_key=SECRET_KEY
    )

    try:
        # Формируем название файла. Извлекаем расширение файла
        _, file_extension = os.path.splitext(local_file_path)

        timestamp = int(time.time())

        # Собираем путь внутри бакета: avatars/ID/ID_TIMESTAMP.ext
        # Это поможет нам избежать коллизий и удобно чистить старые файлы
        object_name = f"avatars/{user_id}/{user_id}_{timestamp}{file_extension.lower()}"
        # object_name = f"avatars/{user_id}/{user_id}{file_extension.lower()}"
        print(f"Загружаем {local_file_path} как {object_name}...")

        content_type = "image/jpeg" if file_extension.lower() in ['.jpg', '.jpeg'] else "image/png"

        # Загрузка файла
        s3.upload_file(
            local_file_path,
            BUCKET_NAME,
            object_name,
            ExtraArgs={'ContentType': content_type}
        )

        # Формируем публичную ссылку (если чтение в хранилище доступно)
        file_url = f"{ENDPOINT}/{BUCKET_NAME}/{object_name}"

        print("Файл успешно загружен!")
        print(f"Ссылка на файл: {file_url}")
        return file_url

    except FileNotFoundError:
        print("Ошибка: Локальный файл не найден.")
    except NoCredentialsError:
        print("Ошибка: Проверьте ключи доступа.")
    except Exception as e:
        print(f"Произошла ошибка: {e}")


if __name__ == "__main__":
    # Путь к файлу на пк, который будем сохранять
    TEST_FILE = r"C:\Users\andre\Downloads\default.jpg"
    USER_ID = "Написать user_id"

    upload_avatar(TEST_FILE, USER_ID)