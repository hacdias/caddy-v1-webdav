package webdav

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/webdav"

	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

func init() {
	caddy.RegisterPlugin("webdav", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
}

// Config is the configuration of a WebDAV instance.
type Config struct {
	*User
	BaseURL string
	Users   map[string]*User
}

// Rule is a dissalow/allow rule.
type Rule struct {
	Regex  bool
	Allow  bool
	Path   string
	Regexp *regexp.Regexp
}

// User contains the settings of each user.
type User struct {
	Scope   string
	Modify  bool
	Rules   []*Rule
	Handler *webdav.Handler
}

// Allowed checks if the user has permission to access a directory/file
func (u User) Allowed(url string) bool {
	var rule *Rule
	i := len(u.Rules) - 1

	for i >= 0 {
		rule = u.Rules[i]

		if rule.Regex {
			if rule.Regexp.MatchString(url) {
				return rule.Allow
			}
		} else if strings.HasPrefix(url, rule.Path) {
			return rule.Allow
		}

		i--
	}

	return true
}

// WebDav is the middleware that contains the configuration for each instance.
type WebDav struct {
	Next    httpserver.Handler
	Configs []*Config
}

// ServeHTTP determines if the request is for this plugin, and if all prerequisites are met.
func (d WebDav) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	var (
		c *Config
		u *User
	)

	for i := range d.Configs {
		// Checks if the current request is for the current configuration.
		if !httpserver.Path(r.URL.Path).Matches(d.Configs[i].BaseURL) {
			continue
		}

		c = d.Configs[i]
		u = c.User

		// Gets the correct user for this request.
		username, ok := r.Context().Value(httpserver.RemoteUserCtxKey).(string)
		if ok {
			if user, ok := c.Users[username]; ok {
				u = user
			}
		}

		// Remove the BaseURL from the url path.
		r.URL.Path = strings.TrimPrefix(r.URL.Path, c.BaseURL)

		// Checks for user permissions relatively to this PATH.
		if !u.Allowed(r.URL.Path) {
			return http.StatusForbidden, nil
		}

		if r.Method == "HEAD" {
			w = newResponseWriterNoBody(w)
		}

		// If this request modified the files and the user doesn't have permission
		// to do so, return forbidden.
		if (r.Method == "PUT" || r.Method == "POST" || r.Method == "MKCOL" ||
			r.Method == "DELETE" || r.Method == "COPY" || r.Method == "MOVE") &&
			!u.Modify {
			return http.StatusForbidden, nil
		}

		// Excerpt from RFC4918, section 9.4:
		//
		// 		GET, when applied to a collection, may return the contents of an
		//		"index.html" resource, a human-readable view of the contents of
		//		the collection, or something else altogether.
		//
		// Get, when applied to collection, will return the same as PROPFIND method.
		if r.Method == "GET" {
			info, err := u.Handler.FileSystem.Stat(context.TODO(), r.URL.Path)
			if err == nil && info.IsDir() {
				r.Method = "PROPFIND"
			}
		}

		// Runs the WebDAV.
		u.Handler.ServeHTTP(w, r)
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

func parse(c *caddy.Controller) ([]*Config, error) {
	configs := []*Config{}

	for c.Next() {
		conf := &Config{
			BaseURL: "/",
			Users:   map[string]*User{},
			User: &User{
				Scope:  ".",
				Rules:  []*Rule{},
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

				rule := &Rule{
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

				u.Handler = &webdav.Handler{
					FileSystem: webdav.Dir(u.Scope),
					LockSystem: webdav.NewMemLS(),
				}

				conf.Users[val] = &User{
					Rules:   conf.Rules,
					Scope:   conf.Scope,
					Modify:  conf.Modify,
					Handler: conf.Handler,
				}

				u = conf.Users[val]
			}
		}

		u.Handler = &webdav.Handler{
			FileSystem: webdav.Dir(u.Scope),
			LockSystem: webdav.NewMemLS(),
		}

		configs = append(configs, conf)
	}

	return configs, nil
}
