# Receptor Plugins

Plugins are build as separate processes connected via RPC and event channels.

## Build your own Plugin
Every go application could be a plugin if it meets the requirements.
To start simple create a fresh package and follow the instructions below.

A plugin need to fit in one of the categories `Watcher` or `Reactor`.
A `Watcher` is the Source of Events, the `Reactor` the Sink.
Because of their different behaviour they have different requirements and need
different setup as a plugin, the workflow is quite similar though.

### Requirements and Lifecycle
There are 2 requirements every plugin needs to fulfill:
- Implement the [Watcher Interface](../pipe/watcher.go) or [Reactor Interface](../pipe/reactor.go)
- Serve the Plugin using plugin interface

Both steps are described below in detail, but first some basics:

The **life cycle** of every plugin looks the same:
- (optional) The plugin is provided a global configuration.
- For each `Service` using the plugin:
  - Receptor asks to `Accept` a service-specific configuration.
  - The Plugins returns an `Endpoint` which could be inserted into the Event pipeline.
  - Receptor executes the `Endpoint` within the service pipeline.
  - (on shutdown) The `Endpoint` gets closed

The interface you need to implement is the same for a `Watcher` and `Reactor`:
```go
type Watcher/Reactor interface {
  Setup(cfg json.RawMessage) error
  Accept(cfg json.RawMessage) (Endpoint, error)
}
```
It's a very small interface giving you all the power you need.

The main difference between a `Watcher` and a `Reactor` is the behaviour of the `Endpoint` described later.

#### Starting with the `Setup` method
This method is used to setup the plugin in **general**, this means it configures
the whole plugin not only for one service.
In this step you should setup some global state and verify the given configuration.

If the configuration is not correct, return a descriptive error explaining what is wrong.
This cancels the global startup and receptor will shutdown. A `nil` error signals the
configuration was accepted.


#### Accepting a Service with the `Accept` method
The `Accept` method is used to setup one instance of your plugin dedicated to a service.
As above the plugin can deny the configuration provided using a descriptive error.
If the plugin accepts the configuration it returns an `Endpoint` configured for the service.

Because there is no guarantee your `Endpoint` is called, only transfer state to the endpoint,
don't start any background process.

#### The `Endpoint` in general
The Endpoint is an instance of your plugin configured for a service, ready to run.
It needs to implement the following interface.

```go
type Endpoint interface {
  Handle(eventCh chan Event, closeCh chan struct{})
}
```

For most plugins there's a simpler way to create an `Endpoint` using a closure
and returning an `EndpointFunc`:

```go
type EndpointFunc func(eventCh chan Event, closeCh chan struct{})
```


As described above, the endpoint needs to be fully configured and ready to run,
so transfer every needed state to the created instance.

The Endpoint provides you with an event-channel and a close-channel needed for communicating
inside the event pipeline.


#### Watcher `Endpoint`
Please read the [Endpoint in general](#the-endpoint-in-general) first.

Your watcher needs to send events down the pipeline, this is simply done by creating a event and send it over the channel:

```go
eventCh <- pipe.NewEventWithNode("Nodename", pipe.NodeUp, "127.0.0.1", 80)
```

You'r almost done with the work now, have a look at [Events](#events) for some examples how to create events.

There's only one requirement left, the shutdown:

Your Endpoint shuts down by returning inside the Handle method.

**When to shut down?**
- eventCh gets closed
- closeCh gets closed

On both signals, simply cleanup, return and you're done!

#### Reactor `Endpoint`
Please read the [Endpoint in general](#the-endpoint-in-general) first.

A Reactor receives events from the pipeline:

```go
event, ok := <- eventCh
```
It's that simple. Have a look at [Events](#events) for examples how to work with events.

Like the watcher, you need to shut down your endpoint properly on those signals:
- eventCh gets closed
- closeCh gets closed

On both signals, simply cleanup, return and you're done!

### Finally: Serve the plugin

To serve your plugin using it's own process your main method needs to look like this:
```go
package main

import (
  "github.com/blang/receptor/plugin"
  "github.com/blang/receptor/plugins/watcher/dummy/dummy"
)

func main() {
  plugin.ServeWatcher(&dummy.DummyWatcher{})
}
```

To serve a `Reactor` its analogous:

```go
plugin.ServeReactor(&filelog.FileLogReactor{})
```

## Events

Simply create events:
```go
ev:=pipe.NewEventWithNode("Nodename", pipe.NodeUp, "127.0.0.1", 80)
ev:=pipe.NewEvent()
ev.AddNewNode("Nodename", pipe.NodeUp, "127.0.0.1", 80)
ev.AddNode(pipe.NewNodeInfo("Nodename", pipe.NodeUp, "127.0.0.1", 80))
```

Get information from events:

An `Event` is a map of NodeName (string) to NodeInfo, which holds NodeStatus, Host etc.
```go
type Event map[string]NodeInfo

type NodeInfo struct {
  Name   string
  Status NodeStatus
  Host   string
  Port   uint16
}
```

For more information see [events.go](../pipe/events.go).
