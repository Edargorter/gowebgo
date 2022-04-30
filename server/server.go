package main

import (
	"os"
	"math"
	"os/exec"
	"flag"
	"fmt"
	"golang.org/x/term"
	"html"
	"io"
	"io/ioutil"
    "net/http"
	"net/http/httputil"
	"strings"
	"strconv"
	"log"
	"time"
	"bufio"
	"bytes"
	"runtime"
)

type RStruct struct {
	req_filename string
	resp_filename string
	recv_time string
	host string
	data bool
}

type CmdStruct struct {
	display string
	function func([]string)
}

type Connection struct {
    Request  *http.Request
    Response *http.Response
}

var port int
var err_str string
var editor string
var v_offset int
var req_names []string
var reqs map[string] RStruct
var intercept bool
var esc map[string] string

//OS commands
var os_cmds map[string] string

//Gowebgo Commands 
var usage_msg string
var cmd_arr []string
var cmd_dict map[string] CmdStruct

// Probably not needed 
func format_request(r *http.Request) string {
	var request []string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	request = append(request, fmt.Sprintf("Host: %v", r.Host))

	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}
	return strings.Join(request, "\n")
}

func ReadHTTPFromFile(r io.Reader) ([]Connection, error) {
    buf := bufio.NewReader(r)
    stream := make([]Connection, 0)

    for {
        req, err := http.ReadRequest(buf)
        if err == io.EOF {
            break
        }
        if err != nil {
            return stream, err
        }

        resp, err := http.ReadResponse(buf, req)
        if err != nil {
            return stream, err
        }

        //save response body
        b := new(bytes.Buffer)
        io.Copy(b, resp.Body)
        resp.Body.Close()
        resp.Body = ioutil.NopCloser(b)

        stream = append(stream, Connection{Request: req, Response: resp})
    }
    return stream, nil
}

// CMD funcs 

func get_req_name(args []string) (string, bool) {
	var req_name string
	if len(args) < 1 {
		return "", false
	}
	if len(args) > 1 && args[0] == "-r" {
		req_name = args[1]
	} else {
		index, err := strconv.Atoi(args[0])
		if err != nil {
			log.Println("\n" + err.Error())
			return "", false
		}
		if index >= 0 && index < len(req_names) {
			req_name = req_names[index]
		}
	}
	return req_name, true
}

func edit_request(args []string) {
	if len(args) < 1 {
		err_str = "Error: fewer args than expected."
	} else {
		req_name, found := get_req_name(args)
		if found {
			if req, ok := reqs[req_name]; ok {
				cmd := exec.Command(editor, "requests/" + req.req_filename)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				if err != nil {
					fmt.Println(err)
				}
			} else {
				err_str = fmt.Sprintf("Error: %s does not exist.", req_name)
			}
		} else {
			err_str = "Error: request does not exist."
		}
	}
}

func delete_request(args []string) {
	if len(args) < 1 {
		err_str = "Error: fewer args than expected."
	} else {
		req_name, found := get_req_name(args)
		if found {
			if req, ok := reqs[req_name]; ok {
				cmd := exec.Command(os_cmds["remove"], "requests/" + req.req_filename)
				err := cmd.Run()
				if err != nil {
					err_str = err.Error()
					return
				}
				delete(reqs, req_name)
			} else {
				err_str = fmt.Sprintf("Error: %s does not exist.", req_name)
			}
		} else {
			err_str = "Error: request does not exist."
		}
	}
}

func rename_request(args []string) {
	if len(args) < 2 {
		err_str = "Error: fewer args than expected."
	} else {
		req_name, found := get_req_name(args)
		new_name := args[len(args) - 1]
		if found {
			if req, ok := reqs[req_name]; ok {
				for i := range req_names {
					if req_names[i] == req_name {
						req_names[i] = new_name
						delete(reqs, req_name)
						reqs[new_name] = req
						err_str = fmt.Sprintf("Renaming to %s", new_name)
						break
					}
				}
			} else {
				err_str = "Error: request does not exist."
			}
		} else {
			err_str = fmt.Sprintf("Error: %s does not exist.", req_name)
		}
	}
}

func send_request(args []string) {
	req_name, found := get_req_name(args)
	if found {
		if req, ok := reqs[req_name]; ok {
			fmt.Println(req.req_filename)
		} else {
			err_str = fmt.Sprintf("Error: %s does not exist.", req_name)
		}
	} else {
		err_str = "Error: request does not exist."
	}
}

func dup_request(args []string) {
	req_name, found := get_req_name(args)
	dup_name := args[len(args) - 1]
	if !found {
		err_str = "Error: request does not exist."
	} else {
		req_names = append(req_names, dup_name)
		reqs[dup_name] = reqs[req_name]
	}
}

func quit(args []string) {
	os.Exit(0)
}

func handle_request(w http.ResponseWriter, req *http.Request) {
	recv_time := time.Now().Format("10:00:00")
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(req.URL.Path))
	req_dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		fmt.Println(err)
	}
	req_filename := fmt.Sprintf("req_%v", len(req_names))
	err = ioutil.WriteFile("requests/" + req_filename, req_dump, 0777)
	if err != nil {
		fmt.Println(err)
	} else {
		req_host := req.Host
		reqs[req_filename] = RStruct{req_filename: req_filename, recv_time: recv_time, host: req_host}
		req_names = append(req_names, req_filename)
	}
	err_str = ""
	display()
	reader := bufio.NewReader(os.Stdin)
	user_inp, _ := reader.ReadString('\n')
	proc_cmd(user_inp)
	display()
}

