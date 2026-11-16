# BBB Stress Test Client

Usage: `./bbb-stress-test [options]`

First set the `bbbHost`, `securitySalt` and others at `config.json`
```bash
jq '.securitySalt = "330a8b08c3b4c61533e1d0c5ce1ac88f" 
    | .bbbServerHost = "bbb30.bbb.imdt.dev"' config.json > tmp.json && mv tmp.json config.json
```
Or inform them as parameter when calling the application.

### Options:
  - `--config=` config.json
  - `--meetingId=` abc12
  - `--numOfUsers=` 10 _(override config)_
  - `--sendChatMessages=` true _(override config)_
  - `--securitySalt=` 2312sdfdsf2 _(override config)_
  - `--serverHost=` myserver.com _(override config)_
  - `--userJoinOrder=` asc|desc|shuffle _(override config)_

### Examples:
  - `./bbb-stress-test --config=config.json`
  - `./bbb-stress-test --config=config.json --meetingId=abc123def456`
  - `./bbb-stress-test --meetingId=abc123def456`

  
## Build
`./build.sh`

## Run
`./bbb-stress-test`

# Dump last version of subscriptions
The files in `subscriptions/` must be exactly what a real HTML5 client sends to the
BBB version under test (a field missing in the server schema makes the whole
subscription fail with `validation-failed`). Regenerate them after every BBB upgrade.

1. Stop any running `bbb-stress-test` (otherwise the bots create dump folders too).
2. Enable the dump in BBB Graphql Middleware (no `yq`? use the `sed` variant):
```bash
sudo yq eval '.dump_queries_dir = "/tmp/queries-dump"' -i /usr/share/bbb-graphql-middleware/config.yml
# or: sudo sed -i 's|^dump_queries_dir:.*|dump_queries_dir: /tmp/queries-dump|' /usr/share/bbb-graphql-middleware/config.yml
sudo rm -rf /tmp/queries-dump
sudo systemctl restart bbb-graphql-middleware
```
3. Create an empty meeting and join it as **moderator**, being the first user so you
   become **presenter** (skip/cancel the audio dialog). Close every other BBB tab:
   they reconnect after the restart and create extra `BC*` folders.
4. Open the panels that trigger the extra subscriptions:
   - **Chat** (open by default) → `GetChatData`, `chatMessages`, `IsTyping`
   - **People / user list** → `Patched_UserListSubscription`, filtered `UsersCount`
   - **Shared notes** → `getPadId`, `GetNotesLastUpdatedAt`
   - Do **not** open the connection status modal: it adds `ConnStatusReport`, which
     regular viewers never subscribe to (and it costs ~15ms per poll per cohort).
   - Go back to **Chat**, so the meeting is left like a regular user would.
5. Close the browser tab.
6. Copy `/tmp/queries-dump/<middlewareID>/BC<n>/*.txt` of your connection (the folder
   whose `CurrentPageAnnotationsQuery` has your meeting's internal id in `pageId`)
   into `subscriptions/`. Drop duplicates with the same operationName + query +
   variables (re-subscriptions caused by reopening a panel) and renumber the files
   keeping the original order. Duplicates with a different query (e.g. the two
   `UsersCount`) must be kept.
7. Disable the dump:
```bash
sudo yq eval '.dump_queries_dir = ""' -i /usr/share/bbb-graphql-middleware/config.yml
# or: sudo sed -i 's|^dump_queries_dir:.*|dump_queries_dir:|' /usr/share/bbb-graphql-middleware/config.yml
sudo systemctl restart bbb-graphql-middleware
```
8. Run `bbb-stress-test`.

Notes on how the files are used:
- `Patched_userCurrentSubscription` is sent first; the rest is sent as a batch after join.
- Files declaring `$pageId: String!` (`CurrentPageAnnotationsQuery`,
  `annotationHistoryStream`) are re-sent with the current page id whenever
  `CurrentPresentationPagesSubscription` reports a page change.

Last dump: BBB 4.0.2 (2026-11-16).
