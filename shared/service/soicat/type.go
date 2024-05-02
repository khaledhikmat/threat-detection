package soicat

type IService interface {
	UpdateCamera(camera Camera) error
	Cameras() ([]Camera, error)
	Finalize()
}
