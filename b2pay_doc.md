Запросы аутентификации
POST /v1/auth/token/get
JSON
Используйте этот запрос, чтобы получить JWT токен по вашему User ID, Email и API ключу.

Максимальное значение token_expiry_hours: 720 (30 дней).

Эндпоинт POST /v1/auth/token/get ограничен по частоте запросов с одного IP. Превышение лимита может привести к временной блокировке доступа.

Используйте полученный JWT токен для всех последующих запросов к API (счета, платежи, статус, список транзакций). Не запрашивайте новый токен при каждом действии.

Запрос
curl -X POST "https://app.b2pay.online/v1/auth/token/get" \
 -H "Content-Type: application/json" \
 -d '{
"user_id": "playgate_store_hdbn",
"email": "daniilpietrov00@gmail.com",
"api_key": "sk_iICXNCfJMaCmNXn3jv1flqG1C6V6QTn-",
"token_expiry_hours": 24
}'
Ответ
{
"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
POST /v1/auth/token/refresh
JSON
Используйте этот запрос, чтобы обновить действующий JWT токен до истечения срока.

Запрос
curl -X POST "https://app.b2pay.online/v1/auth/token/refresh" \
 -H "Content-Type: application/json" \
 -d '{
"token": "<YOUR_JWT_TOKEN>",
"token_expiry_hours": 24
}'
Ответ
{
"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}

Создание счета (платежная форма)
POST /v1/invoices
JSON
Используйте этот endpoint для создания счета на оплату через нашу платежную форму. После создания счета перенаправьте клиента на auth_url из ответа.

Используйте JWT токен из блока аутентификации.

Запрос
curl -X POST "https://app.b2pay.online/v1/invoices" \
 -H "Content-Type: application/json" \
 -H "Authorization: Bearer <YOUR_JWT_TOKEN>" \
 -d '{
"customer_id": "customer-123",
"amount": 100.00,
"currency": "USD",
"description": "Test payment",
"is_returning_customer": true,
"metadata": {
"test_mode": true,
"customer_email": "customer@example.com",
"tracking_id": "order-123",
"return_url": "https://merchant.example.com/return",
"notification_url": "https://merchant.example.com/callback"
}
}'
Ответ
{
"id": "b1b9b6b4-8c5d-4f1d-9a23-0e5f8e4b9a10",
"customer_id": "customer-123",
"amount": 100.0,
"currency": "USD",
"status": "pending",
"description": "Test payment",
"metadata": {
"requires_action": true,
"auth_method": "redirect",
"auth_url": "https://app.b2pay.online/payment/b1b9b6b4-8c5d-4f1d-9a23-0e5f8e4b9a10",
"test_mode": true,
"tracking_id": "order-123"
},
"created_at": "2024-04-02T12:47:26.818Z"
}
Параметры тела запроса (типы)

Параметр Тип необязательный / обязательный
customer_id string обязательный
amount number обязательный
currency string обязательный
description string обязательный
is_returning_customer boolean необязательный
metadata object обязательный
metadata.test_mode boolean обязательный
metadata.customer_email string необязательный
metadata.tracking_id string обязательный
metadata.return_url string необязательный
metadata.notification_url string обязательный
is_returning_customer (необязательный):

Необязательный параметр (boolean). Укажите true, если клиент ранее производил успешные платежи на вашем сайте. Если гейт настроен на приём только вторичного трафика, а этот параметр равен false или не указан, оплата через такой гейт будет невозможна.

i
Процесс оплаты через платежную форму
Этот endpoint создает счет, который будет оплачен через нашу безопасную платежную форму.

Клиент будет перенаправлен на платежную форму, где сможет выбрать способ оплаты и ввести данные для оплаты.

Если customer_email не указан в metadata, клиенту будет предложено ввести его в платежной форме.

После завершения оплаты клиент может вернуться на ваш сайт, используя кнопку return_url.

