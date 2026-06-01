package http

import (
	"embed"
	"io/fs"
	nethttp "net/http"
)

func NewSPAHandler(assets embed.FS, root string) nethttp.Handler {
	sub, err := fs.Sub(assets, root)
	if err != nil {
		panic("static assets not embedded: " + err.Error())
	}

	fileServer := nethttp.FileServer(nethttp.FS(sub))

	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		f, openErr := sub.Open(r.URL.Path)
		if openErr == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		if _, indexErr := sub.Open("index.html"); indexErr != nil {
			nethttp.Error(w, "not found", nethttp.StatusNotFound)
			return
		}

		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
