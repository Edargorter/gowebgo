package main

import (
	"log"
	"bufio"
	"fmt"
	"os"
)

func main() {
	file, err := os.Open("filetest.txt")
	if err != nil {
		log.Fatalf("failed to open")
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var lines[]string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	file.Close()

	//Print lines
	for _, line := range lines {
		fmt.Println(line)
	}
}
