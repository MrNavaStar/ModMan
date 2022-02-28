package util

func Contains(list []string, str string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func Remove(list []interface{}, i int) []interface{} {
    list[i] = list[len(list)-1]
    return list[:len(list)-1]
}