# trapi (WIP)

## Name

*trapi* - Temporary resource record API (**WIP**)

## Description

The trapi plugin allows adding temporary resource records that are held in memory of the server 
instance for a given amount of time. It increases the serial number for every added record and
after record expiry to make master-slave replication possible. It does not yet trigger a NOTIFY
to any slaves.

## Syntax

~~~ txt
trapi [LISTEN ADDRESS]
~~~

## Health

This plugin implements dynamic health checking. It will always return healthy though.

## API

The API is not stable yet and for now is just a POST with keys `origin` (required), `rr` (required) and `ttl`,
where `rr` can be specified multiple times and must be a resource record string parsable by `github.com/miekg/dns:NewRR`.
If `ttl` is not specified the `ttl` of the resource record is taken instead.

## Syntax

[TODO] detailing syntax and supported directives.

## Examples

In this configuration, we are able to insert temporary resource records without authentication
through the API, accessible via HTTP at 127.0.0.1:53080, and forward all other queries to 1.1.1.1.

``` corefile
. {
  trapi 127.0.0.1:53080
  forward . 1.1.1.1
}
```

## Compilation

The plugin should be above the `file` plugin and below the `dnssec` plugin in `plugin.cfg`.