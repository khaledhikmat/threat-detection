package soicat

type IService interface {
	UncapturedCameras() ([]Camera, error)
	Finalize()
}
