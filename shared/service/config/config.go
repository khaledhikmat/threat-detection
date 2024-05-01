package config

import (
	"embed"
	"encoding/json"
	"fmt"
)

type configService struct {
	Capturer Capturer  `json:"capturer"`
	FsData   *embed.FS `json:"-"`
}

func New(fs *embed.FS) IService {
	p := configService{
		FsData: fs,
	}

	err := json.Unmarshal(read(p.FsData, fmt.Sprintf("data/%s.json", "dev")), &p)
	if err != nil {
		panic(err)
	}

	return &p
}

func (s *configService) GetCapturer() Capturer {
	return s.Capturer
}

func (s *configService) Finalize() {
}

func read(fs *embed.FS, file string) []byte {
	fd, err := fs.ReadFile(file)
	if err != nil {
		fd, _ = fs.ReadFile("data/dev.json")
	}

	return fd
}
