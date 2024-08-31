package throttle

type Config struct {
	Enabled bool
	Path    string
	Limit   int
	Hold    string
}
