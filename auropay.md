Aurapay API (1.0.2)
Download OpenAPI specification:Download

URL: https://app.aurapay.tech
API для интеграции платежной системы Aurapay.

Аутентификация
Все запросы к API требуют использования следующих заголовков:

X-ApiKey - ваш API ключ
X-ShopId - идентификатор вашей кассы
Доступные сервисы оплат
card - Оплата с помощью карты
sbp - Оплата с помощью СБП
Инвойсы
Операции по созданию и проверке статуса инвойсов

Создание инвойса

post
/invoice/create

Создает новый инвойс для проведения платежей. После создания инвойса пользователь может перейти по ссылке из поля payment_data.url для оплаты.

Важно:

order_id должен быть уникальным в рамках вашей кассы
Максимальное время жизни инвойса (lifetime) - 43200 минут (30 дней)
Если не указан lifetime, инвойс будет действителен в течение 60 минут
Authorizations:
ApiKeyAuthShopIdAuth
Request Body schema: application/json
required
amount
required
number >= 0.01
Сумма счета в рублях

order_id
required
string
Уникальный идентификатор заказа в вашей системе. Должен быть уникальным в рамках вашей кассы.

success_url
string <uri>
URL для перенаправления пользователя после успешной оплаты

fail_url
string <uri>
URL для перенаправления пользователя после неудачной оплаты

callback_url
string <uri>
URL для отправки webhook-уведомлений о статусе платежа

custom_fields
string
Дополнительные поля, которые будут возвращены в webhook-уведомлении. Могут использоваться для передачи внутренних идентификаторов или другой информации.

comment
string
Комментарий к платежу, который будет отображаться пользователю

lifetime
integer [ 1 .. 43200 ]
Время жизни инвойса в минутах. Максимальное значение - 43200 минут (30 дней). По умолчанию - 60 минут.

service
string
Enum: "card" "sbp"
Предпочтительный способ оплаты. Если не указан, пользователь сможет выбрать способ оплаты самостоятельно.

card - Оплата банковской картой
sbp - Оплата через Систему Быстрых Платежей (СБП)
Responses
200 Инвойс успешно создан
400 Ошибка валидации или бизнес-логики
Request samples
Payload
Content type
application/json

Copy
{
"amount": 1000.5,
"order_id": "ORDER-12345",
"success_url": "https://example.com/success",
"fail_url": "https://example.com/fail",
"callback_url": "https://example.com/webhook",
"custom_fields": "user_id=123&subscription_id=456",
"comment": "Пополнение баланса #123312",
"lifetime": 60,
"service": "sbp"
}
Response samples
200400
Content type
application/json

Copy
Expand allCollapse all
{
"id": "3ed6f489-e8fe-4695-8846-9e6f0dbc3b25",
"order_id": "1234",
"shop_id": "17f9db92-1c87-4ef1-9281-45e1ab74ae96",
"amount": 1,
"comment": "Пополнение баланса #123312",
"service": null,
"expires_at": "2026-01-27 23:31:02",
"created_at": "2026-01-27 22:31:02",
"status": "PENDING",
"payment_data": {
"url": "https://pay.aurapay.tech/3ed6f489-e8fe-4695-8846-9e6f0dbc3b25"
}
}
Проверка статуса инвойса

post
/invoice/status

Позволяет проверить текущий статус существующего инвойса.

Для проверки необходимо указать либо id (UUID инвойса), либо order_id (идентификатор заказа в вашей системе).

Статусы инвойса:

PENDING - Инвойс создан и ожидает оплаты
PAID - Инвойс успешно оплачен
EXPIRED - Срок действия инвойса истек
Authorizations:
ApiKeyAuthShopIdAuth
Request Body schema: application/json
required
One of objectobject
id
required
string <uuid>
Уникальный идентификатор инвойса

Responses
200 Информация об инвойсе успешно получена
404 Инвойс не найден
Request samples
Payload
Content type
application/json

Copy
{
"order_id": "123123"
}
Response samples
200404
Content type
application/json

Copy
Expand allCollapse all
{
"id": "f5bf451e-e1a1-43cd-96e2-a1a13f800080",
"order_id": "123123",
"shop_id": "17f9db92-1c87-4ef1-9281-45e1ab74ae96",
"amount": 1000,
"comment": "Пополнение баланса #123312",
"service": null,
"expires_at": "2026-01-23 18:35:38",
"created_at": "2026-01-23 17:35:38",
"status": "PAID",
"payment_data": {
"url": "https://pay.aurapay.tech/f5bf451e-e1a1-43cd-96e2-a1a13f800080"
}
}
Магазин
Операции с данными магазина

