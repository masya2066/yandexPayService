# Yandex Pay Service to receive P2P payments from Yandex Money
That's service to receive P2P payments from Yandex Money to use it from other application if use want to use microservice architecture.
To get started you just need to create Yandex money wallet and get number address of your wallet

## Download and run the application
* `git clone git@github.com:masya2066/yandexPayService.git`
* `cd yandexPayService`
* `go mod tidy`
* `go build -o .  app/cmd/umani-service/main.go && ./main`

After these command service is running!
To work this service correctly you need to create or set file `config.json` in the root directory of the project.

## Set up yandex wallet
* Visit [Yandex Money](https://yoomoney.ru/transfer/myservices/http-notification)
* Set up site URL to send notifications like `https://109.51.122.99/order/yandex` (But yandex must have access to this URL from the Internet)
* Turn ON `Send URL-Notifications`

* If you have more questions, you can read this documentation [here](https://yoomoney.ru/docs/wallet/using-api/notification-p2p-incoming)

## Restart service after changes
* `go build -o .  app/cmd/umani-service/main.go && ./main`

## Endpoint to use
* Create order: `curl --location 'http://localhost:8080/yandex/order/create' \
--header 'Content-Type: application/json' \
--data-raw '{
    "email": "test@example.com",
    "amount": "100.00",
    "description": "Order12345"
}'`
