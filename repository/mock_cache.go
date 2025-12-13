package repository

type MockCache struct {
	Data map[string]string
}

func NewMockCache() *MockCache {
	return &MockCache{
		Data: make(map[string]string),
	}
}

func (m *MockCache) Get(key string) (string, bool) {
	val, ok := m.Data[key]
	return val, ok
}

func (m *MockCache) Set(key string, value string) error {
	m.Data[key] = value
	return nil
}
