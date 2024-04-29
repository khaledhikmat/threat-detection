package config

func New() IService {
	return &config{}
}

type config struct {
}

func (s *config) Finalize() {
}
