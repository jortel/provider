package main

import (
	"context"
	"fmt"
	"github.com/jortel/provider/pkg/vmware"
	"os"
)

var credentials = vmware.Credentials{
	Host:     os.Getenv("HOST"),
	User:     os.Getenv("USER"),
	Password: os.Getenv("PASSWORD"),
}

func main() {
	p := vmware.Provider{
		Credentials: credentials,
	}

	fmt.Println(p.List())
	fmt.Println(p.Watch(context.TODO()))
}
