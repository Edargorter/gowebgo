package main

import (
	"os"
	"math"
	"os/exec"
	"flag"
    "fmt"
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

type Connection struct {
    Request  *http.Request
    Response *http.Response
}

type RStruct struct {
	filename string
	host string
	data bool
}

var port int
var editor string
var v_offset int
var reqs []RStruct

//OS commands
var os_cmds map[string] string
var options [5]string
var usage_msg string
var cmd_funcs []func(string)

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

func edit_request(filename string) {
	cmd := exec.Command(editor, "requests/" + filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}

func delete_request(filename string) {
	cmd := exec.Command(os_cmds["remove"], "requests/" + filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}

func rename_request(filename string) {
	fmt.Println(filename)
}

func send_request(filename string) {
	fmt.Println(filename)
}

func exit_prog(filename string) {
	os.Exit(0)
}

func handle_request(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(req.URL.Path))
	req_dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		fmt.Println(err)
	}
	req_filename := fmt.Sprintf("req_%v", len(reqs))
	err = ioutil.WriteFile("requests/" + req_filename, req_dump, 0777)
	if err != nil {
		fmt.Println(err)
	} else {
		req_host := req.Host
		n_rs := RStruct{filename: req_filename, host: req_host}
		reqs = append(reqs, n_rs)
	}
	display()
	reader := bufio.NewReader(os.Stdin)
	user_inp, _ := reader.ReadString('\n')
	read_cmd(user_inp)
}

func get_n_byte_string(c byte, n int) string {
	var nbs bytes.Buffer
	for i := 0; i < n; i++ {
		nbs.WriteByte(c)
	}
	return nbs.String()
}

func read_cmd(cmd string) {
	split := strings.Split(cmd, " ")
	if len(split) == 0 {
		log.Print("\nError: fewer args than expected.")
	} else {
		cmd_index, err := strconv.Atoi(split[0])
		if err != nil {
			log.Println("\nPlease specify an integer")
		} else {
			arg := ""
			if len(split) > 1 {
				arg = split[1]
			}
			if cmd_index < 0 || cmd_index > len(cmd_funcs) {
				log.Println("\nCmd index out of range.")
			} else {
				cmd_funcs[cmd_index](arg)
			}
		}
	}
}

func display(){

	req_num := len(reqs)
	req_v_dist := 0

	//Clear screen
	cmd := exec.Command(os_cmds["clear"])
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Print latest request 
	if len(reqs) > 0 {
		last_req_file := fmt.Sprintf("requests/%v", reqs[req_num-1].filename)
		data, err := ioutil.ReadFile(last_req_file)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("\nRequest:\n")
		fmt.Print(string(data))
		req_v_dist = int(math.Min(math.Abs(float64(req_num - v_offset)), float64(v_offset)))
		req_v_dist = int(math.Min(float64(req_num), float64(v_offset)))
	}
	fmt.Println(get_n_byte_string('-', 50))

	// Print previous requests
	fmt.Println("\nName\t\tHost\t\t\tData\n")
	for i := 0; i < req_v_dist; i++ {
		r := reqs[req_num - i - 1]
		fmt.Println(r.filename + "\t\t" + r.host + "\t\t" + strconv.FormatBool(r.data))
	}
	for i := 0; i < v_offset - req_v_dist; i++ {
		fmt.Println()
	}

	fmt.Println("\n" + get_n_byte_string('-', 50) + "\n")

	//Display options 
	fmt.Println(usage_msg + "\n")
	for index, opt := range options {
		fmt.Println(fmt.Sprintf("%v) %s", index, opt))
	}
	fmt.Print("> ")
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
	// Initialise global variables 
	v_offset = 17
	cmd_funcs = []func(string){edit_request, rename_request, send_request, delete_request, exit_prog}

	os_cmds = make(map[string] string)
	edit_request("req_0")


	//Detect OS and set commands 
	os := runtime.GOOS
	if os == "Windows" {
		os_cmds["clear"] = "cls"
		os_cmds["remove"] = "del"
	} else {
		os_cmds["clear"] = "clear"
		os_cmds["remove"] = "rm"
	}

	//Usage msg
	usage_msg = "Usage: <index> <request>"

	//Set options 
	options[0] = "Edit"
	options[1] = "Rename"
	options[2] = "Send"
	options[3] = "Delete"
	options[4] = "Exit"

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
