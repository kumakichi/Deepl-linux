package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/atotto/clipboard"
	"github.com/zserge/webview"
)

var (
	width  int
	height int
)

const (
	appName = "deepl-translator-linux"
)

func init() {
	flag.IntVar(&width, "w", 800, "window width")
	flag.IntVar(&height, "h", 600, "window height")
	flag.Parse()
}

func main() {
	tryStart := 0
	address := filepath.Join(os.TempDir(), fmt.Sprintf("%s.sock", appName))
	log.Printf("socket file: <%s>\n", address)

start:
	if tryStart > 1 {
		log.Fatal("tried too many times")
		return
	}

	l, err := net.Listen("unix", address)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			unixAddr, err := net.ResolveUnixAddr("unix", address)
			if err != nil {
				log.Fatal(err)
			}

			conn, err := net.DialUnix("unix", nil, unixAddr)
			if err != nil {
				if errors.Is(err, syscall.ECONNREFUSED) {
					syscall.Unlink(address) // SIGKILL and SIGSTOP may not be caught
					tryStart += 1
					goto start
				}
				log.Fatal(err)
			}
			defer conn.Close()
			return
		}
		log.Fatal(err)
	}
	defer l.Close()
	defer syscall.Unlink(address)

	w := webview.New(true)
	defer w.Destroy()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGABRT,
	) // SIGKILL and SIGSTOP may not be caught

	go func() {
		for {
			sig := <-sc
			log.Printf("got signal %s\n", sig)
			syscall.Unlink(address)
			os.Exit(0)
		}
	}()

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
