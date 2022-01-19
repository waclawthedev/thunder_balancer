# What is ⚡ThunderBalancer?

It is fast http load balancer for your API/webpage

## Benefits

* ThunderBalancer makes choice based on experience with nodes performance (response time)
* It supports three different http server engines: http.Server, fasthttp and our custom, pure socket-based ThunderEngine
* Choosing the next node makes zero memory allocations
* TLS support out-of-box
* Wide support of HTTP 1.1 methods, cookies, etc.

## How to use
#### Option 1
Run the binary with ENV variables. Every ENV variable is required to be set.
#### Option 2
Start the docker container with ENV variables.

## ENV variables

| Variable           |                                                 Description                                                 |                   Available values | Example                                                                                |
|--------------------|:-----------------------------------------------------------------------------------------------------------:|-----------------------------------:|----------------------------------------------------------------------------------------|
| NODES              |       Nodes that will be used. If is_tls is enabled - then requests will be directed via https scheme       |                                    | [{"node":"127.0.0.1:9000", "is_tls":false}, {"node":"127.0.0.1:9001", "is_tls":false}] |
| TLS                |                                      Use TLS for incoming connections?                                      |                             ON/OFF | YES                                                                                    |
| SELECT_NODE_PERIOD |                      Makes choose new node every SELECT_NODE_PERIOD incoming requests                       |                                    | 10                                                                                     |
| CLEAN_STATS_PERIOD |               Clean stats about nodes performance every CLEAN_STATS_PERIOD incoming requests                |                                    | 10                                                                                     |
| ENGINE             |         Http server will be used to handle incoming requests. More about engines in Engines section         | THUNDER/ FAST_HTTP / STANDARD_HTTP | THUNDER                                                                                |
| HOST_PORT          |                                 IP and PORT to use for incoming connections                                 |                                    | 0.0.0.0:8080                                                                           |
| NODE_TIMEOUT       | Balancer will wait NODE_TIMEOUT msec for response from node                                               H |                                    | 1000                                                                                   |

## Engines

| Engine        | Description                                                                                              | Pros                                                                                                  | Cons                                                                                                                |
|---------------|----------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------|
| STANDARD_HTTP | Uses standard http.Server                                                                                | Very stable. Official package.                                                                        | Not so fast as fasthttp package, uses more memory allocations                                                       |
| FAST_HTTP     | Alternative http server. Can be up to 10x faster (authors declare it). Based on zero-allocation approach | Faster than http.server. Uses less RAM. [ More about it.](https://github.com/valyala/fasthttp)        | May contain more bugs, supported by maintainers of fasthttp.                                                        |
| THUNDER       | Experimental, very fast engine based on pure sockets without parsing of http request/response.           | Very fast. Uses extremely small amount of RAM. Works as pipe: retranslates the requests and responses | Experimental. Not advised for production usage until 1.0 version. Can contain bugs with some configuration of nodes |


## Changelog
#### Version 0.1-alpha1 (current)
Initial version
