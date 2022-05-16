package main

import (
	"os"
	"os/signal"
	"math"
	"os/exec"
	"flag"
	"fmt"
	//"io"
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

	"golang.org/x/term"
	"github.com/go-httpproxy/httpproxy"
)

//Types and structs
type RStruct struct {
	req_filename string
	resp_filename string
	recv_time string
	host string
	data bool
}

type CmdStruct struct {
	display string
	function func([]string)bool
}

const (
	project_name = "Gowebgo"
	input_buffer_length = 32
	reqs_folder = "requests/"
	gowebgo_usage = "gowebgo [-p ={port number}] [-i true | false]"
)

//Default values
var intercept = false
var v_offset = 17
var last_req_v = 17
var username string
var password string
var cert_file string
var key_file string

//Flags 
var editor string
var port int

//Display 
var req_names []string
var reqs = make(map[string] RStruct)
var err_str = ""
var win_width = 75
var win_height = 200

//Colours. See https://www.lihaoyi.com/post/BuildyourownCommandLinewithANSIescapecodes.html
var esc = map[string]string{"reset" : "\u001b[0m",
							"bg_yellow" : "\u001b[43m",
							"bg_blue" : "\u001b[44m",
							"bg_white" : "\u001b[47;1m",
							"green" : "\u001b[32m",
							"black" : "\u001b[30m",
							"red" : "\u001b[31m"}

//OS commands
var os_cmds = make(map[string] string)
//Cmd input buffer
var cmd_buf = make([]byte, 1)
var old_state *term.State

//Gowebgo Commands 
var usage_msg = "Usage: <cmd> [-r req_id | request]"
var cmd_arr = []string{"e", "r", "s", "d", "q"}
var cmd_history []string
var cmd_str string
var cmd_dict = map[string]CmdStruct{"e" : CmdStruct{display: "Edit", function: edit_request},
									"r" : CmdStruct{display: "Rename", function: rename_request},
									"s" : CmdStruct{display: "Send", function: send_request},
									"d" : CmdStruct{display: "Delete", function: delete_request},
									"q" : CmdStruct{display: "Quit", function: quit}}

/* HTTPProxy funcs for github.com/go-httpproxy/httpproxy */

func OnError(ctx *httpproxy.Context, where string, err *httpproxy.Error, opErr error) {
	// Log errors.
	log.Printf("ERR: %s: %s [%s]", where, err, opErr)
}

func OnAccept(ctx *httpproxy.Context, w http.ResponseWriter, r *http.Request) bool {
	// Handle local request has path "/info"
	if r.Method == "GET" && !r.URL.IsAbs() {
		if r.URL.Path == "/info" {
			w.Write([]byte("This is Gowebgo, operating with go-httpproxy."))
			return true
		} else if r.URL.Path == "/cert" {
			file_bytes, err := ioutil.ReadFile(cert_file)
			if err != nil {
				panic(err)
			}
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", "attachment; filename=ca_cert.pem")
			w.Write(file_bytes)
		}
	}
	return false
}

func OnAuth(ctx *httpproxy.Context, authType string, user string, pass string) bool {
	// Authenticate user 
	if user == username && pass == password {
		return true
	}
	return false
}

func OnConnect(ctx *httpproxy.Context, host string) (ConnectAction httpproxy.ConnectAction, newHost string) {
	// Apply "Man in the Middle" to all ssl connections. Never change host.
	return httpproxy.ConnectMitm, host
}

func OnRequest(ctx *httpproxy.Context, req *http.Request) (resp *http.Response) {
	// Log proxying requests.
	//log.Printf("INFO: Proxy: %s %s %d", req.Method, req.URL.String(), ctx.Prx.SessionNo)
	//log.Printf("SESSION NO: %d CONTEXT NO %d", ctx.Prx.SessionNo, ctx.SubSessionNo)
	recv_time := time.Now().Format("15:04:05")
	//fmt.Fprintf(w, "Hello, %q", html.EscapeString(req.URL.Path))
	req_dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		err_str = err.Error()
	}
	req_filename := fmt.Sprintf("req_%v", len(req_names))
	err = ioutil.WriteFile(reqs_folder + req_filename, req_dump, 0777)
	if err != nil {
		err_str = err.Error()
	} else {
		req_host := req.Host
		reqs[req_filename] = RStruct{req_filename: req_filename, recv_time: recv_time, host: req_host}
		req_names = append(req_names, req_filename)
	}
	display()
	return
}

func OnResponse(ctx *httpproxy.Context, req *http.Request, resp *http.Response) {
	// Add header "Via: go-httpproxy".
	resp.Header.Add("Via", "go-httpproxy")
	/*
	body, _ := io.ReadAll(resp.Body)
	log.Printf("RESP %s", string(body[:]))
	*/
}

/* End of HTTPProxy funcs */

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

func read_request_from_file(req RStruct) (*http.Request, bool) {
	req_file, err := os.Open(reqs_folder + req.req_filename)
	if err != nil {
		err_str = err.Error()
		return nil, false
	}
	http_req, err := http.ReadRequest(bufio.NewReader(req_file))
	if err != nil {
		err_str = err.Error()
		return nil, false
	}
	fmt.Printf("%T\n", req)
	return http_req, true
}

