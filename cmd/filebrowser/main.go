package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/cj123/go-filebrowser"
)

var dir string

func init() {
	flag.StringVar(&dir, "dir", ".", "directory to serve files from")
	flag.Parse()
}

func main() {
	browser, err := filebrowser.New(&filebrowser.FS{Base: dir}, filebrowser.StandaloneHTMLTemplate)

	if err != nil {
		panic(err)
	}

	log.Fatal(http.ListenAndServe("127.0.0.1:7788", browser))
}
