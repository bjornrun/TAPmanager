/*
TAPmanager

Copyright (c) 2013 Bjorn Runaker

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package main

import (
    "fmt"
    "net/http"
	"flag"
	"os"
	"os/exec"
	"github.com/stvp/go-toml-config"
)

var (
	tapname            = config.String("tapname", "tap")
	numtap           = config.Int("numtap", 1)
	starttap           = config.Int("starttap", 0)
	startport		= config.Int("startport", 50025)
	startip     = config.String("startip", "10.0.1.136")
	stepip           = config.Int("stepip", 4)
	tapdaemon  = config.String("tapdaemon", "./tapdaemon")
	listen  = config.String("listen", "localhost:18080")

)

const maxTap=256
var cfgFile string
var verbose bool
var cmds [256]*exec.Cmd
var allocNames [256]string
var tapNames	[256]string
var ipAddr		[256]string
var port2tap	[256]int

var Usage = func() {
    fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
    flag.PrintDefaults()
	fmt.Fprintf(os.Stderr,"\nWeb commands:\nhttp://%s/allocate/<signum>_<instance> - allocate a free port -> assigned IP address\n",*listen)
	fmt.Fprintf(os.Stderr,"http://%s/remove/<signum>_<instance> - remove an allocated port\n",*listen)
	fmt.Fprintf(os.Stderr,"http://%s/port/<signum>_<instance> Show port\n",*listen)
	fmt.Fprintf(os.Stderr,"http://%s/ip/<signum>_<instance> Show IP address\n",*listen)
	fmt.Fprintf(os.Stderr,"http://%s/list - list allocated ports\n",*listen)
	fmt.Fprintf(os.Stderr,"Example of tapmanager.cfg:\ntapname=\"tap\"\nnumtap=1\nstarttap=0\nstartip=\"10.1.1.4\"\nstepip=4\ntapdaemon=\"./tapdaemon\"\nlisten=\"127.0.0.1:18080\"\n")
}

func allocateHandler(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Path[len("/allocate/"):]
	fmt.Printf("alloc name = %s\n", name)
	for i, line := range allocNames {
		if (line == name) {
	   	 fmt.Fprintf(w, "Tap:%s Ip:%s , %s Found at %d == %s", tapNames[i], ipAddr[i], name, i, line)
		return
			
		}
  	}
	for i, line := range allocNames {
		if (line == "") {
			if (i >= *numtap) {
				fmt.Fprintf(w, "<h1>Full</h1>")
				return
			} else
			{
			    fmt.Fprintf(w, "<h1>Allocating %s</h1>", name)
				allocNames[i] = name
				cmds[i] = exec.Command(*tapdaemon, tapNames[i], fmt.Sprintf("%d", port2tap[i]))
				cmds[i].Start()
				return				
			}
		}
  	}
	fmt.Fprintf(w, "<h1>Full</h1>")
}

func removeHandler(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Path[len("/remove/"):]
		fmt.Printf("remove name = %s\n", name)

	for i, line := range allocNames {
		if (line == name) {
	   	 fmt.Fprintf(w, "Removed Tap:%s Ip:%s , %s Found at %d == %s", tapNames[i], ipAddr[i], name, i, line)
			allocNames[i] = ""
			fmt.Printf("removed\n")
			if (cmds[i] != nil) {
				cmds[i].Process.Kill()
				cmds[i].Wait()
				cmds[i] = nil
			}
		return
			
		}
		
  	}
    fmt.Fprintf(w, "<h1>not found %s</h1>", name)
}

func portHandler(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Path[len("/port/"):]
		fmt.Printf("port name = %s\n", name)
		for i, line := range allocNames {
		if (line == name) {
			fmt.Fprintf(w,"%d\n",port2tap[i])
			return
		}
		}
	fmt.Fprintf(w,"Not found")
}

func ipHandler(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Path[len("/ip/"):]
		fmt.Printf("ip name = %s\n", name)
		for i, line := range allocNames {
		if (line == name) {
			fmt.Fprintf(w,"%s\n",ipAddr[i])
			return
		}
		}
	fmt.Fprintf(w,"Not found")

}

func listHandler(w http.ResponseWriter, r *http.Request) {
	for i, line := range allocNames {
		if (line != "") {
		fmt.Fprintf(w,"%s: port:%d ip:%s   <br/>\n", line, port2tap[i], ipAddr[i])	
		}
		
	}
}


func main() {
	flag.StringVar(&cfgFile, "c", "tapmanager.cfg", "TAPmanager config setup file")
	flag.BoolVar(&verbose,"v", false, "Verbose")
	
	flag.Usage = Usage
    flag.Parse()
	
	if err := config.Parse(cfgFile); err != nil {
		panic(err)
	}
	
	if  verbose {
		fmt.Printf("TAPmanager\n")
	}
	
	var ip [4]int
	_, err := fmt.Sscanf(*startip, "%d.%d.%d.%d", &ip[0], &ip[1], &ip[2], &ip[3])
    if err != nil {
        panic(err)
    }
	
	for i := 0; i < maxTap; i++ {
		tapNames[i] = fmt.Sprintf("%s%1d",*tapname,*starttap+i)
		ipAddr[i] = fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3]+i*(*stepip))
		port2tap[i] = *startport + i
	}
	
	http.HandleFunc("/allocate/", allocateHandler)
    http.HandleFunc("/remove/", removeHandler)
    http.HandleFunc("/list/", listHandler)
    http.HandleFunc("/ip/", ipHandler)
    http.HandleFunc("/port/", portHandler)
    http.ListenAndServe(*listen, nil)
}
