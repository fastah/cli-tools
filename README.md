# `whereis`- IP geolocation tool

`whereis` - a CLI tool for IP address to approximate location determination via [Fastah's REST API available on the AWS Marketplace](https://aws.amazon.com/marketplace/pp/B084VR96P3). It uses underlying statistical models to provide city-level results, timezone information, country and continent information. 

## Initialization

* `whereis init <API key from Fastah>` ; obtain this key by signing up on [AWS Marketplace](https://aws.amazon.com/marketplace/pp/B084VR96P3)

## Looking up IP geolocation

Like a good Unix-y too, `whereis` can be specified a single IP to geolocate, or be asked to read a list via standard input

* A single IP
** `whereis --ip 202.94.72.116`

* A piped collection of IPs, one per line via standard input
** `printf " 202.94.72.116 \n 1.1.1.1 \n" | whereis --ip -`

## High-performance guide : fast response times using HTTP/2

* This tool maximizes request/response throughput by speaking HTTP/2 with the Fastah API endpoint
* We strongly suggest you use its HTTPClient setup in your own code for minimum API latency (see `root.go` file in directory `whereis/cmd/`)
* To verify that this CLI tool is using HTTP/2 on your server/laptop:
** `GODEBUG=http2debug=1; printf " 202.94.72.116 \n 1.1.1.1 \n" | go run whereis/main.go --ip -`
** If you see one mention of `Transport creating client conn`, you have verified that Go and the Fastah server are successfully using HTTP/2
** If you see no logging output at all, the Go client has used HTTP1.1

## TODO

* Add timing (performance benchmarking) for Fastah REST API calls over HTTP/2
