# Unbound SSH

<p align="center">
    <img width="500" src="logo.svg" />
</p>

unbound-ssh is a tool that allows you to work around the restrictions an ssh server set by an administrator. It only
needs an interactive shell access on the server to work, multi-hop servers are also supported.

the project is largely inspired by [trzsz](https://github.com/trzsz/trzsz-go)
and [SaSSHimi](https://github.com/rsrdesarrollo/SaSSHimi) projects.

# Mechanism

We do not distributed separate binaries for server and agent, so unbound-ssh binary needs to be launched on your laptop
👨🏻‍💻 in
**listen mode** and on the interactive ssh server 🖥️ in **spy mode**. The two process will then communicate in
hex-encoded text (or other available codecs) through **STDIN**/**STDOUT** streams.

Unbound SSH reads [config.toml](config.toml) to know what service to provide on
what port. This file is needed in either **listen mode** or **spy mode**.

For your convenience we have created **preflight** scripts, which gets launched if you press `<ctrl+b>` three times on a
safe directory on the server. The preflight script will upload the binary file (only if needed), config file and its
dependencies (such as the certificate for embedded_ssh service) to the server. After that, you can go ahead and launch
unbound-ssh in spy mode.

# Example

```bash
# let `unbound-ssh listen` tap into the byte stream of your connection
💻 laptop$ unbound-ssh listen -- ssh user@server.com
Connecting to server.com...

# trigger preflight script
# (it uploads unbound-ssh binary, the config file and its dependencies to the server)
🌩️ server$ <ctrl+b><ctrl+b><ctrl+b>

# run `unbound-ssh spy` on your server to connect
🌩️ server$ ./unbound-ssh spy

# 🎉 you're all good, now use the services you defined in config.toml
# the following ⬇️ subsection will show you how to define services
```

# Configuration

Aside from tweaking parameters, Unbound SSH reads [config.toml](config.toml) to know what service to provide on
what port. You are allowed to define more than one service.

## [Embedded SSH](#embedded-ssh)

If the ssh you used to connect your server is too restrictive and does not allow port-forwarding, file
transfer, non-interactive command execution, etc you can have unbound-ssh to launch its own unrestricted ssh server 🌈
and let you connect to it
from your laptop.

```toml
[[service]]
type = "embedded_ssh"
# binds to this host/port on your laptop
bind = "tcp://unbound-ssh.local:10691"
# continue reading to see how to create this
certificate = "cert/unbound-ssh.local-key.pem"
```

Encryption is mandatory for an SSH server so embedded_ssh needs a valid ssl certificate, you can
leverage [txeh](https://github.com/txn2/txeh) and
[mkcert](https://github.com/FiloSottile/mkcert) to define a local host and create a valid local certificate for it.

```shell
# define local host "unbound-ssh.local"
# that resolves to "127.0.0.1" loopback address 
brew install txn2/tap/txeh
sudo txeh add 127.0.0.1 unbound-ssh.local

# create a certificate for it
# unbound ssh uses this to encrypt traffic
brew install mkcert
mkcert -install
mkdir cert; pushd cert; mkcert unbound-ssh.local; popd
```

## Embedded Webdav

Since FTP [can not be served on simple tcp connection](https://www.jscape.com/blog/active-v-s-passive-ftp-simplified);
although not impossible, I did not bother to implement it and
opted for a webdav service for file transfer. Considering the fact that golang's ssh
server [does not support compression](https://github.com/golang/go/issues/31369), and you may not want to bother setting
up certificates and pay for the encryption
overhead that accompanies sftp, you can opt for a simple webdav server for fast quick file
transfer.

```toml
[[service]]
type = "embedded_webdav"
bind = "tcp://127.0.0.1:10690"
```

## Port Forward

For the same reasons as above, you may want to simply launch a port-forward instead of a fully-fledged ssh server. This
is how you can do it.

```toml
[[service]]
type = "port_forward"
bind = "tcp://127.0.0.1:10692"
destination = "tcp://my-database-server.us-east-1.rds.amazonaws.com:3306"
```

if a correct host name is crucial to your use case (e.g. http, https) you can use `txeh` to define a local host and
attach it to your loopback address pretty much like how it is done for the embedded ssh service.

## Echo

This service is only for testing purposes. Spy agent simply echoes back the message you send to it.

```toml
[[service]]
type = "echo"
bind = "tcp://127.0.0.1:10689"
```

# Supported Platforms

- [x] Linux (x64, arm64)
- [x] macOS (x64, arm64)
- [ ] Windows (x64) _(no plan to support it, consider using WSL instead)_

# Feature Comparison

| Feature                        | unbound-ssh      | trzsz | SaSSHimi |
|--------------------------------|------------------|-------|----------|
| Port Forward                   | ✔                | ⨯     | ⨯        |
| VPN                            | ✔ <sup>[1]</sup> | ⨯     | ⨯        |
| File Transfer                  | ✔                | ✔     | ✔        |
| Interactive-Only Shell Support | ✔                | ✔     | ⨯        |
| Multi-Hop Servers              | ✔                | ✔     | ⨯        |
| Agent Auto Install             | ✔                | ⨯     | ⨯        |
| Tmux Support                   | ⨯ <sup>[2]</sup> | ✔     | ⨯        |
| Windows Support                | ⨯ <sup>[3]</sup> | ✔     | ⨯        |

1. through [sshuttle](https://github.com/sshuttle/sshuttle) over [embedded ssh server](#embedded-ssh)
2. consider using multiple ssh sessions through [embedded ssh server](#embedded-ssh) instead
3. consider using [WSL](https://en.wikipedia.org/wiki/Windows_Subsystem_for_Linux) instead

# Future Work

Moved to [FUTUREWORK.md](FUTUREWORK.md)

# Key Library Dependencies

- [creack/pty](github.com/creack/pty) to create pty and interact with it
- [hashicorp/yamux](github.com/hashicorp/yamux) to multiplex connections over a single duplex stream
- [gliderlabs/ssh](github.com/gliderlabs/ssh) to power ssh server
- [spf13/cobra](github.com/spf13/cobra) for process args processing
