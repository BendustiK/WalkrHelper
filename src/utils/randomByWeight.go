package utils

var cw int = 0
var gcd int = 2
var i int = 0

func GetRandomDataByWeight(dataMap map[string]int) string {
	for {
		i = (i + 1) % len(dataMap)
		if i == 0 {
			cw = cw - gcd
			if cw <= 0 {
				cw = getMaxWeight(dataMap)
				if cw == 0 {
					return ""
				}
			}
		}

		for id, weight := range dataMap {
			if weight > 0 && weight >= cw {
				return id
			}
		}

	}
}

func getMaxWeight(dataMap map[string]int) int {
	max := 0
	for _, weight := range dataMap {
		if weight >= max {
			max = weight
		}
	}

	return max
}
