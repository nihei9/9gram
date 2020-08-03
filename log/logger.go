package log

import (
	"fmt"
	"io"
	"os"
)

type logger struct {
	out io.WriteCloser
}

var l *logger

func Init(outputPath string) error {
	f, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	l = &logger{
		out: f,
	}

	return nil
}

func Close() error {
	if l == nil {
		return nil
	}

	return l.out.Close()
}

func GetWriter() io.Writer {
	if l == nil {
		return nil
	}
	return l.out
}

func Log(format string, opts ...interface{}) {
	if l == nil {
		return
	}
	fmt.Fprintf(l.out, format+"\n", opts...)
}
