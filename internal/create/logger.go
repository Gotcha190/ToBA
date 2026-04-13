package create

import "fmt"

type Logger interface {
    Step(msg string)
	Info(msg string)
	Warning(msg string)
    Success(msg string)
    Error(msg string)
}

type ConsoleLogger struct{}

func (l ConsoleLogger) Step(msg string) {
    fmt.Println("[STEP]", msg)
}
func (l ConsoleLogger) Info(msg string){
	fmt.Println("[INFO]", msg)
}

func (l ConsoleLogger) Success(msg string) {
    fmt.Println("[OK]", msg)
}
func (l ConsoleLogger) Warning(msg string) {
	fmt.Println("[WARNING]", msg)
}

func (l ConsoleLogger) Error(msg string) {
    fmt.Println("[ERROR]", msg)
}