Получение баланса магазина

get
/shop/balance

Возвращает текущий баланс вашей кассы.

Балансы:

balance - Доступный баланс для вывода
balance_hold - Заблокированные средства (в процессе обработки)
Authorizations:
ApiKeyAuthShopIdAuth
Responses
200 Баланс успешно получен
400 Ошибка при получении баланса
Response samples
200400
Content type
application/json

Copy
{
"balance": 1187.5,
"balance_hold": 0
}
Выплаты
Операции по созданию выплат и проверке их статуса.

Сервисы выплат (service):

sbp — вывод через Систему быстрых платежей (СБП)
card_ru — вывод на банковскую карту (РФ)
usdt-trc20 — вывод в USDT (сеть TRC-20)
Создание выплаты

post
/payout/create

Создаёт заявку на вывод средств с баланса кассы.

Важно:

order_id — ваш внутренний номер операции; должен быть уникальным в рамках кассы (не повторяться для разных выплат)
callback_url — по желанию: URL для POST-уведомлений о смене статуса выплаты (см. раздел Webhook для выплат в описании тега Webhook)
sbp_bank_id — идентификатор банка в СБП; указывается только при выводе через СБП (service: sbp)
subtract — с кого списывать комиссию: 0 — с суммы выплаты (по умолчанию), 1 — с баланса магазина
Authorizations:
ApiKeyAuthShopIdAuth
Request Body schema: application/json
required
amount
required
number
Сумма выплаты

order_id
required
string [ 1 .. 255 ] characters
Внутренний идентификатор операции выплаты в вашей системе; уникален в пределах кассы

callback_url
string or null <= 500 characters
Необязательный URL (http/https), на который отправляются POST-уведомления о статусе выплаты в формате JSON. Не длиннее 500 символов; подробности — в разделе «Webhook для выплат» (тег Webhook).

service
required
string
Enum: "sbp" "card_ru" "usdt-trc20"
Сервис вывода

wallet_to
required
string [ 3 .. 255 ] characters
Реквизиты для вывода (номер телефона, карта, адрес кошелька и т.д. в зависимости от service)

sbp_bank_id
string <= 255 characters
Идентификатор банка в Системе быстрых платежей (СБП). Нужен только при выводе через СБП (service: sbp)

subtract
integer
Default: 0
Enum: 0 1
С кого списывать комиссию: 0 — с суммы выплаты (по умолчанию), 1 — с баланса магазина

Responses
200 Выплата успешно создана
400 Ошибка валидации или бизнес-логики
Request samples
Payload
Content type
application/json

Copy
{
"amount": 1500.5,
"order_id": "PAYOUT-ORDER-001",
"callback_url": "https://example.com/webhook/payout",
"service": "sbp",
"wallet_to": "+79001234567",
"sbp_bank_id": "100000000004",
"subtract": 0
}
Response samples
200400
Content type
application/json

Copy
{
"id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
"order_id": "PAYOUT-ORDER-001",
"shop_id": "17f9db92-1c87-4ef1-9281-45e1ab74ae96",
"amount": 1500.5,
"amount_to_payout": 1485,
"commission": 15.5,
"service": "sbp",
"debited": 1500.5,
"wallet_to": "+79001234567"
}
Проверка статуса выплаты

post
/payout/status

Возвращает актуальные данные по выплате.

В теле запроса укажите ровно один идентификатор:

id — UUID выплаты в системе Aurapay (из ответа создания или webhook);
order_id — тот же внутренний номер операции, что вы передавали в POST /payout/create.
Статусы выплаты:

PROCESSING — в обработке
SUCCESS — успешно выполнена
ERROR — ошибка
Authorizations:
ApiKeyAuthShopIdAuth
Request Body schema: application/json
required
One of objectobject
id
required
string <uuid>
UUID выплаты в Aurapay (из ответа POST /payout/create или webhook)

Responses
200 Информация о выплате успешно получена
404 Выплата не найдена
Request samples
Payload
Content type
application/json

Copy
{
"order_id": "PAYOUT-ORDER-001"
}
Response samples
200404
Content type
application/json

Copy
{
"id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
"order_id": "PAYOUT-ORDER-001",
"shop_id": "17f9db92-1c87-4ef1-9281-45e1ab74ae96",
"amount": 1500.5,
"amount_to_payout": 1485,
"commission": 15.5,
"service": "sbp",
"debited": 1500.5,
"wallet_to": "+79001234567",
"status": "SUCCESS"
}
Webhook
Webhook-уведомления
URL для Webhook
Указывается в настройках кассы или при создании инвойса (параметр callback_url).