/*
func read_http_from_file(r io.Reader, req_filename string) (Connection, error) {
    buf := bufio.NewReader(r)
    var stream Connection

	req, err := http.ReadRequest(buf)
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

	stream = Connection{Request: req, Response: resp}

    return stream, nil
}
*/

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

func edit_request(args []string) bool {
	if len(args) < 1 {
		err_str = "Error: fewer args than expected."
	} else {
		req_name, found := get_req_name(args)
		if found {
			if req, ok := reqs[req_name]; ok {
				cmd := exec.Command(editor, reqs_folder + req.req_filename)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				if err != nil {
					fmt.Println(err)
					err_str = err.Error()
					return false
				}
			} else {
				err_str = fmt.Sprintf("Error: %s does not exist.", req_name)
				return false
			}
		} else {
			err_str = "Error: request does not exist."
			return false
		}
	}
	return true
}

func delete_request(args []string) bool {
	if len(args) < 1 {
		err_str = "Error: fewer args than expected."
	} else {
		req_name, found := get_req_name(args)
		if found {
			if req, ok := reqs[req_name]; ok {
				cmd := exec.Command(os_cmds["remove"], reqs_folder + req.req_filename)
				err := cmd.Run()
				if err != nil {
					err_str = err.Error()
					return false
				}
				delete(reqs, req_name)
			} else {
				err_str = fmt.Sprintf("Error: %s does not exist.", req_name)
				return false
			}
		} else {
			err_str = "Error: request does not exist."
			return false
		}
	}
	return true
}

func rename_request(args []string) bool {
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
						break
					}
				}
			} else {
				err_str = "Error: request does not exist."
				return false
			}
		} else {
			err_str = fmt.Sprintf("Error: %s does not exist.", req_name)
			return false
		}
	}
	return true
}

func send_request(args []string) bool {
	req_name, found := get_req_name(args)
	if found {
		if req, ok := reqs[req_name]; ok {
			r, ok := read_request_from_file(req)
			if !ok {
				err_str = fmt.Sprintf("Error: error reading request %s.", req_name)
				return false
			}
			fmt.Print(r)
			/*
			client := &http.Client{}
			resp, err := client.Do(r)
			if err != nil {
				err_str = fmt.Sprintf("Error: error sending request %s.", req_name)
				return false
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				err_str = fmt.Sprintf("Error: error reading response to %s.", req_name)
				return false
			}
			*/
			fmt.Print("\r\n")
		} else {
			err_str = fmt.Sprintf("Error: %s does not exist.", req_name)
			return false
		}
	} else {
		err_str = "Error: request does not exist."
		return false
	}
	return true
}

func dup_request(args []string) bool {
	req_name, found := get_req_name(args)
	dup_name := args[len(args) - 1]
	if !found {
		err_str = "Error: request does not exist."
		return false
	} else {
		req_names = append(req_names, dup_name)
		reqs[dup_name] = reqs[req_name]
	}
	return true
}

func quit(args []string) bool {
	term.Restore(int(os.Stdin.Fd()), old_state)
	os.Exit(0)
	return true
}

