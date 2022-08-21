# cloudfront-emulator

**Test your lambda@edge functions locally.**

Have you ever found it frustrating to develop lambda@edge functions? Are you
tired of editing your code, publishing a new version, updating the Cloudfront
distribution, clear the cache, waiting for propagation to finish, and waiting to
see your logs in Cloudwatch?

> **Warning**
> This project is still being developed.

## Demo

To demo this, clone the repo down and run

```bash
go run ./... example/cookie-redirect
```

## Configuration Files

```yaml
---
config:
  port: 3000 # defaults to 443
  addr: localhost # defaults to localhost
  origins:
    example:
      domain: example.com
      path: /
  behaviors:
    - path: /*
      origin: example
      events:
        viewer-request:
          path: ./ # defaults to the path passed into the emulator
          handler: index.handler
        origin-request:
          handler: index.handler
        origin-response:
          handler: index.handler
        viewer-response:
          handler: index.handler
```

## To Do

- [ ] emulator CLI command
  - [ ] use terraform to determine origin and behavior configurations
- [ ] validate header modification
  - [x] viewer-request
  - [x] origin-request
  - [ ] origin-response
  - [ ] viewer-response