Заголовки запроса
Webhook-запрос содержит следующие заголовки:

Content-Type: application/json
X-SIGNATURE: [сгенерированная сигнатура] - сигнатура для проверки подлинности запроса
Пример тела запроса (JSON)
{
"id": "9beea835-0937-4b5c-8f5a-c3a0d0e60346",
"amount": "1250.00",
"status": "PAID",
"comment": "test",
"created_at": "2024-01-30 19:07:45",
"expires_at": "2024-01-30 20:07:45",
"service": "sbp",
"payer_details": "794**\***254",
"payer_ip": "37.251.8.13",
"shop_id": "9bee9309-2585-4332-b63d-e1d897f7ce84",
"order_id": "11113",
"custom_fields": null
}
Описание полей
Поле Тип Описание
id String (UUID) Уникальный идентификатор инвойса
amount String Сумма платежа в рублях (формат: "1250.00")
status String Статус платежа: PAID (оплачен), EXPIRED (истек), PENDING (ожидает оплаты)
comment String Комментарий к платежу, переданный при создании инвойса
created_at String Дата и время создания инвойса (формат: "YYYY-MM-DD HH:MM:SS")
expires_at String Дата и время истечения срока действия инвойса (формат: "YYYY-MM-DD HH:MM:SS")
service String Способ оплаты: card (банковская карта) или sbp (СБП)
payer_details String Детали плательщика (маскированный формат)
payer_ip String IP-адрес плательщика
shop_id String (UUID) Идентификатор вашей кассы
order_id String Уникальный идентификатор заказа в вашей системе
custom_fields String | null Дополнительные поля, переданные при создании инвойса (если были указаны)
Проверка подписи тела запроса выполняется с секретным ключом #2; алгоритм и примеры кода — в разделе Проверка сигнатуры в конце этой страницы.

Webhook для выплат
Адрес webhook. В теле запроса POST /payout/create можно передать необязательный параметр callback_url — полный URL вашего эндпоинта (схема http или https), на который Aurapay отправляет POST с телом JSON при изменении статуса выплаты. Строка URL не длиннее 500 символов; допускается хост в виде доменного имени или IPv4, при необходимости порт, путь и query-строка.

Заголовки. В каждом запросе передаются Content-Type: application/json и X-SIGNATURE — шестнадцатеричная подпись тела.

Проверка подписи — в разделе Проверка сигнатуры в конце этой страницы.

Пример тела запроса (JSON)
{
"id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
"order_id": "PAYOUT-10001",
"shop_id": "9bee9309-2585-4332-b63d-e1d897f7ce84",
"service": "sbp",
"amount": "1500.00",
"amount_to_payout": "1485.00",
"commission": "15.00",
"debited": "1500.00",
"wallet_to": "+79001234567",
"created_at": "2026-04-19 14:30:00",
"status": "SUCCESS"
}
Описание полей
Поле Тип Описание
id String (UUID) Уникальный идентификатор выплаты
order_id String Внутренний идентификатор операции выплаты в вашей системе — тот же, что вы передали при создании заявки
shop_id String (UUID) Идентификатор кассы (UUID), к которой относится выплата
service String Сервис вывода: sbp, card_ru, usdt-trc20
amount String Сумма выплаты
amount_to_payout String Сумма к выплате получателю
commission String Комиссия
debited String Списано с баланса магазина
wallet_to String Реквизиты вывода
created_at String Дата и время создания выплаты (формат: YYYY-MM-DD HH:MM:SS)
status String Статус выплаты (см. ниже)
Статусы выплат в webhook
Значение Описание
PROCESSING В обработке
SUCCESS Успешно выполнена
ERROR Ошибка
Один и тот же webhook может прийти несколько раз; обрабатывайте его идемпотентно — фиксируйте обработку по id или order_id, чтобы не выполнять бизнес-операцию повторно.

Проверка сигнатуры
Один алгоритм для webhook по инвойсам и по выплатам: параметры JSON сортируются по ключам в алфавитном порядке, все значения объединяются в одну строку, строка хешируется HMAC-SHA256 с секретным ключом #2 из настроек кассы; результат сравнивается с заголовком X-SIGNATURE (предпочтительно сравнение с защитой от timing-атак).

Пример заголовка с сигнатурой:

