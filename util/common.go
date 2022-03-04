package util

import "github.com/pterm/pterm"

func GetVersion() string {
	return "1.0"
}

func Contains(list []string, str string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func Fatal(err error) {
	if err != nil {
		pterm.Fatal.Println(err)
	}
}