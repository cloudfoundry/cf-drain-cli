CF Syslog CLI Plugin
====================

The CF Syslog CLI Plugin is a [CF CLI](cf-cli) plugin to simply for creating
and binding to syslog user provided services

## Usage

### Create Drain
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

### Delete Drain
```
$ cf delete-drain --help
NAME:
   delete-drain - Unbinds the service from applications and deletes the
service.

USAGE:
   delete-drain <drain-name>
```

[cf-cli]: https://code.cloudfoundry.org/cli
