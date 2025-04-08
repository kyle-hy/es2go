package utils

// Combinations 随机组合数组的元素
func Combinations[T any](items []T, maxCount int) [][]T {
	var all [][]T
	n := len(items)

	for i := 1; i <= maxCount && i <= n; i++ {
		all = append(all, combinationsOf(items, i)...)
	}

	return all
}

func combinationsOf[T any](items []T, k int) [][]T {
	var res [][]T
	var dfs func(start int, path []T)

	dfs = func(start int, path []T) {
		if len(path) == k {
			temp := make([]T, k)
			copy(temp, path)
			res = append(res, temp)
			return
		}
		for i := start; i < len(items); i++ {
			dfs(i+1, append(path, items[i]))
		}
	}

	dfs(0, []T{})
	return res
}
