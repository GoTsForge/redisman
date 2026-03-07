package main

import (
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/gotsforge/redisman/cmd/parser"
	"github.com/gotsforge/redisman/cmd/protocol"
	"github.com/gotsforge/redisman/cmd/router"
	"github.com/gotsforge/redisman/cmd/server"
)

func main() {
	r := router.NewRouter()
	s := server.NewServer()

	li, err := net.Listen("tcp", ":8081")

	if err != nil {
		panic(err)
	}

	for {
		conn, liErr := li.Accept()
		if liErr != nil {
			panic(liErr)
		}

		go handleConnection(s, conn, r)
	}
}

func handleConnection(s *server.Server, conn net.Conn, r router.Router) {
	defer conn.Close()
	readBuffer := make([]byte, 1024)
	var buffer []byte

	for {
		n, err := conn.Read(readBuffer)
		buffer = append(buffer, readBuffer[:n]...)

		if err != nil {
			if err == io.EOF {
				break
			}

			fmt.Printf("Error reading data from the client! %v\n", err)
			break
		}

		if n > 0 {
			for {
				command, bytesConsumed, status := parser.TryParse(buffer)

				if status == "incomplete" {
					break
				}

				if status == "error" {
					errorResponse := protocol.NewErrorResponse("ERR protocol error")
					conn.Write(errorResponse.Encode())
					break
				}

				command.Name = strings.ToUpper(command.Name)
				response := r.Route(s, command)

				conn.Write(response.Encode())

				buffer = buffer[bytesConsumed:]
			}
		}
	}
}
