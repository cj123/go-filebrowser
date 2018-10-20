package filebrowser

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
)

var ErrAccessDenied = errors.New("filebrowser: access denied")

type ByType []os.FileInfo

func (b ByType) Len() int {
	return len(b)
}

func (b ByType) Less(i, j int) bool {
	if b[i].Name() == ".." {
		return true
	}

	if b[i].IsDir() && !b[j].IsDir() {
		return true
	}

	return false
}

func (b ByType) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

type Template string

type FileSystem interface {
	Open(name string) (File, error)
	Stat(name string) (os.FileInfo, error)
	Walk(root string, walkFunc filepath.WalkFunc) error
	ReadDir(root string) ([]os.FileInfo, error)
	Abs(path string) (string, error)
}

type File interface {
	io.Closer
	io.Reader
	io.ReaderAt
	io.Seeker
	Stat() (os.FileInfo, error)
}

type FS struct {
	Base string
}

func (f *FS) Open(name string) (File, error) {
	return os.Open(filepath.Join(f.Base, name))
}

func (f *FS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(filepath.Join(f.Base, name))
}

func (f *FS) Walk(root string, walkFunc filepath.WalkFunc) error {
	return filepath.Walk(filepath.Join(f.Base, root), walkFunc)
}

func (f *FS) ReadDir(root string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(filepath.Join(f.Base, root))
}

func (f *FS) Abs(path string) (string, error) {
	return filepath.Abs(filepath.Join(f.Base, path))
}

type Browser struct {
	fs       FileSystem
	rootAbs  string
	template Template
}

type previousDirectory struct {
	os.FileInfo
}

func (p *previousDirectory) Name() string {
	return ".."
}

func New(fs FileSystem, template Template) (*Browser, error) {
	rootAbs, err := fs.Abs("")

	if err != nil {
		return nil, err
	}

	return &Browser{fs: fs, template: template, rootAbs: filepath.Clean(rootAbs)}, nil
}

func (b *Browser) FileListing(path string, w io.Writer) error {
	if path == "" {
		path = "/"
	}

	path = filepath.Clean(path)

	abs, err := b.fs.Abs(path)

	if err != nil {
		return err
	} else if len(abs) < len(b.rootAbs) {
		return ErrAccessDenied
	}

	files, err := b.fs.ReadDir(path)

	if err != nil {
		return err
	}

	fmt.Println(abs)

	if abs != "/" && abs != "." && abs != b.rootAbs {
		previous, err := b.fs.Stat("..")

		if err != nil {
			return err
		}

		files = append(files, &previousDirectory{previous})
	}

	sort.Sort(ByType(files))

	t, err := template.New("files").Parse(string(b.template))

	if err != nil {
		return err
	}

	unescaped, err := url.PathUnescape(path)

	if err != nil {
		return err
	}

	return t.Execute(w, map[string]interface{}{
		"Files": files,
		"Path":  path,
		"UnescapedPath":  filepath.Clean(filepath.Join(b.rootAbs, unescaped)),
	})
}

func (b *Browser) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	err := b.FileListing(path, w)

	if err == ErrAccessDenied {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	} else if err != nil {
		b.error(w, r, err)
		return
	}
}

func (b *Browser) error(w http.ResponseWriter, r *http.Request, err error) {
	log.Println("error: ", err)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("an error occurred"))
}

var StandaloneHTMLTemplate Template = `
<html>
<body>
` + FilesOnlyHTMLTemplate + `
</body>
</html>
`

const FilesOnlyHTMLTemplate Template = `
	{{ $path := .Path }}

	<strong>{{ .UnescapedPath }}</strong><br><br>

	<table>
		{{ range $index, $file := .Files }}
			<tr>
			<td>
				{{ $file.Mode.String }}
			</td>
			<td>
				{{ $file.ModTime.Format "Jan 02 2006" }}
			</td>

			<td>
			{{ if $file.IsDir }}
				{{ if ne $path "/" }}
					<a href="?path={{ $path }}%2f{{ $file.Name }}">
				{{ else }}
					<a href="?path={{ $file.Name }}">
				{{ end }}
			{{ end }}

			{{ $file.Name }}

			{{ if $file.IsDir }}
				</a>
			{{ end }}
			</td>
			<td>
				{{ $file.Size }}
			</td>
			</tr>
		{{ end }}
	</table>
`
