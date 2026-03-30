import time

import redis

REDIS_HOST = 'localhost'
REDIS_PORT = 6380
QUEUE_KEY = "new_posts:post_id"


def fill_queue():
    r = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    print(f"🚀 Начинаем заполнение тестирование...")
    try:
        while True:
            for i in range(22):
                r.lpush(QUEUE_KEY, '6132b35b-537c-45e3-866e-6a440f1c2d23')
                r.lpush(QUEUE_KEY, '1cbae323-b25f-4fb0-89fb-bb81df445fce')
            time.sleep(1)
            print(f"\n✨ Готово! В очереди сейчас элементов: {r.llen(QUEUE_KEY)}")

    except Exception as e:
        print(f"❌ Ошибка подключения: {e}")


if __name__ == "__main__":
    fill_queue()

    cache_list_batch4 = [
        {'post_id': 40001, "full_text": "Классный закат.", "result": None, "score": None},
        {'post_id': 40002, "full_text": "Люблю свою работу.", "result": None, "score": None},
        {'post_id': 40003, "full_text": "Сосёт как богиня.", "result": None, "score": None},
        {'post_id': 40004, "full_text": "Горячее порно 18+", "result": None, "score": None},
        {'post_id': 40005, "full_text": "Погода в Амстердаме сегодня идеальная. Солнце, +18, можно гулять без куртки.",
         "result": None, "score": None},
        {'post_id': 40006, "full_text": "Ищу пару для свинга. М и Ж, 25-35 лет. Полный контакт, без табу.",
         "result": None, "score": None},
        {'post_id': 40007,
         "full_text": "Рассказываю про свой новый бизнес. Открыл маленькую кофейню. Уже второй месяц, выручка растёт. Клиенты хвалят кофе и выпечку. Планы — открыть ещё одну точку через полгода.",
         "result": None, "score": None},
        {'post_id': 40008,
         "full_text": "Длинный порно-рассказ. Она пришла к нему домой в одном плаще. Под ним — только чулки и ничего больше. Он сразу прижал её к стене, задрал плащ и вошёл в неё стоя. Трахал грубо, пока она не закричала. Потом на кухне, в спальне, в ванной. Кончал в рот, на грудь, внутрь. Она кончала каждый раз, когда он входил в неё.",
         "result": None, "score": None},
        {'post_id': 40009, "full_text": "Поздравляю с праздником!", "result": None, "score": None},
        {'post_id': 40010, "full_text": "Хочу чтобы меня оттрахали все.", "result": None, "score": None},
        {'post_id': 40011,
         "full_text": "Сегодня научился новому рецепту пасты. С морепродуктами и белым вином. Получилось ресторанное блюдо дома!",
         "result": None, "score": None},
        {'post_id': 40012, "full_text": "Продаю интим-фото и видео. 1000+ контента, любой фетиш. Пишите в ЛС.",
         "result": None, "score": None},
        {'post_id': 40013,
         "full_text": "Очень большой пост про путешествия по Европе. За 3 недели посетил 5 стран: Германия, Франция, Италия, Испания, Португалия. Поезда, самолёты, хостелы, музеи, еда. Каждый день новые впечатления. Лучшее — закат в Лиссабоне. Если планируешь trip — спрашивайте советы, я всё расписал по дням и бюджетам.",
         "result": None, "score": None},
        {'post_id': 40014,
         "full_text": "Мега-длинный NSFW. Они устроили настоящую оргию. Три девушки и два парня. Все голые. Минеты, куни, двойное проникновение, сперма на лицах и телах. Стонали и кричали всю ночь. Каждый кончил по несколько раз. Это было самое грязное и приятное, что я видел.",
         "result": None, "score": None},
        {'post_id': 40015, "full_text": "Всё будет хорошо.", "result": None, "score": None},
        {'post_id': 40016, "full_text": "Порно-марафон до утра.", "result": None, "score": None}
    ]

    safe_base1 = "Мой отпуск в Крыму был просто сказочным. Мы прилетели в Симферополь, потом на такси до Ялты. Сняли маленький домик прямо у моря. Каждое утро начиналось с чашки кофе на террасе и вида на волны. Днём купались, загорали, вечером гуляли по набережной, ели мороженое и слушали живую музыку в кафе. Один день мы поднялись в горы на машине — виды просто захватывали дух. Я сделал больше 500 фотографий. Если кто-то думает, куда поехать летом — однозначно Крым! Рекомендую всем. Это было лучшее путешествие за последние 5 лет."

    safe_base2 = "Hey everyone! Just pushed the latest updates to the DiaRoom backend. Switched the microservices communication to gRPC for better performance and finally fixed that annoying race condition in the Go worker. It was a tough nut to crack, but the results are worth it. Next step: polishing the Flutter UI to make the 'Workshop' section look more cinematic. If you're a fellow dev or a digital artist, hit me up! Let's build something awesome together."

    nsfw_base1 = "Она медленно сняла с себя платье. Её тело блестело в свете лампы. Он подошёл ближе, его руки жадно скользнули по её груди, спустились ниже, к бёдрам. Она застонала, когда его пальцы вошли в неё. Они упали на кровать. Он вошёл в неё резко и глубоко. Она кричала от удовольствия, пока он трахал её всё быстрее и сильнее. Потом он перевернул её раком и вошёл сзади, шлёпая по попе. Кончили одновременно, всё залито спермой. Лучший секс в моей жизни."

    nsfw_base2 = "Полный текст эротического рассказа на 700+ слов. Она лежала голая на кровати, раздвинув ноги. Он опустился на колени и начал лизать её мокрую киску. Язык кружил вокруг клитора, пальцы входили всё глубже. Она извивалась и стонала: «Ещё, сильнее!» Он встал, достал свой толстый хуй и резко вошёл в неё. Трахал жёстко, держа за волосы. Потом вытащил и кончил ей на лицо и в рот. Она всё проглотила и попросила ещё. Они занимались сексом всю ночь: в миссионерской, раком, наездницей, 69. Сперма была везде — на груди, на животе, внутри. Это был настоящий порно-марафон."

    # === САМ БАТЧ (16 элементов) ===
    cache_list_batch5 = [
        {'post_id': 50001, "full_text": "Привет всем!", "result": None, "score": None},
        {'post_id': 50002, "full_text": "Хорошая погода сегодня.", "result": None, "score": None},
        {'post_id': 50003, "full_text": "Хочу секса прямо сейчас.", "result": None, "score": None},
        {'post_id': 50004, "full_text": "Порно на любой вкус.", "result": None, "score": None},

        # Средние (для разнообразия)
        {'post_id': 50005,
         "full_text": "Сегодня приготовил борщ по бабушкиному рецепту. Мясо, свекла, капуста, морковь. Получилось невероятно вкусно! Все соседи просили рецепт.",
         "result": None, "score": None},
        {'post_id': 50006,
         "full_text": "Вчера смотрел порно всю ночь. Там две девки друг друга вылизывали, а потом пришел парень и всех оттрахал.",
         "result": None, "score": None},

        # ОГРОМНЫЕ ТЕКСТЫ (8 штук, по 10 000+ символов)
        {'post_id': 50007, "full_text": safe_base1 * 18, "result": None, "score": None},  # ~11 700 символов — safe
        {'post_id': 50008, "full_text": nsfw_base1 * 16, "result": None, "score": None},  # ~11 200 символов — NSFW
        {'post_id': 50009, "full_text": safe_base2, "result": None, "score": None},  # ~11 400 символов — safe
        {'post_id': 50010, "full_text": nsfw_base2 * 15, "result": None, "score": None},  # ~12 000 символов — NSFW
        {'post_id': 50011, "full_text": safe_base1 * 17, "result": None, "score": None},  # ~11 050 символов — safe
        {'post_id': 50012, "full_text": nsfw_base1 * 18, "result": None, "score": None},  # ~12 600 символов — NSFW
        {'post_id': 50013, "full_text": safe_base2 * 18, "result": None, "score": None},  # ~10 800 символов — safe
        {'post_id': 50014, "full_text": nsfw_base2 * 17, "result": None, "score": None},  # ~13 600 символов — NSFW
        {'post_id': 50015, "full_text": "Спокойной ночи, друзья!", "result": None, "score": None},
        {'post_id': 50016, "full_text": "Хочу чтобы меня отымели втроём.", "result": None, "score": None}
    ]

    # start_time = time.perf_counter()
    # result = analyze_text(cache_list_batch4)
    # end_time = time.perf_counter()   # Фиксируем конец
    #
    # for i in result:
    #     print(i)
    #
    # duration = end_time - start_time
    # print(f"⏱️ Функция analyze_text выполнена за {duration:.4f} сек.", flush=True)