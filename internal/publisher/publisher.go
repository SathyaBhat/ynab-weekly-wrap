package publisher

// Publisher defines the interface for sending messages to various platforms
type Publisher interface {
	Publish(message string) error
}
