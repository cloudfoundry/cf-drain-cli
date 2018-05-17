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
go get code.cloudfoundry.org/cf-drain-cli
cf install-plugin $GOPATH/bin/cf-drain-cli
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

#### Space Drain (Experimental)

**Note:**
The space drain functionality is an experimental feature. If the adapter-type
`service` is selected, it can create additional load in large deployments
because it binds every app in the space to a drain. Be sure to consider your
deployment size when deciding whether to use a full space drain.

```
NAME:
   drain-space - Pushes app to bind all apps in the space to the configured syslog drain

USAGE:
   drain-space [OPTIONS]

OPTIONS:
   --drain-name               Name for the space drain. Required
   --drain-url                Syslog endpoint for the space drain. Required
   --force                    Skip warning prompt. Default is false
   --path                     Path to the space drain app to push. If omitted the latest release will be downloaded
   --type                     Which log type to filter on (logs, metrics, all). Default is all
   --username                 Username to use when pushing the app. If not specified, a user will be created (requires admin permissions)
   --adapter-type             Set the type of adapter. The adapter is responsible for forwarding messages from all apps in the space to the syslog drain. Available options: `service` or `application`. Service will bind a cf user provided service to each app that reads from loggregator and forwards to the drain. Application will deploy a cf application that reads from log-cache and forwards to the drain. Default is `service`
```

[cf-cli]: https://code.cloudfoundry.org/cli
[ci-badge]: https://loggregator.ci.cf-app.com/api/v1/pipelines/products/jobs/cf-drain-cli-tests/badge
[ci-tests]: https://loggregator.ci.cf-app.com/teams/main/pipelines/products/jobs/cf-drain-cli-tests
[golang-dl]: https://golang.org/dl/
[latest-release]: https://github.com/cloudfoundry/cf-drain-cli/releases/latest
