package tools

//获取字符串中需要的字符
func GetUsefulEnStr(content string) string {
	words := ([]rune)(content)
	var outStr string
	for _, item := range words{
		if item >= 'a' && item <= 'z' || item >='A'&& item <='Z' || item >='0' && item <='9' || item == '|'  {
			outStr += string(item)
		}
	}
	return outStr
}