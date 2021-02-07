package communicate

type QueueClientInterface interface {
	AddFunc(obj interface{})
	DeleteFunc(obj interface{})
	Close() error
}
