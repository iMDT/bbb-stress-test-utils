# BBB Stress Test Client

Usage: `./bbb-stress-test [options]`

First set the `bbbHost`, `securitySalt` and others at `config.json`
```bash
jq '.securitySalt = "330a8b08c3b4c61533e1d0c5ce1ac88f" 
    | .bbbServerHost = "bbb30.bbb.imdt.dev"' config.json > tmp.json && mv tmp.json config.json
```
Or inform them as parameter when calling the application.

### Options:
  - `--config` config.json
  - `--meetingId` abc12
  - `--numOfUsers` 10 _(override config)_
  - `--sendChatMessages` true _(override config)_
  - `--securitySalt` 2312sdfdsf2 _(override config)_
  - `--serverHost` myserver.com _(override config)_

### Examples:
  - `./bbb-stress-test --config=config.json`
  - `./bbb-stress-test --config=config.json --meetingId=abc123def456`
  - `./bbb-stress-test --meetingId=abc123def456`

  
## Build
`./build.sh`

## Run
`./bbb-stress-test`

# Dump last version of subscriptions
- Configure BBB Graphql Middleware to dump the queries
```bash
sudo yq eval '.dump_queries_dir = "/tmp/queries-dump"' -i /usr/share/bbb-graphql-middleware/config.yml
sudo systemctl restart bbb-graphql-middleware
```
- Join a session with a user
- Close the browser
- Copy the generated subscriptions in `/tmp/queries-dump/middlewareID/BC0001/*.txt`
- Paste the queries in the directory `subscriptions` placed in the same directory of `bbb-stress-test`
- Disable dump of queries in BBB Graphql Middleware
```bash
sudo yq eval '.dump_queries_dir = ""' -i /usr/share/bbb-graphql-middleware/config.yml
sudo systemctl restart bbb-graphql-middleware
```
- Run `bbb-stress-test`
