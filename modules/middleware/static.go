// Copyright 2013 The Martini Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package middleware

import (
	"log"
	"net/http"
	"path"
	"runtime"
	"strings"

	"github.com/go-martini/martini"

	"github.com/gogits/gogs/modules/setting"
)

// StaticOptions is a struct for specifying configuration options for the martini.Static middleware.
type StaticOptions struct {
	// Prefix is the optional prefix used to serve the static directory content
	Prefix string
	// SkipLogging will disable [Static] log messages when a static file is served.
	SkipLogging bool
	// IndexFile defines which file to serve as index if it exists.
	IndexFile string
	// Expires defines which user-defined function to use for producing a HTTP Expires Header
	// https://developers.google.com/speed/docs/insights/LeverageBrowserCaching
	Expires func() string
}

func prepareStaticOptions(options []StaticOptions) StaticOptions {
	var opt StaticOptions
	if len(options) > 0 {
		opt = options[0]
	}

	// Defaults
	if len(opt.IndexFile) == 0 {
		opt.IndexFile = "index.html"
	}
	// Normalize the prefix if provided
	if opt.Prefix != "" {
		// Ensure we have a leading '/'
		if opt.Prefix[0] != '/' {
			opt.Prefix = "/" + opt.Prefix
		}
		// Remove any trailing '/'
		opt.Prefix = strings.TrimRight(opt.Prefix, "/")
	}
	return opt
}

// Static returns a middleware handler that serves static files in the given directory.
func Static(directory string, staticOpt ...StaticOptions) martini.Handler {
	if runtime.GOOS == "windows" {
		if len(directory) < 2 || directory[1] != ':' {
			directory = path.Join(setting.StaticRootPath, directory)
		}
	} else if !path.IsAbs(directory) {
		directory = path.Join(setting.StaticRootPath, directory)
	}

	dir := http.Dir(directory)
	opt := prepareStaticOptions(staticOpt)

	return func(res http.ResponseWriter, req *http.Request, log *log.Logger) {
		if req.Method != "GET" && req.Method != "HEAD" {
			return
		}
		file := req.URL.Path
		// if we have a prefix, filter requests by stripping the prefix
		if opt.Prefix != "" {
			if !strings.HasPrefix(file, opt.Prefix) {
				return
			}
			file = file[len(opt.Prefix):]
			if file != "" && file[0] != '/' {
				return
			}
		}
		f, err := dir.Open(file)
		if err != nil {
			// discard the error?
			return
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			return
		}

		// try to serve index file
		if fi.IsDir() {
			// redirect if missing trailing slash
			if !strings.HasSuffix(req.URL.Path, "/") {
				http.Redirect(res, req, req.URL.Path+"/", http.StatusFound)
				return
			}

			file = path.Join(file, opt.IndexFile)
			f, err = dir.Open(file)
			if err != nil {
				return
			}
			defer f.Close()

			fi, err = f.Stat()
			if err != nil || fi.IsDir() {
				return
			}
		}

		if !opt.SkipLogging {
			log.Println("[Static] Serving " + file)
		}

		// Add an Expires header to the static content
		if opt.Expires != nil {
			res.Header().Set("Expires", opt.Expires())
		}

		http.ServeContent(res, req, file, fi.ModTime(), f)
	}
}
