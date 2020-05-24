package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"syscall"

	"github.com/atotto/clipboard"
	"github.com/zserge/webview"
)

var (
	width  int
	height int
	port   int
)

func init() {
	flag.IntVar(&width, "w", 800, "window width")
	flag.IntVar(&height, "h", 600, "window height")
	flag.IntVar(&port, "p", 9331, "listen port(for single instance)")
	flag.Parse()
}

func main() {
	address := fmt.Sprintf(":%d", port)

	l, err := net.Listen("tcp4", address)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			tcpAddr, err := net.ResolveTCPAddr("tcp", address)
			if err != nil {
				log.Fatal(err)
			}

			conn, err := net.DialTCP("tcp4", nil, tcpAddr)
			if err != nil {
				log.Fatal(err)
			}
			defer conn.Close()
			return
		}
		log.Fatal(err)
	}
	defer l.Close()

	w := webview.New(true)
	defer w.Destroy()

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Printf("accept error %+v", err)
			}
			conn.Close()

			clipboardContent, err := clipboard.ReadAll()
			if err != nil {
				log.Printf("[err] clipboard readall fail: %+v\n", err)
				continue
			}
			log.Printf("got clipboard text: [%s]\n", clipboardContent)

			cbContent := strings.Replace(clipboardContent, "\n", "\\n", -1)
			cbContent = strings.Replace(cbContent, "\"", "\\\"", -1)

			w.Dispatch(func() {
				w.Eval(`inputValue(document.getElementsByClassName('lmt__source_textarea')[0],"` + cbContent + `")`)
			})
		}
	}()

	w.SetTitle("Deepl-linux")
	w.SetSize(width, height, webview.HintNone)
	w.Navigate("https://www.deepl.com/translator")
	w.Eval(js)

	w.Run()
}

var js = `const inputValue = function (dom, st) {
  var evt = new InputEvent('input', {
    inputType: 'insertText',
    data: st,
    dataTransfer: null,
    isComposing: false
  });
  dom.value = st;
  dom.dispatchEvent(evt);
}`
