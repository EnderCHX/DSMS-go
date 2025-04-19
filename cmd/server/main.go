package server

import (
	"fmt"
	"github.com/EnderCHX/DSMS-go/internal/connect"
	"net"
)

func main() {
	s, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			con, err := s.Accept()
			if err != nil {
				panic(err)
			}
			defer con.Close()
			go func() {
				client := connect.NewConn(con)
				defer client.Close()

				reader, err := client.Receive()
				if err != nil {
					panic(err)
				}
				for {
					buf := make([]byte, 1024)
					n, err := reader.Read(buf)
					if err != nil {
						if err.Error() == "EOF" {
							return
						}
						panic(err)
					}
					fmt.Println(string(buf[:n]))
				}
			}()
		}
	}()

	select {}
}
