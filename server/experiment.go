package main

import (
	"os"
	"os/exec"
	"log"
	"time"
	"fmt"
	"golang.org/x/term"
)

var cmd_buf = make([]byte, 1)
var cmd_str = ""
var old_state *term.State
var dis = "one line\nnext line\nthird line \t\t\t after three tabs\n"
var counter = 0

//Clear screen
func cls() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func display() {
	for ;; {
		cls()
		fmt.Print(dis)
		fmt.Printf("Prompt %d > ", counter)
		fmt.Print(cmd_str)
		time.NewTimer(2 * time.Second)
		counter++

		//Raw TERM 
		old_state, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		_, err = os.Stdin.Read(cmd_buf)

		//Enter 
		if cmd_buf[0] == 13 {
			cmd_str = ""
		} else {
			cmd_str += string(cmd_buf[0])
		}

		//Restore to old state 
		term.Restore(int(os.Stdin.Fd()), old_state)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func main() {
	display()
}
