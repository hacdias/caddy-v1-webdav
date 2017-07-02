# webdav

[![Build](https://img.shields.io/travis/hacdias/caddy-webdav.svg?style=flat-square)](https://travis-ci.org/hacdias/caddy-webdav)
[![community](https://img.shields.io/badge/community-forum-ff69b4.svg?style=flat-square)](https://caddy.community)
[![Go Report Card](https://goreportcard.com/badge/github.com/hacdias/caddy-webdav?style=flat-square)](https://goreportcard.com/report/hacdias/caddy-webdav)

Caddy plugin that implements WebDAV. You can download this plugin with Caddy on its [official download page](https://caddyserver.com/download).

## Syntax

```
webdav [baseurl] [scope]
```

+ **baseurl** is the place where you can access the WebDAV interface. Defaults to `/`.
+ **scope** is an absolute or relative (to the current working directory of Caddy) path that indicates the scope of the WebDAV. Defaults to `.`.

It is highly recommended to use this directive alongside with `[basicauth](https://caddyserver.com/docs/basicauth)` to protect the WebDAV interface.


## Examples

WebDAV interface on `/` for the entire file system:

```
webdav / /
```

WebDAV interface on `/` for the current working directory:

```
webdav
```

WebDAV interface on `/webdav` for `/var/www`:

```
webdav /webdav /var/www
```

WebDav interface on `/webdav` for the current working directory:

```
webdav /webdav
```