package webdav

import (
	"context"
	"net/http"
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
	BaseURL    string
	Scope      string
	FileSystem webdav.FileSystem
	Handler    *webdav.Handler
}

// WebDav is the middleware that contains the configuration for each instance.
type WebDav struct {
	Next    httpserver.Handler
	Configs []*Config
}

// ServeHTTP determines if the request is for this plugin, and if all prerequisites are met.
func (d WebDav) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	for i := range d.Configs {
		// Checks if the current request is for the current configuration.
		if !httpserver.Path(r.URL.Path).Matches(d.Configs[i].BaseURL) {
			continue
		}

		c := d.Configs[i]

		r.URL.Path = strings.TrimPrefix(r.URL.Path, c.BaseURL)

		if r.Method == "HEAD" {
			w = newResponseWriterNoBody(w)
		}

		// Excerpt from RFC4918, section 9.4:
		//
		// 		GET, when applied to a collection, may return the contents of an
		//		"index.html" resource, a human-readable view of the contents of
		//		the collection, or something else altogether.
		//
		// Get, when applied to collection, will return the same as PROPFIND method.
		if r.Method == "GET" {
			info, err := c.FileSystem.Stat(context.TODO(), r.URL.Path)
			if err == nil && info.IsDir() {
				r.Method = "PROPFIND"
			}
		}

		// Runs the WebDAV.
		d.Configs[i].Handler.ServeHTTP(w, r)
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
			Scope:   ".",
		}

		args := c.RemainingArgs()

		if len(args) > 0 {
			conf.BaseURL = args[0]
		}

		if len(args) > 1 {
			conf.Scope = args[1]
		}

		if len(args) > 2 {
			return nil, c.ArgErr()
		}

		conf.BaseURL = strings.TrimSuffix(conf.BaseURL, "/")
		conf.BaseURL = strings.TrimPrefix(conf.BaseURL, "/")
		conf.BaseURL = "/" + conf.BaseURL

		if conf.BaseURL == "/" {
			conf.BaseURL = ""
		}

		conf.FileSystem = webdav.Dir(conf.Scope)
		conf.Handler = &webdav.Handler{
			FileSystem: conf.FileSystem,
			LockSystem: webdav.NewMemLS(),
		}

		configs = append(configs, conf)
	}

	return configs, nil
}
