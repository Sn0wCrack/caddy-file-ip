# caddy-file-ip

Caddy module that provides `http.ip_sources` output by reading IP address ranges from local files.

## Installation

Using xcaddy:

```bash
xcaddy build --with github.com/caddyserver/caddy-file-ip
```

Or using `caddy add-package`:

```bash
caddy add-package github.com/caddyserver/caddy-file-ip
```

## Configuration

The module reads IP ranges (in CIDR notation) from local files. Each line can contain a single IP address or CIDR prefix. Lines starting with `#` are treated as comments and ignored.

### JSON Configuration

```json
{
  "apps": {
    "http": {
      "servers": {
        "example": {
          "listen": [":8080"],
          "routes": [{
            "match": [{
              "remote_ip": {
                "source_ranges": ["@{file_ip_source}"]
              }
            }],
            "handle": [{
              "handler": "static_response",
              "body": "Access Denied"
            }]
          }]
        }
      }
    }
  },
  "modules": {
    "http.ip_sources": {
      "file_ip_source": {
        "files": ["/path/to/ip-ranges.txt"],
        "watch": true,
        "interval": "1h"
      }
    }
  }
}
```

### Caddyfile Configuration

```
# Global options or server block
{
    ip_sources file /path/to/ip-ranges.txt {
        watch
        interval 1h
    }
}
```

Or inline:

```
@denied remote_ip source_ranges file /path/to/ip-ranges.txt
```

## Options

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `files` | List of file paths containing IP ranges (CIDR notation) | array of string | required |
| `watch` | Enable file watching using fsnotify to automatically reload on changes | bool | false |
| `interval` | Refresh interval for re-reading files (use with `watch` disabled) | duration | 0 (no refresh) |

## File Format

Each file should contain one IP range per line in CIDR notation:

```
# Example ip-ranges.txt
10.0.0.0/8
192.168.0.0/16
172.16.0.0/12
2001:db8::/32
```

## Usage with trusted_proxies

```
trusted_proxies file /path/to/ip-ranges.txt {
    watch
}
```
