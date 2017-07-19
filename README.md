# webdav

[![Build](https://img.shields.io/travis/hacdias/caddy-webdav.svg?style=flat-square)](https://travis-ci.org/hacdias/caddy-webdav)
[![community](https://img.shields.io/badge/community-forum-ff69b4.svg?style=flat-square)](https://caddy.community)
[![Go Report Card](https://goreportcard.com/badge/github.com/hacdias/caddy-webdav?style=flat-square)](https://goreportcard.com/report/hacdias/caddy-webdav)

Caddy plugin that implements WebDAV. You can download this plugin with Caddy on its [official download page](https://caddyserver.com/download).

## Syntax

```
webdav [url] {
    scope       path
    allow       path
    allow_r     regex
    block       path
    block_r     regex
}
```

+ **url** is the place where you can access the WebDAV interface. Defaults to `/`.
+ **scope** is an absolute or relative (to the current working directory of Caddy) path that indicates the scope of the WebDAV. Defaults to `.`.
+ **allow** and **block** are used to allow or deny access to specific files or directories using their relative path to the scope.
+ **allow_r** and **block_r** and variations of the previous options but you are able to use regular expressions with them.

It is highly recommended to use this directive alongside with [`basicauth`](https://caddyserver.com/docs/basicauth) to protect the WebDAV interface.

```
webdav {
    # You set the global configurations here and
    # all the users will inherit them.
    user1:
    # Here you can set specific settings for the 'user1'.
    # They will override the global ones for this specific user.
}
```

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