func handle_request(w http.ResponseWriter, req *http.Request) {
	recv_time := time.Now().Format("15:04:05")
	//fmt.Fprintf(w, "Hello, %q", html.EscapeString(req.URL.Path))
	req_dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		fmt.Println(err)
	}
	req_filename := fmt.Sprintf("req_%v", len(req_names))
	err = ioutil.WriteFile(reqs_folder + req_filename, req_dump, 0777)
	if err != nil {
		fmt.Println(err)
	} else {
		req_host := req.Host
		reqs[req_filename] = RStruct{req_filename: req_filename, recv_time: recv_time, host: req_host}
		req_names = append(req_names, req_filename)
	}
	display()
	//TODO handle intercept 
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
			cmd_history = append(cmd_history, cmd)
		} else {
			err_str = "Invalid command."
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

func read_request_file(req RStruct) []string {
	req_file := fmt.Sprintf(reqs_folder + "%v", req.req_filename)
	file, err := os.Open(req_file)
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
	return lines
}

func display() {

	//Get terminal dimensions 
	width, height, err := term.GetSize(0)
	if err != nil {
		//log.Printf("Using default width %v\n", win_width)
	} else {
		win_width = width
		win_height = height
	}

	req_num := len(req_names) + win_height - win_height
	req_v_dist := 0

	cls()
	var inter_str string
	if intercept {
		inter_str = fmt.Sprintf("%sON", esc["red"])
	} else {
		inter_str = fmt.Sprintf("%sOFF", esc["green"])
	}

	//Indicate status 
	fmt.Printf("%s %s -Gowebgo-  Port: %d | Editor: %s | Intercept: %s %s\r\n", esc["bg_white"], esc["black"], port, editor, inter_str, esc["reset"])

	var disp int = last_req_v

	// Print latest request 
	if req_num > 0 {
		req_lines := read_request_file(reqs[req_names[req_num-1]])
		for _, line := range req_lines {
			if len(line) > 0 && (line[len(line)-1] == 0x0D || line[len(line)-1] == 0x0A) {
				log.Fatalf("return/newline feed detected")
			}
			fmt.Printf("%s\r\n", line)
			disp--
			if disp == 0 {
				break
			}
		}
	}
	//Remaining offset
	for i := 0; i < disp; i++ {
		fmt.Print("\r\n")
	}

	//Set vertical offset for previous requests 
	req_v_dist = int(math.Min(float64(req_num), float64(v_offset)))

	//Separator 
	fmt.Print(get_n_byte_string('-', win_width) + "\r\n\r\n")

	// Print previous requests
	fmt.Print("ID\t\tName\t\tHost\t\t\tResp\t\tCode\t\tTime\r\n\r\n")

	var req_id int
	var req_name string

	for i := 0; i < req_v_dist; i++ {
		if i == 0 {
			fmt.Print(esc["bg_yellow"])
			fmt.Print(esc["black"])
		}
		req_id = req_num -i - 1
		req_name = req_names[req_id]
		r := reqs[req_name]
		fmt.Print(strconv.Itoa(req_id) + "\t\t" + req_name + "\t\t" + r.host + "\t\t" + strconv.FormatBool(r.data) + "\t\t" + "200" + "\t\t" + r.recv_time + "\r\n")
		if i == 0 {
			fmt.Print(esc["reset"])
		}
	}
	for i := 0; i < v_offset - req_v_dist; i++ {
		fmt.Print("\r\n")
	}

	//Separator 
	fmt.Print("\r\n" + get_n_byte_string('-', win_width) + "\r\n\r\n")

	//Display options 
	fmt.Print(usage_msg + "\r\n")
	for _, cmd_letter := range cmd_arr {
		fmt.Print(fmt.Sprintf("%s (%v) ", cmd_dict[cmd_letter].display, cmd_letter))
	}
	fmt.Print("\r\n" + err_str + "\r\n> " + string(cmd_str))
}

func read_stdin() {

	for {
		//Read one byte 
		_, err := os.Stdin.Read(cmd_buf)
		if err != nil {
			fmt.Println(err)
			return
		}
		c := cmd_buf[0]
		switch c {

		//If "enter", then process command and set cmd_str to nothing 
		case 13:
			proc_cmd(cmd_str)
			cmd_str = ""
			display()

		//Backspace character
		case 0x7f:
			if len(cmd_str) > 0 {
				cmd_str = cmd_str[:len(cmd_str) - 1]
				fmt.Print("\b\033[K")
			}

		//SIGINT -> quit
		case 3:
			quit(make([]string, 0))

		//Otherwise, add c to cmd string 
		default:
			char := string(cmd_buf[0])
			//Print to stdout 
			fmt.Print(char)
			cmd_str += char
		}
	}
}

func main() {
	//Detect OS and set commands 
	host_os := runtime.GOOS

	if host_os == "Windows" {
		os_cmds["clear"] = "cls"
		os_cmds["remove"] = "del"
	} else {
		os_cmds["clear"] = "clear"
		os_cmds["remove"] = "rm"
	}

	//Get terminal dimensions 
	if term.IsTerminal(0) {
		width, height, err := term.GetSize(0)
		if err != nil {
			//log.Printf("Using default width %v\n", win_width)
		} else {
			win_width = width
			win_height = height
		}
	}

	start_time := time.Now().Format("15:04:05")

	flag.IntVar(&port, "p", 8081, "port number for proxy")
	flag.StringVar(&editor, "e", "vim", "cli editor of choice")
	flag.StringVar(&username, "U", "user", "auth: username")
	flag.StringVar(&password, "P", "pass", "auth: password")
	flag.StringVar(&cert_file, "pub", "gowebgo_cert.pem", "Public key (CA cert)")
	flag.StringVar(&key_file, "priv", "gowebgo_key.pem", "Private key")
	flag.BoolVar(&intercept, "i", false, "intercept requests")
	flag.Parse()

	//Display settings 
	fmt.Println("Running:", project_name,
				"\nOS:", host_os,
				"\n@", start_time,
				"\nPort:", port,
				"\nEditor:", editor)

	prev_state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf(err.Error())
	}
	old_state = prev_state
	//Switch back to old state 

	//Start Stdin goroutine
	go read_stdin()

	//Start Signal Interrupt Handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(){
		for sig := range c {
			// sig is a ^C, handle it
			fmt.Print(sig)
		}
	}()

	// Create a new proxy with default certificate pair.
	prx, _ := httpproxy.NewProxy()

	// Set handlers.
	prx.OnError = OnError
	prx.OnAccept = OnAccept
	prx.OnAuth = OnAuth
	prx.OnConnect = OnConnect
	prx.OnRequest = OnRequest
	prx.OnResponse = OnResponse

	// Listen...
	http.ListenAndServe(":8081", prx)
}
