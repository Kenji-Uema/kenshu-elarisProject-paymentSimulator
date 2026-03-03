curl -i -X POST http://localhost:8080/v1/payments/card:pay \
    -H "Content-Type: application/json" \
    -d '{
      "card": {
        "number": "4242424242424242",
        "expMonth": 12,
        "expYear": 2030,
        "cvv": "123",
        "holderName": "Test User"
      }
    }'

