package caddy_file_ip

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

func (m *FileIPSource) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next()

	if d.NextArg() {
		m.Files = append(m.Files, d.Val())
	}

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		switch d.Val() {
		case "file":
			if !d.NextArg() {
				return d.ArgErr()
			}
			m.Files = append(m.Files, d.Val())
		case "watch":
			if d.NextArg() {
				return d.ArgErr()
			}
			m.Watch = true
		case "interval":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return err
			}
			m.Interval = caddy.Duration(val)
		default:
			return d.ArgErr()
		}
	}

	return nil
}
