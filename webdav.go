package webdav

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	wd "golang.org/x/net/webdav"

	"github.com/hacdias/webdav"
	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

func init() {
	caddy.RegisterPlugin("webdav", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
}

// WebDav is the middleware that contains the configuration for each instance.
type WebDav struct {
	Next    httpserver.Handler
	Configs []*webdav.Config
}

// ServeHTTP determines if the request is for this plugin, and if all prerequisites are met.
func (d WebDav) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	for i := range d.Configs {
		// Checks if the current request is for the current configuration.
		if !httpserver.Path(r.URL.Path).Matches(d.Configs[i].BaseURL) {
			continue
		}

		d.Configs[i].ServeHTTP(w, r)
		return 0, nil
	}

	return d.Next.ServeHTTP(w, r)
}

// setup configures a new FileManager middleware instance.
func setup(c *caddy.Controller) error {
	configs, err := parse(c)
	if err != nil {
		return err
	}

	httpserver.GetConfig(c).AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return WebDav{Configs: configs, Next: next}
	})

	return nil
}

func parse(c *caddy.Controller) ([]*webdav.Config, error) {
	configs := []*webdav.Config{}

	for c.Next() {
		conf := &webdav.Config{
			BaseURL: "/",
			Users:   map[string]*webdav.User{},
			User: &webdav.User{
				Scope:  ".",
				Rules:  []*webdav.Rule{},
				Modify: true,
			},
		}

		args := c.RemainingArgs()

		if len(args) > 0 {
			conf.BaseURL = args[0]
		}

		if len(args) > 1 {
			return nil, c.ArgErr()
		}

		conf.BaseURL = strings.TrimSuffix(conf.BaseURL, "/")
		conf.BaseURL = strings.TrimPrefix(conf.BaseURL, "/")
		conf.BaseURL = "/" + conf.BaseURL

		if conf.BaseURL == "/" {
			conf.BaseURL = ""
		}

		u := conf.User

		for c.NextBlock() {
			switch c.Val() {
			case "scope":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}

				u.Scope = c.Val()
			case "allow", "allow_r", "block", "block_r":
				ruleType := c.Val()

				if !c.NextArg() {
					return configs, c.ArgErr()
				}

				if c.Val() == "dotfiles" && !strings.HasSuffix(ruleType, "_r") {
					ruleType += "_r"
				}

				rule := &webdav.Rule{
					Allow: ruleType == "allow" || ruleType == "allow_r",
					Regex: ruleType == "allow_r" || ruleType == "block_r",
				}

				if rule.Regex {
					if c.Val() == "dotfiles" {
						rule.Regexp = regexp.MustCompile("\\/\\..+")
					} else {
						rule.Regexp = regexp.MustCompile(c.Val())
					}
				} else {
					rule.Path = c.Val()
				}

				u.Rules = append(u.Rules, rule)
			case "modify":
				if !c.NextArg() {
					u.Modify = true
					continue
				}

				val, err := strconv.ParseBool(c.Val())
				if err != nil {
					return nil, err
				}

				u.Modify = val
			default:
				if c.NextArg() {
					return nil, c.ArgErr()
				}

				val := c.Val()
				if !strings.HasSuffix(val, ":") {
					return nil, c.ArgErr()
				}

				val = strings.TrimSuffix(val, ":")

				u.Handler = &wd.Handler{
					FileSystem: wd.Dir(u.Scope),
					LockSystem: wd.NewMemLS(),
				}

				conf.Users[val] = &webdav.User{
					Rules:   conf.Rules,
					Scope:   conf.Scope,
					Modify:  conf.Modify,
					Handler: conf.Handler,
				}

				u = conf.Users[val]
			}
		}

		u.Handler = &wd.Handler{
			FileSystem: wd.Dir(u.Scope),
			LockSystem: wd.NewMemLS(),
		}

		configs = append(configs, conf)
	}

	return configs, nil
}