X-SIGNATURE: 6eff1010e1c441acf342510288a0acec6f8ae9e0fc5264eb9fff7371ee80667b
Пример скрипта на PHP

<?php
// Секретный ключ #2
$secretKey2 = "";

// Получение параметров в хука
$params = json_decode(file_get_contents("php://input"), true);

// Сортировка массива по ключам в алфавитном порядке
ksort($params);

// Объединение всех значений массива в одну строку
$concatenatedString = implode('', $params);

// Генерация хеша с использованием HMAC-SHA256 и секретного ключа
$generatedSignature = hash_hmac("SHA256", $concatenatedString, $secretKey2);

// Сравнение сгенерированного хеша с полученной сигнатурой
if(hash_equals($generatedSignature, $_SERVER['HTTP_X_SIGNATURE'])){
    echo "Сигнатура верна!"
}
?>

Пример скрипта на JavaScript (Node.js)
const crypto = require('crypto');

// Секретный ключ #2
const secretKey2 = '';

// Express.js пример
app.post('/webhook', express.json(), (req, res) => {
const params = req.body;
const receivedSignature = req.headers['x-signature'];

// Сортировка параметров по ключам в алфавитном порядке
const sortedKeys = Object.keys(params).sort();

// Объединение всех значений в одну строку
const concatenatedString = sortedKeys.map(key => params[key]).join('');

// Генерация хеша с использованием HMAC-SHA256 и секретного ключа
const generatedSignature = crypto
.createHmac('sha256', secretKey2)
.update(concatenatedString)
.digest('hex');

// Сравнение сгенерированного хеша с полученной сигнатурой
if (crypto.timingSafeEqual(
Buffer.from(generatedSignature),
Buffer.from(receivedSignature)
)) {
res.status(200).json({ message: 'Сигнатура верна!' });
} else {
res.status(401).json({ error: 'Неверная сигнатура' });
}
});
Пример скрипта на Python
import hmac
import hashlib
import json
from flask import Flask, request, jsonify

app = Flask(**name**)

# Секретный ключ #2

SECRET_KEY_2 = ''

@app.route('/webhook', methods=['POST'])
def webhook(): # Получение параметров из тела запроса
params = request.get_json()

    # Получение сигнатуры из заголовков
    received_signature = request.headers.get('X-SIGNATURE')

    # Сортировка параметров по ключам в алфавитном порядке
    sorted_params = dict(sorted(params.items()))

    # Объединение всех значений в одну строку
    concatenated_string = ''.join(str(v) for v in sorted_params.values())

    # Генерация хеша с использованием HMAC-SHA256 и секретного ключа
    generated_signature = hmac.new(
        SECRET_KEY_2.encode('utf-8'),
        concatenated_string.encode('utf-8'),
        hashlib.sha256
    ).hexdigest()

    # Сравнение сгенерированного хеша с полученной сигнатурой
    if hmac.compare_digest(generated_signature, received_signature):
        return jsonify({'message': 'Сигнатура верна!'}), 200
    else:
        return jsonify({'error': 'Неверная сигнатура'}), 401

if **name** == '**main**':
app.run(debug=True)
Обработка webhook
Ниже — общие правила для webhook по инвойсам и по выплатам: ответ вашего сервера, повторные попытки доставки и рекомендации по реализации.

✅ Успешная обработка
HTTP Status: 200 OK
Webhook считается успешно обработанным, если ваш сервер возвращает HTTP-код 200.

🔄 Повторная отправка
Если ваш сервер возвращает любой код ответа, отличный от 200, webhook будет автоматически переотправляться:

Параметр Значение
Максимальное количество попыток 5 раз
Условие остановки Получение HTTP-кода 200
Интервал между попытками Автоматически определяется системой
Пример сценария:

Попытка 1: HTTP 500 → 🔄 Переотправка
Попытка 2: HTTP 500 → 🔄 Переотправка
Попытка 3: HTTP 200 → ✅ Успешно, переотправка прекращена
⚠️ Рекомендации
Важно: Рекомендуется обрабатывать webhook идемпотентно — один и тот же webhook может быть доставлен несколько раз. Убедитесь, что ваша логика обработки учитывает это и не создаёт дубликаты операций.

Практические советы:

Используйте уникальный идентификатор операции (id или order_id) для проверки, был ли webhook уже обработан
Сохраняйте статус обработки webhook в базе данных
Перед выполнением бизнес-действий сверяйте актуальный статус операции (инвойса или выплаты) с ожидаемым
