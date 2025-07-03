```json
响应状态: 200 OK
响应内容: {"resMsg":"Y21/VyeVlXXPumJW2NWv6h3ZG93DTAd3NC+H3YbDYoTZ0hIAl+6U35kGf9dH0+BVz0kkf+NNPn8eWdkf5olB2uEpBXwj82EZyElNAtxMy+g=","resKey":"C1FjcyhxYsK8e5PhtZpfPV/OdZe5GGmSJ1BcAtxp6WMcQMYB44sY5i3jcY6CGyRVbyinvLrldhScIyC06tUPiARexGHrHHT0AppJWAEjy6gXtyRbz44Gk0dMSBL3fmtzpNmYX6sFsAouhrPi7zN5KSoL/fco0eBDCObcmKal/9I="}
```

> 为了保证网络传输的安全，请求需要加密，响应需要解密。推送的消息无需加解密。

```json
{
  "sendTime": 1719564492000,
  "wsMessageType": "BOND_QUOTE",
  "data": {
    "messageId": "MSG2025070300001",
    "messageType": "BOND_QUOTE_UPDATE",
    "timestamp": 1719564492000,
    "securityCode": "019603.IB",
    "data": {
      "askPrices": [
        {
          "brokerId": "BROKER001",
          "isTbd": "N",
          "isValid": "Y",
          "minTransQuantity": 1000000,
          "orderQty": 5000000,
          "price": 100.25,
          "quoteOrderNo": "ASK2025070300001",
          "quoteTime": "2025-07-03T14:54:52.000Z",
          "securityCode": "019603.IB",
          "settleType": "T+1",
          "settleDate": "2025-07-04",
          "side": "ASK",
          "yield": 3.125
        }
      ],
      "bidPrices": [
        {
          "brokerId": "BROKER002",
          "isTbd": "N",
          "isValid": "Y",
          "minTransQuantity": 1000000,
          "orderQty": 10000000,
          "price": 100.15,
          "quoteOrderNo": "BID2025070300001",
          "quoteTime": "2025-07-03T14:54:30.000Z",
          "securityCode": "019603.IB",
          "settleType": "T+1",
          "settleDate": "2025-07-04",
          "side": "BID",
          "yield": 3.135
        }
      ]
    }
  }
}
```