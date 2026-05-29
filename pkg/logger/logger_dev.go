//go:build !prod

package logger

import (
	"log"
)

func Println(v ...interface{}) {
	log.Println(v...)
}

func Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func Info(v ...interface{}) {
	log.Println(append([]interface{}{"[INFO]"}, v...)...)
}

func Error(v ...interface{}) {
	log.Println(append([]interface{}{"[ERROR]"}, v...)...)
}
