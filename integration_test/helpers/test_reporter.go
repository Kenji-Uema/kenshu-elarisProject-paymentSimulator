package helpers

type TestReporter interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}
