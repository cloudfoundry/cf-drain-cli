# Space Drain
The space drain creates bindings for all apps in the space its deployed to.
The app refreshes bindings every minute, so that new apps are bound to the
syslog drain.

## Deploying
While the CF Drain CLI is the preferred deployment strategy, this app can be
deployed with out it.

Build the app with

```
go build
```

Once the app is built, deploy it with the following command

```
cf push <name> -b binary_buildpack -c ./space_drain -u proccess --no-start
```

Set the envrioronment variables in [Configuration](#config)

run

```
cf start <name>
```

## <a="config"></a> Configuration
Set the following environment variables on the app with the command

```
cf set-env <name> VARIABLE <VALUE>
```

SPACE_ID - The ID (rather than the name) of the space the drain is deployed to
DRAIN_NAME - The space drain app name. This is used so the drain ignores itself
DRAIN_URL - Where to drain the apps. https, syslog, and syslog-tls are supported
DRAIN_TYPE - Wether to drain log, metrics, counter, or all
API_ADDR - The address of your CF API
UAA_ADDR - the address of your UAA API
CLIENT_ID - The UAA client to fetch auth tokens given a UAA Refresh token
SKIP_CERT_VERIFY - Whether to Skip SSL Validation on outbound calls
REFRESH_TOKEN - The Refresh token to be used to get auth tokens

