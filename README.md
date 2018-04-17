CF Drain CLI Plugin
[![Concourse Badge][ci-badge]][ci-tests]
====================

The CF Drain CLI Plugin is a [CF CLI][cf-cli] plugin to simplify interactions
with user provided syslog drains.

### Installing Plugin

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
   drain [options] <app-name> <syslog-drain-url>

OPTIONS:
   -password           The password to use for authentication when the `adapter-type` is `application`. Required if `adapter-type` is `application`.
   -type               The type of logs to be sent to the syslog drain. Available types: `logs`, `metrics`, and `all`. Default is `logs`
   -username           The username to use for authentication when the `adapter-type` is `application`. Required if `adapter-type` is `application`.
   -adapter-type       Set the type of adapter. The adapter is responsible for forwarding messages to the syslog drain. Available options: `service` or `application`. Service will use a cf user provided service that reads from loggregator and forwards to the drain. Application will deploy a cf application that reads from log-cache and forwards to the drain. Default is `service`
   -drain-name         The name of the app that will be created to forward messages to your drain. Default is `cf-drain-UUID`
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
The space drain functionality is an experimental feature. In large
deployments, it can create additional load because it binds every app in the
space to a drain. Be sure to consider your deployment size when deciding
whether to use a full space drain.

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
   --skip-ssl-validation      Whether to ignore certificate errors. Default is false
   --type                     Which log type to filter on (logs, metrics, all). Default is all
   --username                 Username to use when pushing the app. If not specified, a user will be created (requires admin permissions)
```

[cf-cli]: https://code.cloudfoundry.org/cli
[ci-badge]: https://loggregator.ci.cf-app.com/api/v1/pipelines/products/jobs/cf-drain-cli-tests/badge
[ci-tests]: https://loggregator.ci.cf-app.com/teams/main/pipelines/products/jobs/cf-drain-cli-tests
