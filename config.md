---
layout: default
title: pages.reference
parent: Reference
nav_order: 2
---

# Configuration Reference
{: .no_toc }

<!-- ## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc} -->

---


## backend

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|chainId|Optionally set the Chain ID of the blockchain. Otherwise the Network ID will be queried, and used as the Chain ID in signing|number|`-1`
|connectionTimeout|The maximum amount of time that a connection is allowed to remain with no data transmitted|[`time.Duration`](https://pkg.go.dev/time#Duration)|`30s`
|expectContinueTimeout|See [ExpectContinueTimeout in the Go docs](https://pkg.go.dev/net/http#Transport)|[`time.Duration`](https://pkg.go.dev/time#Duration)|`1s`
|headers|Adds custom headers to HTTP requests|`map[string]string`|`<nil>`
|idleTimeout|The max duration to hold a HTTP keepalive connection between calls|[`time.Duration`](https://pkg.go.dev/time#Duration)|`475ms`
|maxConnsPerHost|The max number of connections, per unique hostname. Zero means no limit|`int`|`0`
|maxIdleConns|The max number of idle connections to hold pooled|`int`|`100`
|maxIdleConnsPerHost|The max number of idle connections, per unique hostname. Zero means net/http uses the default of only 2.|`int`|`100`
|passthroughHeadersEnabled|Enable passing through the set of allowed HTTP request headers|`boolean`|`false`
|requestTimeout|The maximum amount of time that a request is allowed to remain open|[`time.Duration`](https://pkg.go.dev/time#Duration)|`30s`
|tlsHandshakeTimeout|The maximum amount of time to wait for a successful TLS handshake|[`time.Duration`](https://pkg.go.dev/time#Duration)|`10s`
|url|URL for the backend JSON/RPC server / blockchain node|url|`<nil>`

## backend.auth

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|password|Password|`string`|`<nil>`
|username|Username|`string`|`<nil>`

## backend.dns

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|servers|An optional list of DNS server addresses (host or host:port, port defaults to 53) to use instead of the system resolver. Setting this forces use of Go's built-in DNS resolver.|`[]string`|`<nil>`
|timeout|The dial timeout when contacting a configured DNS server|[`time.Duration`](https://pkg.go.dev/time#Duration)|`<nil>`

## backend.net

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|cidrDenylist|A list of CIDR ranges to which outbound connections are blocked, as a core SSRF mitigation. Empty by default.|`[]string`|`<nil>`

## backend.proxy

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|url|Optional HTTP proxy URL|url|`<nil>`

## backend.retry

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|count|The maximum number of times to retry|`int`|`5`
|enabled|Enables retries|`boolean`|`false`
|errorStatusCodeRegex|The regex that the error response status code must match to trigger retry|`string`|`<nil>`
|factor|The retry backoff factor|`float32`|`2`
|initWaitTime|The initial retry delay|[`time.Duration`](https://pkg.go.dev/time#Duration)|`250ms`
|maxWaitTime|The maximum retry delay|[`time.Duration`](https://pkg.go.dev/time#Duration)|`30s`

## backend.throttle

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|burst|The maximum number of requests that can be made in a short period of time before the throttling kicks in.|`int`|`<nil>`
|requestsPerSecond|The average rate at which requests are allowed to pass through over time.|`int`|`<nil>`

## backend.tls

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|ca|The TLS certificate authority in PEM format (this option is ignored if caFile is also set)|`string`|`<nil>`
|caFile|The path to the CA file for TLS on this API|`string`|`<nil>`
|cert|The TLS certificate in PEM format (this option is ignored if certFile is also set)|`string`|`<nil>`
|certFile|The path to the certificate file for TLS on this API|`string`|`<nil>`
|clientAuth|Enables or disables client auth for TLS on this API|`string`|`<nil>`
|enabled|Enables or disables TLS on this API|`boolean`|`false`
|insecureSkipHostVerify|When to true in unit test development environments to disable TLS verification. Use with extreme caution|`boolean`|`<nil>`
|key|The TLS certificate key in PEM format (this option is ignored if keyFile is also set)|`string`|`<nil>`
|keyFile|The path to the private key file for TLS on this API|`string`|`<nil>`
|requiredDNAttributes|A set of required subject DN attributes. Each entry is a regular expression, and the subject certificate must have a matching attribute of the specified type (CN, C, O, OU, ST, L, STREET, POSTALCODE, SERIALNUMBER are valid attributes)|`map[string]string`|`<nil>`

## backend.ws

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|backgroundConnect|When true the connection is established in the background with infinite reconnect (makes initialConnectAttempts redundant when set)|`boolean`|`false`
|connectionTimeout|The amount of time to wait while establishing a connection (or auto-reconnection)|[`time.Duration`](https://pkg.go.dev/time#Duration)|`45s`
|heartbeatInterval|The amount of time to wait between heartbeat signals on the WebSocket connection|[`time.Duration`](https://pkg.go.dev/time#Duration)|`30s`
|initialConnectAttempts|The number of attempts FireFly will make to connect to the WebSocket when starting up, before failing|`int`|`5`
|path|The WebSocket sever URL to which FireFly should connect|WebSocket URL `string`|`<nil>`
|readBufferSize|The size in bytes of the read buffer for the WebSocket connection|[`BytesSize`](https://pkg.go.dev/github.com/docker/go-units#BytesSize)|`16Kb`
|url|URL to use for WebSocket - overrides url one level up (in the HTTP config)|`string`|`<nil>`
|writeBufferSize|The size in bytes of the write buffer for the WebSocket connection|[`BytesSize`](https://pkg.go.dev/github.com/docker/go-units#BytesSize)|`16Kb`

## cors

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|credentials|CORS setting to control whether a browser allows credentials to be sent to this API|`boolean`|`true`
|debug|Whether debug is enabled for the CORS implementation|`boolean`|`false`
|enabled|Whether CORS is enabled|`boolean`|`true`
|headers|CORS setting to control the allowed headers|`[]string`|`[*]`
|maxAge|The maximum age a browser should rely on CORS checks|[`time.Duration`](https://pkg.go.dev/time#Duration)|`600`
|methods| CORS setting to control the allowed methods|`[]string`|`[GET POST PUT PATCH DELETE]`
|origins|CORS setting to control the allowed origins|`[]string`|`[*]`

## fileWallet

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|defaultPasswordFile|Optional default password file to use, if one is not specified individually for the key (via metadata, or file extension)|string|`<nil>`
|disableListener|Disable the filesystem listener that automatically detects the creation of new keystore files|boolean|`<nil>`
|enabled|Whether the Keystore V3 filesystem wallet is enabled|boolean|`true`
|path|Path on the filesystem where the metadata files (and/or key files) are located|string|`<nil>`
|signerCacheSize|Maximum of signing keys to hold in memory|number|`250`
|signerCacheTTL|How long ot leave an unused signing key in memory|duration|`24h`

## fileWallet.filenames

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|passwordExt|Optional to use to look up password files, that sit next to the key files directly. Alternative to metadata when you have a password per keystore|string|`<nil>`
|passwordPath|Optional directory in which to look for the password files, when passwordExt is configured. Default is the wallet directory|string|`<nil>`
|passwordTrimSpace|Whether to trim leading/trailing whitespace (such as a newline) from the password when loaded from file|boolean|`true`
|primaryExt|Extension for key/metadata files named by <ADDRESS>.<EXT>|string|`<nil>`
|primaryMatchRegex|Regular expression run against key/metadata filenames to extract the address (takes precedence over primaryExt)|regexp|`<nil>`
|with0xPrefix|When true and passwordExt is used, password filenames will be generated with an 0x prefix|boolean|`<nil>`

## fileWallet.metadata

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|format|Set this if the primary key file is a metadata file. Supported formats: auto (from extension) / filename / toml / yaml / json (please quote "0x..." strings in YAML)|string|`auto`
|keyFileProperty|Go template to look up the key-file path from the metadata. Example: '{{ index .signing "key-file" }}'|go-template|`<nil>`
|passwordFileProperty|Go template to look up the password-file path from the metadata|go-template|`<nil>`

## log

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|compress|Determines if the rotated log files should be compressed using gzip|`boolean`|`<nil>`
|filename|Filename is the file to write logs to.  Backup log files will be retained in the same directory|`string`|`<nil>`
|filesize|MaxSize is the maximum size the log file before it gets rotated|[`BytesSize`](https://pkg.go.dev/github.com/docker/go-units#BytesSize)|`100m`
|forceColor|Force color to be enabled, even when a non-TTY output is detected|`boolean`|`<nil>`
|includeCodeInfo|Enables the report caller for including the calling file and line number, and the calling function. If using text logs, it uses the logrus text format rather than the default prefix format.|`boolean`|`false`
|level|The log level - error, warn, info, debug, trace|`string`|`info`
|maxAge|The maximum time to retain old log files based on the timestamp encoded in their filename.|[`time.Duration`](https://pkg.go.dev/time#Duration)|`24h`
|maxBackups|Maximum number of old log files to retain|`int`|`2`
|noColor|Force color to be disabled, event when TTY output is detected|`boolean`|`<nil>`
|timeFormat|Custom time format for logs|[Time format](https://pkg.go.dev/time#pkg-constants) `string`|`2006-01-02T15:04:05.000Z07:00`
|utc|Use UTC timestamps for logs|`boolean`|`false`

## log.json

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|enabled|Enables JSON formatted logs rather than text. All log color settings are ignored when enabled.|`boolean`|`false`

## log.json.fields

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|file|configures the JSON key containing the calling file|`string`|`file`
|func|Configures the JSON key containing the calling function|`string`|`func`
|level|Configures the JSON key containing the log level|`string`|`level`
|message|Configures the JSON key containing the log message|`string`|`message`
|timestamp|Configures the JSON key containing the timestamp of the log|`string`|`@timestamp`

## server

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|address|Local address for the JSON/RPC server to listen on|string|`127.0.0.1`
|port|Port for the JSON/RPC server to listen on|number|`8545`
|publicURL|External address callers should access API over|string|`<nil>`
|readTimeout|The maximum time to wait when reading from an HTTP connection|duration|`15s`
|shutdownTimeout|The maximum amount of time to wait for any open HTTP requests to finish before shutting down the HTTP server|[`time.Duration`](https://pkg.go.dev/time#Duration)|`10s`
|writeTimeout|The maximum time to wait when writing to a HTTP connection|duration|`15s`

## server.auth

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|type|The auth plugin to use for server side authentication of requests|`string`|`<nil>`

## server.auth.basic

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|passwordfile|The path to a .htpasswd file to use for authenticating requests. Passwords should be hashed with bcrypt.|`string`|`<nil>`

## server.tls

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|ca|The TLS certificate authority in PEM format (this option is ignored if caFile is also set)|`string`|`<nil>`
|caFile|The path to the CA file for TLS on this API|`string`|`<nil>`
|cert|The TLS certificate in PEM format (this option is ignored if certFile is also set)|`string`|`<nil>`
|certFile|The path to the certificate file for TLS on this API|`string`|`<nil>`
|clientAuth|Enables or disables client auth for TLS on this API|`string`|`<nil>`
|enabled|Enables or disables TLS on this API|`boolean`|`false`
|insecureSkipHostVerify|When to true in unit test development environments to disable TLS verification. Use with extreme caution|`boolean`|`<nil>`
|key|The TLS certificate key in PEM format (this option is ignored if keyFile is also set)|`string`|`<nil>`
|keyFile|The path to the private key file for TLS on this API|`string`|`<nil>`
|requiredDNAttributes|A set of required subject DN attributes. Each entry is a regular expression, and the subject certificate must have a matching attribute of the specified type (CN, C, O, OU, ST, L, STREET, POSTALCODE, SERIALNUMBER are valid attributes)|`map[string]string`|`<nil>`