# receptor-watcher-dummy

dummy fires alternating nodeup and down events every 2 seconds. Used for testing purpose.

## Config

### Global
No global configuration

### Service
```json
{
  "watchers": {
    "dummywatcher1": {
      "type": "dummy",
      "cfg": {
        "name": "TestNode1",
        "host": "127.0.0.1",
        "port": 80
      }
    }
  }
}
```

