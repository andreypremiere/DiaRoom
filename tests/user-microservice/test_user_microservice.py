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

# Id с которым будет создаваться пользователь
CONST_UUID = "2d0ed77e-81e1-4b71-b59d-27e60107e49a"


@pytest.mark.parametrize("payload, expected_status, expected_value", [
    # Позитивный сценарий (пользователь и комната будут созданы)
    ({"id": CONST_UUID, "numberPhone": "+79233283528", "roomId": "unique_room Id", "roomName": "My room Name"},
     200, "id"),
    # Негативный: существующий номер телефона
    ({"numberPhone": "+79233283528", "roomId": "unique_room", "roomName": "My room Name"},
     404, "error"),
    # Негативный: существующий id room
    ({"numberPhone": "+79233283527", "roomId": "unique_room Id", "roomName": "My room Name"},
     404, "error"),
    # Негативный: номер телефона отсутствует
    ({"roomId": "unique_room2", "roomName": "My room Name"},
     404, "error"),
    # Негативный: отсутствует id комнаты
    ({"numberPhone": "+79233283526", "roomName": "My room Name"},
     404, "error"),
    # Пришли не те параметры
    ({"number_phone": "+79233283529", "room_id": "unique_room3", "room_name": "My room Name"},
     400, "error")
])
def test_register(base_url, payload, expected_status, expected_value):
    """
    Тестирование регистрации пользователя и создания комнаты через API.

    :param base_url: Базовый адрес хоста тестируемого сервиса.
    :param payload: Тело JSON-запроса с данными пользователя и комнаты.
    :param expected_status: Ожидаемый HTTP статус-код ответа (200, 400, 404).
    :param expected_value: Ключ, который должен присутствовать в JSON-ответе (id или error).
    :return: None
    """
    # Отправляем запрос
    response = requests.post(f"{base_url}/auth/newUser", json=payload)

    # Проверяем статус-код
    assert response.status_code == expected_status

    # Проверяем тип данных (Content-Type)
    assert response.headers["Content-Type"] == "application/json"

    data = response.json()

    # Проверяем наличие и тип параметров в JSON
    assert expected_value in data
    assert data[expected_value] != ""


def get_code_from_db(user_id):
    """
    Получение проверочного кода пользователя из базы данных Redis.

    :param user_id: Идентификатор пользователя, используемый в качестве ключа.
    :return: Строковое значение кода, если ключ найден, иначе None.
    """
    # Подключаемся к Redis
    # decode_responses=True преобразует байты в обычную строку (utf-8) автоматически
    r = redis.Redis(host='localhost', port=6379, db=0, decode_responses=True)

    # Формируем ключ
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
    """
    Тестирование авторизации пользователя по номеру телефона или ID комнаты.

    :param base_url: Базовый адрес хоста тестируемого сервиса.
    :param value: Идентификатор для входа (номер телефона или идентификатор комнаты).
    :param expected_status: Ожидаемый HTTP статус-код ответа (200 или 400).
    :param expected_value: Ключ, который должен присутствовать в JSON-ответе (userId или error).
    :return: None
    """
    response = requests.post(f"{base_url}/auth/login", json={"value": value})

    assert response.status_code == expected_status

    assert response.headers["Content-Type"] == "application/json"

    data = response.json()

    assert expected_value in data
    assert data[expected_value] != ""


@pytest.mark.parametrize("payload, code_type, expected_status, expected_value, wrong_code", [
    ({"userId": CONST_UUID}, '', 200, "token", False),
    ({"userId": CONST_UUID}, '', 400, "error", True),
    ({"userId": "ljflsjif-flkjwf-jlfjs"}, None, 400, "error", False)
])
def test_verify_user_by_id(base_url, payload, code_type, expected_status, expected_value, wrong_code):
    """
    Тестирование верификации пользователя по ID с проверкой кода из Redis и сохранением токена.

    :param base_url: Базовый адрес хоста тестируемого сервиса.
    :param payload: Тело запроса, содержащее идентификатор пользователя.
    :param code_type: Ожидаемый тип данных кода для проверки (используется для assert type).
    :param expected_status: Ожидаемый HTTP статус-код ответа.
    :param expected_value: Ключ в ответе, который нужно проверить (token или error).
    :param wrong_code: Флаг подстановки неверного кода ('000') для негативных сценариев.
    :return: None
    """
    code = get_code_from_db(payload["userId"])
    if code is not None:
        if not wrong_code:
            payload["code"] = code
        else:
            payload["code"] = '000'

    assert type(code) == type(code_type)

    response = requests.post(f"{base_url}/auth/verifyUser", json=payload)

    assert response.status_code == expected_status

    assert response.headers["Content-Type"] == "application/json"

    data = response.json()

    assert expected_value in data
    assert data[expected_value] != ""

    # Сохраняем токен для дальнейших тестов
    if expected_status == 200 and expected_value == "token":
        token = data[expected_value]
        test_context["token"] = token

        # Декодируем JWT без проверки секретного ключа (только Payload)
        decoded_jwt = jwt.decode(token, options={"verify_signature": False})

        test_context["user_id"] = decoded_jwt.get("user_id")
        test_context["room_id"] = decoded_jwt.get("room_id")


