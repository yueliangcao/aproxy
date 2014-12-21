package logx

import (
	"fmt"
	"io"
	"log"
)

type Logger struct {
	log.Logger
	out io.Writer
}

func (l *Logger) Output(calldepth int, s string) error {
	if l.out == nil {
		return nil
	}

	fmt.Println("output")

	return l.Output(calldepth, s)
}
