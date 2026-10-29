package services

// Logger exposes functionality to write messages in stdout.
type Logger interface {
	Info(msg string)
	// Error is used to send formatted as error message.
	Error(msg string, err error)
	// Debug is used to send formatted debug message.
	Debug(msg string)
	// Warn is used to send formatted warning message.
	Warn(msg string)
}
