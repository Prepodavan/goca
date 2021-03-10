package main

import (
	"flag"
	"fmt"
	"github.com/Prepodavan/goca/internal/delivery"
	"github.com/Prepodavan/goca/internal/delivery/issuer"
	"github.com/Prepodavan/goca/internal/usecases"
	"os"
)

var (
	addr      = flag.String("address", "localhost:8080", "address listen to")
	rootCrt   = flag.String("CA", "assets/ssl/root.crt", "filepath to trusted certificate")
	rootKey   = flag.String("CAkey", "assets/ssl/root.key", "filepath to trusted certificate key")
	debugMode = flag.Bool("debug", false, "debug mode switcher")
	docsPath  = flag.String("docs", "api", "path to dir with api docs")
)

func main() {
	flag.Parse()
	if err := checkFile(*rootKey); err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := checkFile(*rootCrt); err != nil {
		fmt.Println(err.Error())
		return
	}
	uc, err := usecases.NewConsoleUtilWrapper(*rootKey, *rootCrt, usecases.WithDebug(*debugMode))
	if err != nil {
		panic(err)
	}
	issuerUseCase := issuer.NewService(uc)
	server := delivery.NewServer(*docsPath, issuerUseCase)
	fmt.Println(server.Run(*addr))
}

func checkFile(fp string) error {
	_, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Errorf("can't read file with path `%s`: %s", fp, err.Error())
	}
	return nil
}
