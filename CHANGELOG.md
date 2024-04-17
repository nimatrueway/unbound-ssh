# CHANGELOG

## v1.0-alpha6

- Fix bug in `preflight` script, specifically the binary uploading step where cancellation did not restore the tty to
  cooked mode

## v1.0-alpha5

- Provide better RPC infrastructure for listen-mode and spy-mode to communicate
- Solid infrastructure for running preflight scripts
- Better incremental cancellable preflight upload script
- Add functional tests for preflight script, and echo service
- Change preflight trigger to simple 3x(Ctrl+G) key sequence
- Add release script