# receptor-reactor-filelogger

filelogger writes incoming events to a log file. Used for testing purpose.

## Config

### Global
No global configuration

### Service
```json
{
  "reactors": {
    "filelogreactor1": {
      "type": "filelogger",
      "cfg": {
        "filename": "/tmp/receptor_events.log",
        "unbuffered": true
      }
    }
  }
}
```

If Unbuffered is true, output is written after every event.