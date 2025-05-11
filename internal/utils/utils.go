package utils

import "fmt"

func HyperlinkFile(path string, show ...string) string {
	s := path
	if len(show) != 0 {
		s = show[0]
	}
	return fmt.Sprintf("\x1b]8;;file://%s\x1b\\%s\x1b]8;;\x1b\\", path, s)
}

func Hyperlink(link string, show ...string) string {
	s := link
	if len(show) != 0 {
		s = show[0]
	}
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", link, s)
}

func IsNumber(s string, additional ...byte) bool {
	if len(s) == 0 {
		return false
	}

	testMap := [256]bool{
		'+': true, '-': true,
		'0': true, '1': true, '2': true, '3': true, '4': true,
		'5': true, '6': true, '7': true, '8': true, '9': true,
	}
	for _, a := range additional {
		testMap[a] = true
	}

	for _, b := range []byte(s) {
		if !testMap[b] {
			return false
		}
	}
	return true
}

func WaitAck(msg string) bool {
	fmt.Print(msg + " (y/n): ")
	var input string
	_, _ = fmt.Scanln(&input)
	if input == "y" || input == "Y" {
		return true
	}
	return false
}