func get_n_byte_string(c byte, n int) string {
	var nbs bytes.Buffer
	for i := 0; i < n; i++ {
		nbs.WriteByte(c)
	}
	return nbs.String()
}

func proc_cmd(cmd string) {
	split := strings.Split(strings.TrimSuffix(cmd, "\n"), " ")
	if len(split) == 0 {
		log.Print("\nError: fewer args than expected.")
	} else {
		cmd_letter := split[0]
		if c_struct, ok := cmd_dict[cmd_letter]; ok {
			c_struct.function(split[1:])
		} else {
			log.Println("\nInvalid command.")
		}
	}
}

//Clear screen
func cls() {
	cmd := exec.Command(os_cmds["clear"])
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func display() {

	//Get terminal dimensions 
	win_width := 75
	win_height := 200
	if term.IsTerminal(0) {
		width, height, err := term.GetSize(0)
		if err != nil {
			//log.Printf("Using default width %v\n", win_width)
		} else {
			win_width = width
			win_height = height
		}
	}

	req_num := len(req_names) + win_height - win_height
	req_v_dist := 0

	cls()

	// Print latest request 
	if len(req_names) > 0 {
		last_req_file := fmt.Sprintf("requests/%v", reqs[req_names[req_num-1]].req_filename)
		data, err := ioutil.ReadFile(last_req_file)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(data))
		req_v_dist = int(math.Min(math.Abs(float64(req_num - v_offset)), float64(v_offset)))
		req_v_dist = int(math.Min(float64(req_num), float64(v_offset)))
	}

	//Separator 
	fmt.Println(get_n_byte_string('-', win_width))

	// Print previous requests
	fmt.Println("\nName\t\tHost\t\t\tResp\t\tCode\t\tTime\n")
	for i := 0; i < req_v_dist; i++ {
		if i == 0 {
			fmt.Print(esc["bg_yellow"])
			fmt.Print(esc["black"])
		}
		req_name := req_names[req_num - i - 1]
		r := reqs[req_name]
		fmt.Println(req_name + "\t\t" + r.host + "\t\t" + strconv.FormatBool(r.data) + "\t\t" + r.recv_time)
		if i == 0 {
			fmt.Print(esc["reset"])
		}
	}
	for i := 0; i < v_offset - req_v_dist; i++ {
		fmt.Println()
	}

	//Separator 
	fmt.Println("\n" + get_n_byte_string('-', win_width) + "\n")

	//Display options 
	fmt.Println(usage_msg)
	fmt.Println("\n" + err_str + "\n")
	for _, cmd_letter := range cmd_arr {
		fmt.Print(fmt.Sprintf("%s (%v) ", cmd_dict[cmd_letter].display, cmd_letter))
	}
	fmt.Print("\n> ")
}

func flag_init() {
	const (
		usage = "gowebgo [-p | -port]={port number}"
	)
	flag.IntVar(&port, "p", 8081, "port number for proxy")
	flag.StringVar(&editor, "e", "vim", "cli editor of choice")
	flag.Parse()
}

func main() {
	// Initialise 
	reqs = make(map[string] RStruct)

	//Default values
	intercept = false
	v_offset = 17
	port = 8081

	//Colours. See https://www.lihaoyi.com/post/BuildyourownCommandLinewithANSIescapecodes.html
	esc = make(map[string] string)
	esc["reset"] = "\u001b[0m"
	esc["bg_yellow"] = "\u001b[43m"
	esc["bg_blue"] = "\u001b[44m"
	esc["green"] = "\u001b[32m"
	esc["black"] = "\u001b[30m"
	esc["red"] = "\u001b[31m"

	//Cmd array
	cmd_arr = []string{"e", "r", "s", "d", "q"}

	//Cmd struct dict
	cmd_dict = make(map[string] CmdStruct)
	cmd_dict["e"] = CmdStruct{display: "Edit", function: edit_request}
	cmd_dict["r"] = CmdStruct{display: "Rename", function: rename_request}
	cmd_dict["s"] = CmdStruct{display: "Send", function: send_request}
	cmd_dict["d"] = CmdStruct{display: "Delete", function: delete_request}
	cmd_dict["q"] = CmdStruct{display: "Quit", function: quit}

	//Detect OS and set commands 
	os := runtime.GOOS
	os_cmds = make(map[string] string)
	if os == "Windows" {
		os_cmds["clear"] = "cls"
		os_cmds["remove"] = "del"
	} else {
		os_cmds["clear"] = "clear"
		os_cmds["remove"] = "rm"
	}

	//Usage msg
	usage_msg = "Usage: <cmd> <request>"

	start_time := time.Now().Format("10:00:00")
	prog_name := "gowebgo"
	flag_init()

	fmt.Println("Running:", prog_name,
				"\nOS:", os,
				"\n@", start_time,
				"\nPort:", port,
				"\nEditor:", editor)

	display()
    http.HandleFunc("/", handle_request)

    log.Fatal(http.ListenAndServe(":8081", nil))
}
