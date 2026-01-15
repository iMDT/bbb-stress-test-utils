# BBB Stress Test Client

Usage: `./bbb-stress-test [options]`

First set the `bbbUrl`, `hasuraWs`, `securitySalt` and others at `config.json`

### Options:
  - `--config` config.json
  - `--meetingId` abc12

### Examples:
  - `./bbb-stress-test --config=config.json`
  - `./bbb-stress-test --config=config.json --meetingId=abc123def456`
  - `./bbb-stress-test --meetingId=abc123def456`

  
## Build
`./build.sh`

## Run
`./bbb-stress-test`
