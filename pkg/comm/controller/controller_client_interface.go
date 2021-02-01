package communicate

type ControllerClientInterface interface {
	AddFunc(obj interface{})
	DeleteFunc(obj interface{})
	Close() error
}
