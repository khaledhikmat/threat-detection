package config

type IService interface {
	IsDapr() bool
	IsDiagrid() bool
	GetCapturer() Capturer
	Finalize()
}
