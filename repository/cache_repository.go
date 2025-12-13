package repository

type CacheRepository interface {
	Get(key string) (string, bool)
	Set(key string, value string) error
}
