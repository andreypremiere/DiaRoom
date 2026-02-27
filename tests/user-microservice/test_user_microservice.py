import pytest
import requests
import redis

CONST_UUID = "2d0ed77e-81e1-4b71-b59d-27e60107e49a"

@pytest.mark.parametrize("payload, expected_status, expected_value", [
    # 1. Позитивный сценарий
    ({"id": CONST_UUID, "numberPhone": "+79233283528", "roomId": "unique_room Id", "roomName": "My room Name"},
     200, "id"),
    # 2. Негативный: существующий номер телефона
    ({"numberPhone": "+79233283528", "roomId": "unique_room", "roomName": "My room Name"},
     404, "error"),
    # 3. Негативный: существующий id room
    ({"numberPhone": "+79233283527", "roomId": "unique_room Id", "roomName": "My room Name"},
     404, "error"),
    # 4. Негативный: номер телефона отсутствует
    ({"roomId": "unique_room2", "roomName": "My room Name"},
     404, "error"),
    # 5. Негативный: отсутствует id комнаты
    ({"numberPhone": "+79233283526", "roomName": "My room Name"},
     404, "error"),
    # 6. Пришли не те параметры
    ({"number_phone": "+79233283529", "room_id": "unique_room3", "room_name": "My room Name"},
     400, "error")
])
def test_register(auth_url, payload, expected_status, expected_value):
    # 1. Отправляем запрос
    response = requests.post(f"{auth_url}/newUser", json=payload)

    # 2. Проверяем статус-код
    assert response.status_code == expected_status

    # 3. Проверяем тип данных (Content-Type)
    assert response.headers["Content-Type"] == "application/json"

    data = response.json()

    # 4. Проверяем наличие и тип параметров в JSON
    assert expected_value in data
    assert data[expected_value] != ""
    # assert isinstance(data["jwt"], str)
    # assert data["user_id"] > 0


def get_code_from_db(user_id):
    # Подключаемся к Redis
    # decode_responses=True преобразует байты в обычную строку (utf-8) автоматически
    r = redis.Redis(host='localhost', port=6379, db=0, decode_responses=True)

    # Формируем ключ (уточните в коде микросервиса, какой там формат ключа)
    # Например: f"code:{user_id}"
    key = f"{user_id}"

    code = r.get(key)

    if code is None:
        return None

    return code


@pytest.mark.parametrize("payload, code_type, expected_status, expected_value, wrong_code", [
    ({"userId": CONST_UUID}, '', 200, "jwt", False),
    ({"userId": CONST_UUID}, '', 400, "error", True),
    ({"userId": "ljflsjif-flkjwf-jlfjs"}, None, 400, "error", False)
])
def test_verify_user_by_id(auth_url, payload, code_type, expected_status, expected_value, wrong_code):
    code = get_code_from_db(payload["userId"])
    if code is not None:
        if not wrong_code:
            payload["code"] = code
        else:
            payload["code"] = '000'

    assert type(code) == type(code_type)
    # 1. Отправляем запрос
    response = requests.post(f"{auth_url}/verifyUser", json=payload)

    # 2. Проверяем статус-код
    assert response.status_code == expected_status

    # 3. Проверяем тип данных (Content-Type)
    assert response.headers["Content-Type"] == "application/json"

    data = response.json()

    # 4. Проверяем наличие и тип параметров в JSON
    assert expected_value in data
    assert data[expected_value] != ""

