package main

import (
	"fmt"
	"time"
)

var c chan int

func ready(w string, sec int) {
	time.Sleep(time.Duration(sec) * time.Second)
	fmt.Println(fmt.Sprintf("%v Sleep %v", w, sec))
	c <- 1
}

func main() {
	c = make(chan int)
	go ready("test5", 6)
	go ready("test2", 3)
	go ready("test1", 2)
	go ready("test3", 4)
	go ready("test4", 5)

	fmt.Println("Lalalala")

	<-c
	<-c
}
