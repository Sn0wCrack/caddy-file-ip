# caddy-file-ip

Caddy module that provides `http.ip_sources` output by reading IP address ranges from local files.

## Installation

Using xcaddy:

```bash
xcaddy build --with github.com/sn0wcrack/caddy-file-ip
```

Or using `caddy add-package`:

```bash
caddy add-package github.com/sn0wcrack/caddy-file-ip
```

## Configuration

The module reads IP ranges (in CIDR notation) from local files. Each line can contain a single IP address or CIDR prefix. Lines starting with `#` are treated as comments and ignored.

## Options

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `file` | List of file paths containing IP ranges (CIDR notation) | array of string | required |
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

## Usage with `trusted_proxies`

```
trusted_proxies file /path/to/ip-ranges.txt {
    watch
}

trusted_proxies file {
    file /path/to/ip-ranges-1.txt
    file /path/to/ip-ranges-2.txt
    interval 1m
}
```

## Usage with `dynamic_client_ip`

You can get `dynamic_client_ip` from [here](https://github.com/tuzzmaniandevil/caddy-dynamic-clientip)

```caddy
@denied dynamic_client_ip file {
    file /path/to/ip-ranges.txt
    watch
}
abort @denied
```
