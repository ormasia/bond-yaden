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
askPrices：买价
bidPrices：卖价


BondQuoteDetail全部记录
BondLatestQuote直接记录json

导出格式
```go
headers := []string{
		"债券代码",
		"买方价格", "买方收益率", "买方数量", "买方报价时间",
		"卖方价格", "卖方收益率", "卖方数量", "卖方报价时间",
		"消息ID", "消息类型", "发送时间", "时间戳", "更新时间",
		"买方券商ID", "卖方券商ID",
	}
```
是对峙形式，买卖数量如果不一致，另一方输出空白；

每日分表，详情表和最新表都要按日期创建表
询问后：写入前判断，然后建表；或者每天凌晨自动建表服务，兜底重试一下？（有写入权限，就有建表权限）

还差 nacos接入，我先把配置的yaml写出来，后面可以直接放到nacos

需要生产环境账号，测试一下；

是不是可以考虑池化操作，解析的时候复用一下对象？