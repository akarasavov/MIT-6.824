package main

import (
	"fmt"
	"mapreduce"
)

func main() {
	hi := hi("test")
	fmt.Print(hi[0])
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