i
Статусы платежа
pending — платёж создан и находится в обработке или требует дополнительных действий (3‑D Secure и т.п.)
success — платёж успешно обработан, средства зарезервированы/зачислены и учитываются на балансах с учётом настроек T+ и роллинга
failed — платёж был обработан, но отклонён (ошибка на стороне банка или валидации)
refunded — платёж был успешно возвращён (средства вернулись клиенту)
refund_cancelled — возврат был отменён (сумма возврата и комиссия возвращены на баланс мерчанта)
chargeback — платёж был сторнирован банком‑эмитентом (чарджбэк)
fraud — платёж был заблокирован внутренней системой антифрода (списания и зачисления средств не происходят).
expired — финальный статус не получен в течение времени опроса; транзакция закрыта без операций по балансу. На notification_url отправляется callback со статусом expired.
!
Callback (notification_url)
При создании платежа вы передаёте metadata.notification_url. После изменения статуса транзакции (успешно, ошибка, возврат, чарджбэк, фрод) наша система отправляет POST запрос на этот URL.

Тело callback‑запроса — JSON в том же формате, что и ответ метода /v1/invoices, но с актуальным статусом и metadata (включая requires_action, auth_method, auth_url, test_mode).

Для тестовых платежей (test_mode = true) callback также отправляется, но балансы мерчанта не изменяются.

Для способа оплаты card_acquiring при известном номере карты в метаданных в callback в metadata передаётся поле card_mask — маска карты (первые 6 и последние 4 цифры, остальные скрыты звёздочками, количество \* равно количеству скрытых цифр). Полный номер карты в коллбэке не передаётся.

Ожидаемый ответ. Ваш endpoint должен отвечать HTTP 200 OK. Любой другой код ответа (4xx, 5xx) или ошибка соединения (таймаут, отказ) считаются неудачей; мы будем повторно отправлять коллбэк.

Проверка подписи коллбэка: Мы отправляем заголовок X-Callback-Signature: HMAC-SHA256 от сырого JSON-тела запроса, подписанный вашим API-ключом, в формате sha256=<hex>. Для проверки считайте HMAC-SHA256(тело_запроса, api_key), закодируйте в hex и сравните со значением заголовка (после префикса "sha256="). Совпадение означает, что коллбэк отправлен нами.

Примеры генерации/проверки подписи (PHP, Python, Go)
Пример callback для тестовой транзакции (status: success):

{
"id": "b1b9b6b4-8c5d-4f1d-9a23-0e5f8e4b9a10",
"customer_id": "customer-123",
"amount": 100.0,
"currency": "USD",
"status": "success",
"payment_method": "card_acquiring",
"description": "Test payment",
"provider_name": "Test Gateway",
"created_at": "2024-04-02T12:47:26.818000000+00:00",
"metadata": {
"test_mode": true,
"tracking_id": "order-123",
"requires_action": false,
"card_mask": "424242**\*\***4242"
}
}
T
Тестовый гейт
Если в metadata.test_mode = true, платёж обрабатывается через внутренний тестовый гейт. В этом режиме балансы мерчанта никогда не изменяются, независимо от статуса транзакции.

Тестовый гейт работает только при payment_method = "card_acquiring".

Для эмуляции разных сценариев в тестовом гейте используйте следующие тестовые номера карт (card_number):

4111111111111111 — платёж проходит со статусом success.
4000000000000002 — платёж завершается со статусом failed.
4000000000000101 — платёж создаётся со статусом pending.

Получение списка доступных гейтов
GET /v1/gateways
JSON
Используйте этот endpoint, чтобы получить список ваших платежных гейтов и их текущие параметры.

Используйте JWT токен из блока аутентификации.

Параметры запроса
Все параметры необязательны.

limit (необязательный) — количество записей на странице (1-100), по умолчанию 20
offset (необязательный) — смещение для пагинации, по умолчанию 0
gateway_type (необязательный) — фильтр по типу гейта (код gateway_type)
Запрос
curl -X GET "https://app.b2pay.online/v1/gateways?limit=20&offset=0&gateway_type=card_acquiring" \
 -H "Authorization: Bearer <YOUR_JWT_TOKEN>"
