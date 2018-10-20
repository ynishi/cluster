package main

import (
	"fmt"
	"github.com/ynishi/cluster"
)

func main() {
	fmt.Println("main")
	service := cluster.NewDefaultClusterService("0.0.0", nil)
	fmt.Println(service.Version())
}
