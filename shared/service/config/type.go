package config

type IService interface {
	GetCapturer() Capturer
	Finalize()
}
