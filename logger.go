package main

type silentLogger struct{}

func (s *silentLogger) Output(depth int, msg string) error {
	return nil
}
