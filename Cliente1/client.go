package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

const BUFFER_SIZE = 5000
const NAME_BUFFER_SIZE = 30
const FILE_SIZE_BUFFER = 10

type Client struct {
	Name       string
	reader     *bufio.Reader
	writer     *bufio.Writer
	connection net.Conn
}

var readMsg bool = true

func cliente(c net.Conn, err error) {
	var nickname string
	var msg string
	var opc int = 1
	in := bufio.NewReader(os.Stdin)
	fmt.Println("Nickname")
	nickname, _ = in.ReadString('\n')
	client := &Client{
		Name:       nickname,
		reader:     bufio.NewReader(c),
		writer:     bufio.NewWriter(c),
		connection: c,
	}
	err = gob.NewEncoder(client.connection).Encode(client)
	if err != nil {
		fmt.Println(err)
		return
	}
	go client.ReadFromServer()
	go client.ReceiveFileFromServer()
	for opc != 0 {
		fmt.Println("1.- Enviar mensaje")
		fmt.Println("2.- Enviar archivo")
		fmt.Scanln(&opc)
		switch opc {
		case 1:
			fmt.Print("Mensaje: ")
			msg, _ = in.ReadString('\n')
			client.WriteToServer(msg)
		case 2:
			fmt.Print("Ruta del archivo: ")
			msg, _ = in.ReadString('\n')
			msg = strings.TrimSuffix(msg, "\r\n")
			client.WriteToServer("file\n")
			client.sendFileToClient(msg)
		}
	}
	c.Close()
}

func (c *Client) WriteToServer(msg string) {
	_, err := c.writer.WriteString(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = c.writer.Flush()
	if err != nil {
		fmt.Println(err)
		return
	}
}

func (c *Client) ReadFromServer() {
	for {
		if readMsg {
			msg, err := c.reader.ReadString('\n')
			if err != nil {
				fmt.Println("ERROR")
				return
			}
			if strings.Contains(msg, "file") {
				readMsg = false
				c.ReceiveFileFromServer()
			} else {
				fmt.Println(msg)
			}
		}
	}
}

func (c *Client) sendFileToClient(path string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println(err)
		return
	}
	fileSize := fillString(strconv.FormatInt(fileInfo.Size(), 10), FILE_SIZE_BUFFER)
	fileName := fillString(fileInfo.Name(), NAME_BUFFER_SIZE)
	c.connection.Write([]byte(fileSize))
	c.connection.Write([]byte(fileName))
	sendBuffer := make([]byte, BUFFER_SIZE)
	for {
		_, err = file.Read(sendBuffer)
		if err == io.EOF {
			break
		}
		c.connection.Write(sendBuffer)
	}
}

func (c *Client) ReceiveFileFromServer() {
	if !readMsg {
		fileName := make([]byte, NAME_BUFFER_SIZE)
		fileSize := make([]byte, FILE_SIZE_BUFFER)
		c.connection.Read(fileSize)
		size, _ := strconv.ParseInt(strings.Trim(string(fileSize), "$"), 10, NAME_BUFFER_SIZE)
		c.connection.Read(fileName)
		name := strings.Trim(string(fileName), "$")
		newFile, err := os.Create(name)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer newFile.Close()
		var received int64
		for {
			if (size - received) < BUFFER_SIZE {
				io.CopyN(newFile, c.connection, (size - received))
				c.connection.Read(make([]byte, (received+BUFFER_SIZE)-size))
				break
			}
			io.CopyN(newFile, c.connection, BUFFER_SIZE)
			received += BUFFER_SIZE
		}
		readMsg = true
	}
}

func fillString(temp string, toLength int) string {
	for {
		lengtString := len(temp)
		if lengtString < toLength {
			temp = temp + "$"
			continue
		}
		break
	}
	return temp
}

func main() {
	c, err := net.Dial("tcp", ":9999")
	if err != nil {
		fmt.Println(err)
		return
	}
	go cliente(c, err)
	for true {
	}
}
