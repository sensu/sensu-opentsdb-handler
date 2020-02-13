# sensu-opentsdb-handler

## Table of Contents
- [Overview](#overview)
- [Usage Examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Resource definition](#resource-definition)
- [Installation from source](#installation-from-source)
- [Contributing](#contributing)

## Overview

The Sensu OpenTSDB Handler is a [Sensu Event Handler][9] that sends metrics to
an [OpenTSDB][10] server via [its telnet API][12].

[Sensu][11] can collect metrics using check output metric extraction or the
`statsd` listener. Those collected metrics pass through an event pipeline,
allowing Sensu to deliver normalized metrics to the configured metric event
handlers.

This OpenTSDB handler enables extracting, tagging and storing that metric data
into an OpenTSDB database.

## Usage Examples

Help:
```
an opentsdb handler built for use with sensu

Usage:
  sensu-opentsdb-handler [flags]
  sensu-opentsdb-handler [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -h, --help                       help for sensu-opentsdb-handler
      --host string                OpenTSDB host to send metrics to
      --port string                OpenTSDB port to send metrics to (default "4242")
      --prefix string              Prefix metrics name with this string
      --prefix-entity-name         Prefix metrics name with the entity name
      --retries uint               Number of times to try to connect to the server (default 3)
      --retry-delay uint           Delay in seconds between connection attempts (default 1)
      --space-replacement string   String to replace spaces with if the entity name or tags contain any (default "-")
      --tag-host                   Add a host tag holding the entity name to metrics (default true)
      --tags stringToString        Add these tags to metrics (default [])
```

## Configuration

### Asset registration

Assets are the easiest way to make use of this handler. If you're using
`sensuctl` 5.13 with `sensu-backend` 5.13 or later, you can use the following
command to add this handler as an asset:

```
sensuctl asset add sensu/sensu-opentsdb-handler
```

If you're using an earlier version of Sensu, you can find the asset on the
[Bonsai Asset Index][13].

### Resource definition

```yml
---
type: Handler
api_version: core/v2
metadata:
  name: send-metrics-to-opentsdb
  namespace: default
spec:
  command: sensu-opentsdb-handler --host localhost --prefix sensu --tags type=system,source=sensu
  runtime_assets:
  - sensu-opentsdb-handler
  type: pipe
```

## Installation from source

The preferred way of installing and deploying this handler is to use it as an asset. If you would
like to compile and install the plugin from source or contribute to it, download the latest version
or create an executable script from this source.

From the local path of the `sensu-opentsdb-handler` repository:

```
go build
```

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md
[9]: https://docs.sensu.io/sensu-go/latest/reference/handlers/#how-do-sensu-handlers-work
[10]: http://opentsdb.net
[11]: https://github.com/sensu/sensu-go
[12]: http://opentsdb.net/docs/build/html/api_telnet/index.html
[13]: https://bonsai.sensu.io/assets/sensu/sensu-opentsdb-handler
