package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	// If the file doesn't exist, create it or append to the file
	file, err := os.OpenFile("logs.txt", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}
	read(file)
	return

	// 40byteのレコードを1億レコード
	// 4GBのファイル作成
	log.SetOutput(file)

	counter := 0

	for i := 0; i < 100000000; i++ {
		log.Printf("Hello world\t%v\n", counter)
		counter++
	}
	file.Close()

}

func read(r *os.File) {
	count, _ := r.Seek(0, 2)
	for count > 10 {
		count /= 2
		fmt.Println(count)
		r.Seek(count, 0)
		scanner := bufio.NewScanner(r)
		scanner.Scan()
		fmt.Println(scanner.Text())
		scanner.Scan()
		fmt.Println(scanner.Text())
	}
}
