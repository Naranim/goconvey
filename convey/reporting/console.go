package reporting

import (
	"fmt"
	"io"
)

type console struct{}

func (self *console) Write(p []byte) (n int, err error) {
	return fmt.Print(string(p))
}
func (self *console) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}
func NewConsole() io.WriteSeeker {
	return new(console)
}
