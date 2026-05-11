package main

import (
	"context"
	"io/fs"
	"os"

	"golang.org/x/net/webdav"

	swaggerFiles "github.com/swaggo/files/v2"
)

var swaggerHandler = func() *webdav.Handler {
	h := &webdav.Handler{
		FileSystem: webdav.NewMemFS(),
		LockSystem: webdav.NewMemLS(),
	}

	fs.WalkDir(swaggerFiles.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return h.FileSystem.Mkdir(context.Background(), path, 0755)
		}
		data, err := fs.ReadFile(swaggerFiles.FS, path)
		if err != nil {
			return err
		}
		f, err := h.FileSystem.OpenFile(context.Background(), path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(data)
		return err
	})

	return h
}()
