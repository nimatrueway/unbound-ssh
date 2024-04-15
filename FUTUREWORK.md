# Future Work

- [ ] remove wildcard bullshit from the config file
- [ ] audit all timeout usages and separate them
- [ ] make it work without `config.toml` 
  - [ ] make it work only with args instead of config.toml

# Idea Dump

- [ ] implement gui.transfer
- [ ] improve command executor (fire-and-forget, single-command-capture(<-), capture-result, capture-result-and-error)
- [ ] build foundation for better user-feedback / action while especially in non-transfer mode
- [ ] bring back read-close on yamux forwarder
- [ ] add a lot of comments on the config file
- [ ] display live transfer stats
- [ ] add test for preflight
- [ ] create clean public documentation like mitmproxy
- [ ] github workflow for release
- [ ] productionize preflight to make unbound-ssh executable, and download config/cert
- [ ] preflight triggerred using ctrl+b twice
- [ ] preflight to blink on top-right corner displaying preflight mode, plus press ctrl+c to cancel
- [ ] unit test Send&Receive of control stream
- [ ] clean code and refactor, write test (all methods should be synchronous, no goroutine leaks, no double close, etc.,
  proper coherent re-usable modules)
- [ ] add timeout to yamux control handshake. gracefully close if not connected.
- [ ] improve test coverage and documentation
- [ ] better logging of the stream
  - [ ] create more efficient custom codecs
- [ ] support tmux
- [ ] offer other fast working codecs (e.g. base64, base32, etc.)
- [ ] automatically reconnect on connection loss