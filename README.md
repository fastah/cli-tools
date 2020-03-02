# `whereis`- IP geolocation tool

`whereis` - a CLI tool for IP address to approximate location determination via [Fastah's REST API available on the AWS Marketplace](https://aws.amazon.com/marketplace/pp/B084VR96P3). It uses underlying statistical models to provide city-level results, timezone information, country and continent information. 

## Initialization

* `whereis init <API key from Fastah>` ; obtain this key by signing up on [AWS Marketplace](https://aws.amazon.com/marketplace/pp/B084VR96P3)
* `whereis --ip 202.94.72.116` ; lookup geo-location information via the Fastah API service

## TODO

* Cleanup MMDB comparison/bake-off logic
* Add timing (performance benchmarking) for Fastah REST API calls over HTTP/2