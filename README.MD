Task for the position of Golang developer.

You need to create an HTTP service that can restrict
number of requests (rate limit) from one IPv4 subnet.
If there are no restrictions, then you need to issue the same static content.

Requirements:
 - language: Go
 - code should be posted on GitHub
 - response must conform to RFC 6585 specification
 - The IP must be extracted from the X-Forwarded-For header

Configuration options:
 - subnet: /24 (mask 255.255.255.0)
 - request limit per minute
 - limit time 

Example: after 20 requests from IP 37.147.14.178 and 80 requests from IP 37.147.14.1
the service returns a 429 error for any request from the 37.147.14.0/24 subnet within the next two minutes.

Additionally:
- test coverage
- containerization, the ability to run with docker-compose up
- a separate handler for resetting the prefix limit





**Описание запуска:**

`docker-compose up -d`

The server will start on 3000 port by default.

Port can be changed. This setting is stored in [port]: config/config.yaml

There are rate limit settings in the same config file.

>curl localhost:3000/run -H 'X-Forwarded-For: 123.45.67.89'
>
>curl -X POST localhost:3000/reset -H 'Accept: application/json' -d '{"IP":"123.45.67.89"}'

Example:

![](../../../Pictures/Screenshot from 2023-03-04 11-45-44.png)

**Tests run:**

Simple run
>make test

Tests run with race condition detection and coverage
>make test_coverage

The same as in previous, but additionally it show uncovered code.
>make test_coverage-html













**My actual CV:**

https://docs.google.com/document/d/1K8SK-AoKhG7rv40dmRrzb4sxN4-SsD5HDNsy1fQVkrM/edit?usp=sharing