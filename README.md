# trapi (WIP)

## Name

*trapi* - Transient resource record API (**Very much WIP**)

## Description

The trapi plugin allows adding transient resource records that are held in memory of the server 
instance for a given amount of time.

~~~ txt
trapi
~~~

## Metrics

If monitoring is enabled (via the *prometheus* directive) the following metric is exported:

* `coredns_trapi_request_count_answered{server}` - query count answered by the *trapi* plugin.

The `server` label indicated which server handled the request, see the *metrics* plugin for details.

## Health

This plugin implements dynamic health checking. It will always return healthy though.

## API

The API is not yet specified and for now is just a POST with a request body parsable by github.com/miekg/dns:NewRR.


## Syntax

[TODO] detailing syntax and supported directives.

## Examples

In this configuration, we are able to insert transient resource records without authentication
through the API, accessible via HTTP at 127.0.0.1:53080, and forward all other queries to 1.1.1.1.

``` corefile
. {
  trapi 127.0.0.1:53080
  forward . 1.1.1.1
}
```