package subtree

import "strings"

func splitTopic(topic string) []string {
	tmp := strings.Split(strings.Trim(topic, "/"), "/")

	// 去除所有的空字符
	for _, v := range tmp {
		if v == "" {
			result := make([]string, 0)
			for _, v := range tmp {
				if v != "" {
					result = append(result, v)
				}
			}
			return result
		}
	}
	return tmp
}
func HasWildcard(topic string) bool {

	for i := 0; i < len(topic); i++ {
		if topic[i] == '#' || topic[i] == '+' {
			return true
		}
	}
	return false

}

// 判断订阅的topic是否有大于0
func subQosMoreThan0(topics map[string]int32) bool {
	for _, v := range topics {
		if v > 0 {
			return true
		}
	}
	return false
}
