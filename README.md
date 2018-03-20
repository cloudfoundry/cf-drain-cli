CF Drain CLI Plugin
====================

The CF Drain CLI Plugin is a [CF CLI](cf-cli) plugin to simplify interactions
with user provided syslog drains.

### Installing Plugin

```
go get code.cloudfoundry.org/cf-drain-cli
cf install-plugin $GOPATH/bin/cf-drain-cli
```

### Usage

#### Create Drain
```
$ cf create-drain --help
NAME:
   create-drain - Creates a user provided service for syslog drains and binds
it to a given application.

USAGE:
   create-drain [options] <app-name> <drain-name> <syslog-drain-url>

OPTIONS:
   -type       The type of logs to be sent to the syslog drain. Available types: `logs`, `metrics`, and `all`. Default is `logs`
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
   push-space-drain - Pushes app to bind all apps in the space to the configured syslog drain

USAGE:
   push-space-drain [OPTIONS]

OPTIONS:
   -path                      Path to the space drain app to push
   -skip-ssl-validation       Whether to ignore certificate errors. Default is false
   -type                      Which log type to filter on (logs, metrics, all). Default is all
   -username                  Username to use when pushing the app
   -drain-name                Name for the space drain
   -drain-url                 Syslog endpoint for the space drain
   -password                  Password to use when pushing the app
   -force                     Skip warning prompt. Default is false
```

[cf-cli]: https://code.cloudfoundry.org/cli
