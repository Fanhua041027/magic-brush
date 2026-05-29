//go:build prod

package logger

func Println(v ...interface{}) {}

func Printf(format string, v ...interface{}) {}

func Info(v ...interface{}) {}

func Error(v ...interface{}) {}
