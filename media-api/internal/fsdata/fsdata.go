package fsdata

import (
	"embed"
)

var (
	//go:embed data
	data embed.FS
)

func GetEmbeddedConfigData() *embed.FS {
	return &data
}
