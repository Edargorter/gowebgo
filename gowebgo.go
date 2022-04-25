package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	fmt.Println("Gowebgo.")
	fmt.Println("Initialisingo...")

	// GET
	resp, err := http.Get("https://edargorter.xyz")
	if err != nil {
		log.Fatalln(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	sb := string(body)
	log.Printf(sb)

	// POST 
	post_body, _ := json.Marshal(map[string]string {
		"name": "Zach",
		"email": "nothing@nothing.nothing",
	})
	response_body := bytes.NewBuffer(post_body)
	resp, err = http.Post("https://edargorter.xyz", "application/json", response_body)

	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatalln(err)
	}
	sb = string(body)
	log.Printf(sb)
}
