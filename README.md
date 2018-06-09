# 5medias

[5medias] is a simple [SOCKS5] ([rfc]) proxy.

It only supports TCP CONNECT commands, and static username/password
authentication.

This authentication is only useful to prevent the most basic unauthorized
access, but it is not really secure and it's extremely easy to sniff. Don't
rely on it for real security.

## Install

5medias is written in [Go], so to download and install the last version you
can run:

```sh
go install blitiri.com.ar/go/5medias
```


[5medias]: https://blitiri.com.ar/git/r/5medias
[SOCKS5]: https://en.wikipedia.org/wiki/SOCKS
[rfc]: https://www.ietf.org/rfc/rfc1928.txt
[Go]: https://golang.org
