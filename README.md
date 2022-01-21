### :warning: Deprecated :warning:
This plugin is no longer actively maintained. This repo will be archived, and the cf-drain-cli entry in the community plugins repo removed, after June 2022.

CF Drain CLI Plugin
[![Concourse Badge][ci-badge]][ci-tests]
====================

The CF Drain CLI Plugin is a [CF CLI][cf-cli] plugin to simplify interactions
with user provided syslog drains.

### Installing Plugin

#### From CF-Community

```
cf install-plugin -r CF-Community "drains"
```

#### From Binary Release

1. Download the binary for the [latest release][latest-release] for your
   platform.
1. Install it into the cf cli:

```
cf install-plugin download/path/cf-drain-cli
```

#### From Source Code

Make sure to have the [latest Go toolchain][golang-dl] installed.

```
go get code.cloudfoundry.org/cf-drain-cli/cmd/cf-drain-cli
cf install-plugin $GOPATH/bin/cf-drain-cli
```

### Quick Start

#### Create an app Drain
```
cf drain my-app syslog://my-drain.com --drain-name my-drain
```

#### List all drains in a space
```
cf drains 
```

#### Delete a drain
```
cf delete-drain my-drain
```

#### Drain all apps in a space
```
cf drain-space syslog://my-drain.com --drain-name my-space-drain
```

#### Delete Space Drain
```
cf delete-drain-space my-space-drain
```

### Usage
#### Create Drain
```
$ cf drain --help
NAME:
   drain - Creates a user provided service for syslog drains and binds it to a given application.

USAGE:
   drain <app-name> <syslog-drain-url> [options]

OPTIONS:
   --drain-name         The name of the drain that will be created. If excluded, the drain name will be `cf-drain-UUID`.
   --type               The type of logs to be sent to the syslog drain. Available types: `logs`, `metrics`, and `all`. Default is `logs`
```

#### Delete Drain
```
$ cf delete-drain --help
NAME:
   delete-drain - Unbinds the service from applications and deletes the
service.

USAGE:
   delete-drain <drain-name>
```

#### Bind Drain
```
$ cf bind-drain --help
NAME:
   bind-drain - Binds an application to an existing syslog drain.

USAGE:
   bind-drain <app-name> <drain-name>
```

#### List Drains
```
$ cf drains --help
NAME:
   drains - Lists all services for syslog drains.

USAGE:
   drains
```

#### Space Drain

```
cf drain-space --help
NAME:
   drain-space - Pushes app to bind all apps in the space to the configured syslog drain.

USAGE:
   drain-space SYSLOG_DRAIN_URL [--drain-name NAME] [--path PATH] [--type TYPE]

OPTIONS:
   --drain-name       Name for the space drain.
   --path             Path to the space drain app to push. If omitted the latest release will be downloaded.
   --type             Which log type to filter on (logs, metrics, all). Default is all.
```

#### Delete Space Drain

```
$ cf delete-drain-space --help
NAME:
   delete-drain-space - Deletes space drain app and unbinds all the apps in the space from the configured syslog drain.

USAGE:
   delete-drain-space DRAIN_NAME [--force]

OPTIONS:
   --force       Skip warning prompt. Default is false.
```

[cf-cli]: https://code.cloudfoundry.org/cli
[ci-badge]: https://loggregator.ci.cf-app.com/api/v1/pipelines/products/jobs/cf-drain-cli-tests/badge
[ci-tests]: https://loggregator.ci.cf-app.com/teams/main/pipelines/products/jobs/cf-drain-cli-tests
[golang-dl]: https://golang.org/dl/
[latest-release]: https://github.com/cloudfoundry/cf-drain-cli/releases/latest
