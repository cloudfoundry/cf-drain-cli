CF Drain CLI Plugin
====================

The CF Drain CLI Plugin is a [CF CLI](cf-cli) plugin to simply for creating
and binding to syslog user provided services

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

[cf-cli]: https://code.cloudfoundry.org/cli
