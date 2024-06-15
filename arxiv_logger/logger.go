package arxivlogger

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

var f *os.File
var enabled bool

func init() {
	enabled = false
}

func Initialize(debugEnable bool, filename ...string) {
	enabled = debugEnable
	if len(filename) == 0 {
		log.SetOutput(os.Stdout)
	} else {
		var err error
		f, err = os.OpenFile(filename[0], os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Could not open file for logging: %s", filename[0])
			return
		}
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			<-c
			log.Printf("Caught shutdown, closing log file")
			f.Close()
			os.Exit(1)
		}()
		log.SetOutput(f)
	}
}

func Fatalf(s string, v ...any) {
	if enabled {
		log.Printf(s, v...)
	}
	os.Exit(1)
}

func Fatal(v ...any) {
	if enabled {
		log.Print(v...)
	}
	os.Exit(1)
}

func Printf(s string, v ...any) {
	if enabled {
		log.Printf(s, v...)
	}
}

func Print(v ...any) {
	if enabled {
		log.Print(v...)
	}
}
func Println(v ...any) {
	if enabled {
		log.Println(v...)
	}
}
