# trapi (WIP)

## Name

*trapi* - Temporary resource record API (**WIP**)

## Description

The trapi plugin allows adding temporary resource records that are held in memory of the server 
instance for a given amount of time. It increases the serial number for every added record, as well as 
after record expiry to make zone transfer possible. Slaves are notified on every API call, but not 
yet on record expiry. 

## Syntax

~~~ txt
trapi [LISTEN ADDRESS]
~~~
[TODO] detailing syntax and supported directives.

## Health

This plugin implements dynamic health checking. It will always return healthy though.

## API

The API is not stable yet and for now is just a POST with keys `origin` (required), `rr` (required) and `ttl`,
where `rr` can be specified multiple times and must be a resource record string parsable by `github.com/miekg/dns:NewRR`. 
If `ttl` is not specified the `ttl` of the resource record is taken instead.

## Examples

In this configuration, we are able to insert temporary resource records with an authentication token
via the API, accessible by HTTP on `127.0.0.1:53080` and serve the zone(s) specified in `example.org.db`.

``` corefile
. {
  trapi 127.0.0.1:53080 {
    token abc
  }
  file example.org.db {
    transfer to 192.0.2.1
  }
}
```

Insert a resource record via the API (with `multipart/form-data`):
```
curl -v -F 'token=abc' -F 'ttl=60' -F 'origin=example.org' -F 'rr=example.org. 7200 IN TXT foo' 127.0.0.1:53080
```

Or alternatively (with `application/x-www-form-urlencoded`):
```
curl -v -d 'token=abc&ttl=60&origin=example.org&rr=example.org. 7200 IN TXT foo' 127.0.0.1:53080
```

## Compilation

The plugin should be above the `file` plugin and below the `dnssec` plugin in `plugin.cfg`.