Ответ
{
"gateways": [
{
"id": "c8b5c965-f7b9-46b7-99c1-6c5f65a41df7",
"external_name": "My Test Gateway",
"gateway_type_code": "card_acquiring",
"gateway_type_name": "Card Acquiring",
"gateway_type_payment_methods": "Mastercard, Visa",
"gateway_type_active": true,
"currency_code": "USD",
"available_countries": ["US", "GB"],
"supported_card_types": ["visa", "mastercard"],
"min_amount": 10,
"max_amount": 5000,
"commission_percent_deduction": 2.5,
"fixed_commission_success": 0.3,
"fixed_commission_error": 0.0,
"rolling_percent": 10,
"rolling_period": 7,
"tp_period": 3,
"refund_commission": 15,
"chargeback_commission": 60,
"global_status": true,
"local_status": true,
"traffic_type": "both",
"admin_disabled": false,
"h2h_enabled": true,
"generate_billing_data": false
}
],
"total": 1,
"limit": 20,
"offset": 0
}

Получение статуса транзакции
GET /v1/transactions/{id}/status
JSON
Используйте этот endpoint для получения актуального статуса транзакции.

Используйте JWT токен из блока аутентификации.

Ограничение: 10 запросов в минуту на пользователя

Запрос
curl -X GET "https://app.b2pay.online/v1/transactions/b1b9b6b4-8c5d-4f1d-9a23-0e5f8e4b9a10/status" \
 -H "Authorization: Bearer <YOUR_JWT_TOKEN>"
Ответ
{
"id": "b1b9b6b4-8c5d-4f1d-9a23-0e5f8e4b9a10",
"customer_id": "customer-123",
"amount": 100.0,
"currency": "USD",
"status": "success",
"payment_method": "card_acquiring",
"description": "Test payment",
"provider_name": "My Test Gateway",
"metadata": {
"requires_action": false,
"test_mode": true,
"tracking_id": "order-123"
},
"created_at": "2024-04-02T12:47:26.818Z"
}

Переотправка коллбэка транзакции
POST /v1/transactions/{id}/resend-callback
JSON
Используйте этот endpoint для ручной переотправки callback на notification_url транзакции (например, если ваш сервер не ответил вовремя). Тело запроса не требуется. Коллбэк отправляется с заголовком X-Callback-Signature так же, как при автоматической отправке.

Используйте JWT токен из блока аутентификации.

Ограничение: не более 10 ручных переотправок в час на одну транзакцию.

Запрос
curl -X POST "https://app.b2pay.online/v1/transactions/b1b9b6b4-8c5d-4f1d-9a23-0e5f8e4b9a10/resend-callback" \
 -H "Authorization: Bearer <YOUR_JWT_TOKEN>"
Ответ
{
"success": true,
"message": "Callback sent successfully",
"callback_attempts": 2,
"callback_sent": true
}

Получение списка транзакций
GET /v1/transactions
JSON
Возвращает постраничный список транзакций. Поддерживаются фильтры по периоду, статусу, способу оплаты, валюте и поиску.

Используйте JWT токен из блока аутентификации.

Параметры запроса
Все параметры необязательны. Без from_date/to_date по умолчанию используются последние 3 дня.

Пагинация задаётся только параметром page (страница с 1).

limit (необязательный) — 1–100, по умолчанию 10
page (необязательный) — номер страницы (с 1), по умолчанию 1
from_date, to_date — YYYY-MM-DD (по умолчанию: последние 3 дня)
status — pending, success, failed, refunded, refund_cancelled, chargeback, fraud
payment_method, currency, search
id — UUID одной транзакции (возвращается одна транзакция)
Запрос
curl -X GET "https://app.b2pay.online/v1/transactions?limit=10&page=1&from_date=2024-04-01&to_date=2024-04-10&status=success" \
 -H "Authorization: Bearer <YOUR_JWT_TOKEN>"
Ответ
{
"transactions": [
{
"id": "b1b9b6b4-8c5d-4f1d-9a23-0e5f8e4b9a10",
"merchant_user_id": "example",
"customer_id": "customer-123",
"amount": 100.0,
"currency": "USD",
"payment_method": "card_acquiring",
"status": "success",
"description": "Test payment",
"metadata": {
"tracking_id": "order-123",
"test_mode": true,
"gateway_external_name": "My Test Gateway"
},
"callback_url": "https://merchant.example.com/callback",
"callback_sent": true,
"callback_attempts": 1,
"created_at": "2024-04-02T12:47:26.818Z",
"updated_at": "2024-04-02T12:48:00.000Z"
}
],
"total": 1,
"limit": 10,
"offset": 0
}
