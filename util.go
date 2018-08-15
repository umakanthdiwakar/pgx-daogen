package main

import (
	"bufio"
	"fmt"
	"unicode"
)

func convertCase(name string) string {
	b := []byte(name)
	nb := []byte{}
	nb = append(nb, byte(unicode.ToUpper(rune(b[0]))))
	usSeen := false
	for i := 1; i < len(b); i++ {
		if b[i] == '_' {
			usSeen = true
		} else {
			if usSeen {
				nb = append(nb, byte(unicode.ToUpper(rune(b[i]))))
				usSeen = false
			} else {
				nb = append(nb, b[i])
			}
		}
	}
	return string(nb)
}

var _global_writer *bufio.Writer

func pp(args ...interface{}) {
	fmt.Println(args...)
}
func ff(s string, args ...interface{}) {
	fmt.Fprintf(_global_writer, s, args...)
	_global_writer.Flush()
}
