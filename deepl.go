package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/zserge/webview"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	width  int
	height int
)

const (
	appName  = "deepl-translator-linux"
	selector = "document.getElementsByClassName('lmt__source_textarea')[0]"
)

func init() {
	flag.IntVar(&width, "w", 800, "window width")
	flag.IntVar(&height, "h", 600, "window height")
	flag.Parse()
}

func main() {
	address := filepath.Join(os.TempDir(), fmt.Sprintf("%s.sock", appName))
	log.Printf("socket file: <%s>\n", address)

	l := translateListener(address)
	defer l.Close()
	defer syscall.Unlink(address)

	go signalHandle(address)

	w := webview.New(true)
	defer w.Destroy()

	go translateWorker(w, l)

	w.SetTitle("Deepl-linux")
	w.SetSize(width, height, webview.HintNone)
	w.Navigate("https://www.deepl.com/translator")
	w.Eval(inputText)

	go startupHandler(w) // TODO: better way to process this, like DOMContentLoaded

	w.Run()
}

func startupHandler(w webview.WebView) {
	for i := 0; i < 8; i++ {
		time.Sleep(time.Second * 2)
		cbContent, err := getClipboard()
		if err != nil || strings.TrimSpace(cbContent) == "" {
			log.Printf("[err] clipboard readall fail: %+v\n", err)
			continue
		}
		w.Dispatch(func() {
			w.Eval(`processStartup(` + selector + `,"` + cbContent + `")`)
		})
	}
}

var inputText = `const inputValue = function(dom, st) {
    var evt = new InputEvent('input', {
        inputType: 'insertText',
        data: st,
        dataTransfer: null,
        isComposing: false
    });
    dom.value = st;
    dom.dispatchEvent(evt);
};

const processStartup = function(dom, msg) {
    if (dom.value != "") {
        return;
    }

    inputValue(dom, msg);
};`

func getClipboard() (string, error) {
	clipboardContent, err := clipboard.ReadAll()
	if err != nil {
		return "", err
	}
	log.Printf("got clipboard text: [%s]\n", clipboardContent)

	cbContent := strings.Replace(clipboardContent, "\n", "\\n", -1)
	cbContent = strings.Replace(cbContent, "\"", "\\\"", -1)
	return cbContent, nil
}

func translateWorker(w webview.WebView, l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("accept error %+v", err)
		}
		conn.Close()

		cbContent, err := getClipboard()
		if err != nil {
			log.Printf("[err] clipboard readall fail: %+v\n", err)
			continue
		}

		w.Dispatch(func() {
			w.Eval(`inputValue(` + selector + `,"` + cbContent + `")`)
		})
	}
}

func translateListener(address string) net.Listener {
	tryStart := 0

start:
	if tryStart > 1 {
		log.Fatal("tried too many times")
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
			return nil
		}
		log.Fatal(err)
	}

	return l
}

func signalHandle(address string) {
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
}
