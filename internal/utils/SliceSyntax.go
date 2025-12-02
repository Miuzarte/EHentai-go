package utils

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type SliceSyntaxes []SliceSyntax

func (sss SliceSyntaxes) String() string {
	if len(sss) == 0 {
		return ""
	}
	res := make([]string, len(sss))
	for i, ss := range sss {
		res[i] = ss.String()
	}
	return strings.Join(res, ",")
}

// ToIndexes 将多个 SliceSyntax 转换为索引
func (sss SliceSyntaxes) ToIndexes(sliceLen int) (indexes []int, err error) {
	if len(sss) == 0 {
		return nil, fmt.Errorf("empty slice syntax")
	}
	for _, ss := range sss {
		var indexs []int
		indexs, err = ss.ToIndexes(sliceLen)
		if err != nil {
			return
		}
		indexes = append(indexes, indexs...)
	}
	return
}

// ToIndexesNoRepeat 将多个 SliceSyntax 转换为索引, 会去重
func (sss SliceSyntaxes) ToIndexesNoRepeat(sliceLen int) (indexes []int, err error) {
	if len(sss) == 0 {
		return nil, fmt.Errorf("empty slice syntax")
	}
	set := Set[int]{}
	for _, ss := range sss {
		var indexs []int
		indexs, err = ss.ToIndexes(sliceLen)
		if err != nil {
			return
		}
		indexes = append(indexes, set.Clean(indexs)...)
	}
	return indexes, nil
}

type SliceSyntax [2]int

func (ss SliceSyntax) String() string {
	if ss[1] == -1 {
		return "[" + strconv.Itoa(ss[0]) + "]"
	} else {
		switch {
		case ss[0] == 0 && ss[1] == 0:
			return "[:]"
		case ss[0] == 0:
			return "[:" + strconv.Itoa(ss[1]) + "]"
		case ss[1] == 0:
			return "[" + strconv.Itoa(ss[0]) + ":]"
		case ss[0] == ss[1]-1: // index
			return "[" + strconv.Itoa(ss[0]) + "]"
		default:
			return "[" + strconv.Itoa(ss[0]) + ":" + strconv.Itoa(ss[1]) + "]"
		}
	}
}

// ToIndexes 将 SliceSyntax 转换为索引列,
//
// 负数索引会被转换为正数索引, 例如 -1 会被转换为 sliceLen-1
func (ss SliceSyntax) ToIndexes(sliceLen int) (indexes []int, err error) {
	// editing copy
	if ss[0] < 0 {
		ss[0] += sliceLen
	}
	if ss[1] <= 0 {
		ss[1] += sliceLen
	}

	if ss[0] < 0 || ss[1] < 0 || ss[0] > sliceLen || ss[1] > sliceLen {
		return nil, fmt.Errorf("slice syntax out of range %d: %s", sliceLen, ss.String())
	}
	indexes = make([]int, ss[1]-ss[0])
	for i := range ss[1] - ss[0] {
		indexes[i] = i + ss[0]
	}
	return
}

// ParseSliceSyntaxes 解析多个切片语法, 不限制分隔符
func ParseSliceSyntaxes(str string) (sss SliceSyntaxes, err error) {
	stack := Stack[int]{} // 存储 '[' 的索引
	for i, char := range []byte(str) {
		switch char {
		case '[':
			stack.Push(i)

		case ']':
			start, ok := stack.Pop()
			if !ok {
				continue
			}

			// 提取包括中括号在内的内容
			var ss SliceSyntax
			ss, err = ParseSliceSyntax(str[start : i+1])
			if err != nil {
				return
			}
			sss = append(sss, ss)

		}
	}
	return sss, nil
}

// ParseSliceSyntax 解析切片语法,
//
// 只支持 [n:m] / [:m] / [n:] / [n] 的格式,
//
// 其中 n, m 可以是负数, 代表从后往前索引,
//
// 传入的字符串需要带上中括号
func ParseSliceSyntax(str string) (ss SliceSyntax, err error) {
	if str[0] != '[' || str[len(str)-1] != ']' ||
		!IsNumber(str[1:len(str)-1], '[', ':', ']') {
		return ss, fmt.Errorf("invalid slice syntax: %s", str)
	}

	if bytes.IndexByte([]byte(str), ':') >= 0 {
		var start, end int

		splits := strings.Split(str, ":")
		// "[n" -> "n", "m]" -> "m"
		s, e := splits[0][1:], splits[1][:len(splits[1])-1]

		if s != "" {
			start, err = strconv.Atoi(s)
			if err != nil {
				panic(err)
			}
		}
		if e != "" {
			end, err = strconv.Atoi(e)
			if err != nil {
				panic(err)
			}
		}

		if start > end {
			start, end = end, start
		}
		return SliceSyntax{start, end}, nil

	} else {
		var index int

		// "[n]" -> "n"
		str = str[1 : len(str)-1]
		index, err = strconv.Atoi(str)
		if err != nil {
			panic(err)
		}
		return SliceSyntax{index, index + 1}, nil
	}
}

// DoIndexes is possible to panic
func DoIndexes[T any](slice []T, indexes []int) []T {
	if len(indexes) == 0 {
		return nil
	}
	res := make([]T, len(indexes))
	for i, index := range indexes {
		res[i] = slice[index]
	}
	return res
}