@pytest.mark.parametrize("method, use_context_id, custom_payload, content_type, expected_status, expected_key", [
    ("POST", True, None, "application/json", 200, "id"),
    ("GET", True, None, "application/json", 405, "error"),
    ("POST", True, None, "text/plain", 415, "error"),
    ("POST", False, {"roomId": "12345"}, "application/json", 400, "error"),
    ("POST", False, {"roomId": "ffffffff-ffff-ffff-ffff-ffffffffffff"}, "application/json", 500, "error"),
])
def test_get_room_by_room_id(base_url, method, use_context_id, custom_payload, content_type, expected_status,
                             expected_key):
    """
    Тестирование эндпоинта получения данных комнаты с проверкой авторизации и различных HTTP-методов.

    :param base_url: Базовый адрес хоста тестируемого сервиса.
    :param method: HTTP-метод запроса (POST, GET и т.д.) для проверки ограничений API.
    :param use_context_id: Флаг использования room_id, полученного из предыдущих тестов верификации.
    :param custom_payload: Пользовательское тело запроса для негативных сценариев (невалидные ID).
    :param content_type: Значение заголовка Content-Type для проверки валидации типа данных сервером.
    :param expected_status: Ожидаемый HTTP статус-код (200, 400, 405, 415, 500).
    :param expected_key: Ключ в JSON-ответе, наличие которого необходимо проверить (id или error).
    :return: None
    """
    if not test_context["room_id"]:
        pytest.skip("Пропущен: нет room_id из теста авторизации")

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
    ("POST", {"Content-Type": "application/json"}, {"userId": CONST_UUID}, 200, "roomId"),
    ("GET", {"Content-Type": "application/json"}, {"userId": CONST_UUID}, 405, "error"),
    ("POST", {"Content-Type": "text/plain"}, {"userId": CONST_UUID}, 415, "error"),
    ("POST", {}, {"userId": CONST_UUID}, 415, "error"),
    ("POST", {"Content-Type": "application/json"}, {"userId": "not-a-valid-uuid"}, 400, "error"),
    ("POST", {"Content-Type": "application/json"}, {"wrongKey": CONST_UUID}, 500, "error"),
    ("POST", {"Content-Type": "application/json"}, {"userId": "00000000-0000-0000-0000-000000000001"}, 500, "error"),
])
def test_get_room_id_by_user_id(base_url, method, headers, payload, expected_status, expected_key):
    """
    Тестирование получения ID комнаты по ID пользователя.

    :param base_url: Базовый адрес хоста тестируемого сервиса.
    :param method: Используемый HTTP-метод (проверка на 405 Method Not Allowed).
    :param headers: Словарь заголовков запроса (проверка Content-Type и 415 error).
    :param payload: Тело запроса с идентификатором пользователя или некорректными ключами.
    :param expected_status: Ожидаемый HTTP статус-код ответа сервера.
    :param expected_key: Ожидаемый ключ в JSON-ответе (roomId или error).
    :return: None
    """
    json_data = json.dumps(payload)

    response = requests.request(
        method=method,
        url=f"{base_url}/rooms/getRoomIdByUserId",
        headers=headers,
        data=json_data
    )

    assert response.status_code == expected_status
    assert response.headers["Content-Type"] == "application/json"

    data = response.json()
    assert expected_key in data
    assert data[expected_key] != ""