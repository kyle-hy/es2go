package utils

import "strings"

// FilterOut 从source中踢出exclude的元素
func FilterOut[T comparable](source, exclude []T) []T {
	toExclude := make(map[T]struct{})
	for _, item := range exclude {
		toExclude[item] = struct{}{}
	}

	var result []T
	for _, item := range source {
		if _, found := toExclude[item]; !found {
			result = append(result, item)
		}
	}
	return result
}

// CombineSlices 随机组合两个切片的元素
func CombineSlices[T any](a, b []T) [][]T {
	var result [][]T
	for _, itemA := range a {
		for _, itemB := range b {
			result = append(result, []T{itemA, itemB})
		}
	}
	return result
}

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

// Cartesian 对data中的多个子切片进行两两组合（笛卡尔积）并拼接字符串
func Cartesian(data [][]string) []string {
	if len(data) == 0 {
		return nil
	}

	var result []string
	var backtrack func(index int, path []string)

	backtrack = func(index int, path []string) {
		if index == len(data) {
			result = append(result, strings.Join(path, ""))
			return
		}
		for _, val := range data[index] {
			backtrack(index+1, append(path, val))
		}
	}

	backtrack(0, []string{})
	return result
}
