# receptor-watcher-restapiserver

restapiserver provides a http restapi to fire events.

## Config

### Global
```json
{
  "watchers": {
    "restapiserver": {
      "listen": "127.0.0.1:8001"
    }
  }
}
```

### Service
```json
{
  "watchers": {
    "mywatcher": {
      "type": "restapiserver",
      "cfg": {
        "service": "service1"
      }
    }
  }
}
```

## Usage

Using the above example configuration, following rest endpoint is available:

POST http://127.0.0.1:8001/service/service1
Body:
```json
{
  "name":"Testnode",
  "type":"nodedown",
  "host":"126.0.0.2",
  "port": 80
}
```
Response: 200 - "OK"

Request Body Types:
- "nodedown"
- "nodeup"

If the request is not acceptable: Responsecode 400 - Error msg
