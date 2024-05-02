package capturer

type IService interface {
	Capturers() ([]Capturer, error)
	Finalize()
}
