import json

import jwt
import pytest
import requests
import redis

# Глобальное хранилище состояния между тестами
test_context = {
    "token": None,
    "user_id": None,
    "room_id": None
}

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
def test_register(base_url, payload, expected_status, expected_value):
    # 1. Отправляем запрос
    response = requests.post(f"{base_url}/auth/newUser", json=payload)

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


@pytest.mark.parametrize("value, expected_status, expected_value",
                         [
                             ('+79233283528', 200, "userId"),
                             ('unique_room Id', 200, "userId"),
                             ('unique_room ', 400, "error"),
                         ])
def test_login_user(base_url, value, expected_status, expected_value):
    response = requests.post(f"{base_url}/auth/login", json={"value": value})

    # 2. Проверяем статус-код
    assert response.status_code == expected_status

    # 3. Проверяем тип данных (Content-Type)
    assert response.headers["Content-Type"] == "application/json"

    data = response.json()

    # 4. Проверяем наличие и тип параметров в JSON
    assert expected_value in data
    assert data[expected_value] != ""


@pytest.mark.parametrize("payload, code_type, expected_status, expected_value, wrong_code", [
    ({"userId": CONST_UUID}, '', 200, "token", False),
    ({"userId": CONST_UUID}, '', 400, "error", True),
    ({"userId": "ljflsjif-flkjwf-jlfjs"}, None, 400, "error", False)
])
def test_verify_user_by_id(base_url, payload, code_type, expected_status, expected_value, wrong_code):
    code = get_code_from_db(payload["userId"])
    if code is not None:
        if not wrong_code:
            payload["code"] = code
        else:
            payload["code"] = '000'

    assert type(code) == type(code_type)
    # 1. Отправляем запрос
    response = requests.post(f"{base_url}/auth/verifyUser", json=payload)

    # 2. Проверяем статус-код
    assert response.status_code == expected_status

    # 3. Проверяем тип данных (Content-Type)
    assert response.headers["Content-Type"] == "application/json"

    data = response.json()

    # 4. Проверяем наличие и тип параметров в JSON
    assert expected_value in data
    assert data[expected_value] != ""

    if expected_status == 200 and expected_value == "token":
        token = data[expected_value]
        test_context["token"] = token

        # Декодируем JWT без проверки секретного ключа (нам нужен только Payload)
        decoded_jwt = jwt.decode(token, options={"verify_signature": False})

        # Внимательно проверь, как именно ключи называются в твоем JWT (user_id или userId)
        test_context["user_id"] = decoded_jwt.get("user_id")
        test_context["room_id"] = decoded_jwt.get("room_id")


@pytest.mark.parametrize("method, use_context_id, custom_payload, content_type, expected_status, expected_key", [
    # 1. Позитивный сценарий (берем валидный ID из контекста)
    ("POST", True, None, "application/json", 200, "id"),

    # 2. Негативный: неверный HTTP метод
    ("GET", True, None, "application/json", 405, "error"),

    # 3. Негативный: неверный Content-Type
    ("POST", True, None, "text/plain", 415, "error"),

    # 4. Негативный: невалидный формат UUID
    ("POST", False, {"roomId": "12345"}, "application/json", 400, "error"),

    # 5. Негативный: несуществующая комната
    ("POST", False, {"roomId": "ffffffff-ffff-ffff-ffff-ffffffffffff"}, "application/json", 500, "error"),
])
def test_get_room_by_room_id(base_url, method, use_context_id, custom_payload, content_type, expected_status,
                             expected_key):
    if not test_context["room_id"]:
        pytest.skip("Пропущен: нет room_id из теста авторизации")

    # Формируем тело запроса
    if use_context_id:
        payload = {"roomId": test_context["room_id"]}
    else:
        payload = custom_payload

    headers = {"Content-Type": content_type}
    if test_context["token"]:
        headers["Authorization"] = f"Bearer {test_context['token']}"

    response = requests.request(
        method=method,
        url=f"{base_url}/rooms/getRoomByRoomId",
        headers=headers,
        json=payload
    )

    assert response.status_code == expected_status
    data = response.json()
    assert expected_key in data

    if expected_status == 200:
        assert data[expected_key] == test_context["room_id"]


@pytest.mark.parametrize("method, headers, payload, expected_status, expected_key", [
    # 1. Позитивный сценарий
    ("POST", {"Content-Type": "application/json"}, {"userId": CONST_UUID}, 200, "roomId"),

    # 2. Негативный: неверный HTTP метод (ожидаем 405 Method Not Allowed)
    ("GET", {"Content-Type": "application/json"}, {"userId": CONST_UUID}, 405, "error"),

    # 3. Негативный: неверный Content-Type (ожидаем 415 Unsupported Media Type)
    ("POST", {"Content-Type": "text/plain"}, {"userId": CONST_UUID}, 415, "error"),
    ("POST", {}, {"userId": CONST_UUID}, 415, "error"),  # Вообще без заголовка

    # 4. Негативный: ошибка парсинга (передан не UUID, а обычная строка)
    ("POST", {"Content-Type": "application/json"}, {"userId": "not-a-valid-uuid"}, 400, "error"),

    # 5. Негативный: неверный ключ в JSON (приведет к дефолтному UUID 00000000... и ошибке БД)
    ("POST", {"Content-Type": "application/json"}, {"wrongKey": CONST_UUID}, 500, "error"),

    # 6. Негативный: пользователя не существует (судя по твоему коду Go, вернется 500 ошибка)
    ("POST", {"Content-Type": "application/json"}, {"userId": "00000000-0000-0000-0000-000000000001"}, 500, "error"),
])
def test_get_room_id_by_user_id(base_url, method, headers, payload, expected_status, expected_key):
    # Используем requests.request, чтобы динамически менять метод (POST/GET)
    json_data = json.dumps(payload)

    response = requests.request(
        method=method,
        url=f"{base_url}/rooms/getRoomIdByUserId",  # Подставь правильный роут
        headers=headers,
        data=json_data
    )

    assert response.status_code == expected_status
    assert response.headers["Content-Type"] == "application/json"

    data = response.json()
    assert expected_key in data
    assert data[expected_key] != ""