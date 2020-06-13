package log

type Interface interface {
	Enabled(level Level) bool
	WithName(name string) Interface
	WithFields(fields ...Field) Interface
	V(msg string, fields ...Field)
	D(msg string, fields ...Field)
	I(msg string, fields ...Field)
	E(msg string, fields ...Field)
	Flush() error
}
