package communicate

type BundleClientInterface interface {
	ReleaseJob(key string) error
	Close() error
}