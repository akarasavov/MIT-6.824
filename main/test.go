package main

import (
	"mapreduce"
	"bytes"
	"strconv"
	"io/ioutil"
	"fmt"
)

func main() {

	mapInts := make(map[string][]int)
	ints := mapInts["a"]
	ints = append(ints, 1)
	ints = append(ints, 2)
	mapInts["a"] = ints

	var buffer bytes.Buffer

	for key, values := range mapInts {
		for _, value := range values {
			n, err := buffer.WriteString(key + ":" + strconv.Itoa(value) + "\n")
			fmt.Print(n)
			fmt.Print(err)
		}
	}
	fmt.Print(len(buffer.Bytes()))
	file := ioutil.WriteFile("/home/akt/go/src/main/lext.txt", buffer.Bytes(), 0644)
	fmt.Print(file)
}

type KeyValue struct {
	Key   string
	Value string
}

func hi(value string) []mapreduce.KeyValue {
	keyValue := mapreduce.KeyValue{Key: value, Value: value}
	values := make([]mapreduce.KeyValue, 1)
	values[0] = keyValue
	return values
}
