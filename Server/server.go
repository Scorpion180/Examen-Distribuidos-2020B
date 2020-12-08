package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const BUFFER_SIZE = 5000
const NAME_BUFFER_SIZE = 30
const FILE_SIZE_BUFFER = 10

var readMSg bool = true
var filePath string

type Client struct {
	Name       string
	Reader     *bufio.Reader
	Writer     *bufio.Writer
	Connection net.Conn
}
type File struct {
	Name    string
	Content []byte
}

var clients []Client

func WriteLog(log chan string, send chan string) {
	file, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	for {
		msg := <-log
		file.WriteString(msg + "\n")
		send <- strings.Split(msg, ":")[0]
	}
}

func SendToClients(send chan string) {
	for {
		u := <-send
		content, err := ioutil.ReadFile("log.txt")
		if err != nil {
			fmt.Println(err)
		}
		for _, user := range clients {
			if user.Name != u {
				_, err := user.Writer.WriteString(string(content))
				if err != nil {
					fmt.Println(err)
				}
				err = user.Writer.Flush()
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}
}

func SendFileToClients(sendFile chan string) {
	for {
		u := <-sendFile
		u = strings.Split(u, ":")[0]
		for _, user := range clients {
			if user.Name != u {

				_, err := user.Writer.WriteString("file\n")
				if err != nil {
					fmt.Println(err)
				}

				time.Sleep(250 * time.Millisecond)
				err = user.Writer.Flush()
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println("Mandando archivo " + filePath + " a: " + user.Name)
				file, err := os.Open(filePath)
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
				user.Connection.Write([]byte(fileSize))
				user.Connection.Write([]byte(fileName))
				sendBuffer := make([]byte, BUFFER_SIZE)
				for {
					_, err = file.Read(sendBuffer)
					if err == io.EOF {
						break
					}
					user.Connection.Write(sendBuffer)
				}
			}
		}
	}
}

func servidor() {
	log := make(chan string)
	send := make(chan string)
	sendFile := make(chan string)
	s, err := net.Listen("tcp", ":9999")
	if err != nil {
		fmt.Println(err)
		return
	}
	go WriteLog(log, send)
	go SendToClients(send)
	go SendFileToClients(sendFile)
	for {
		c, err := s.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go addClient(c, log, sendFile)
	}
}

func addClient(connection net.Conn, log chan string, sendFile chan string) {
	client := Client{}
	E := gob.NewDecoder(connection).Decode(&client)
	if E != nil {
		fmt.Println(E)
		return
	}

	client.Name = strings.TrimSuffix(client.Name, "\r\n")
	client.Reader = bufio.NewReader(connection)
	client.Writer = bufio.NewWriter(connection)
	client.Connection = connection
	clients = append(clients, client)

	fmt.Println(client.Name + " se conectó")
	go client.userInput(log, sendFile)
}

func (c *Client) userInput(log chan string, sendFile chan string) {
	for {
		msg, err := c.Reader.ReadString('\n')
		if err != nil {
			fmt.Println("ERROR")
			return
		}
		if strings.Contains(msg, "file") {
			readMSg = false
			c.ReceiveFileFromClient(log, sendFile)
		} else {
			fmt.Println(c.Name + ": " + msg)
			log <- c.Name + ": " + msg
		}

	}
}

func (c *Client) ReceiveFileFromClient(log chan string, sendFile chan string) {
	fileName := make([]byte, NAME_BUFFER_SIZE)
	fileSize := make([]byte, FILE_SIZE_BUFFER)
	c.Connection.Read(fileSize)
	size, _ := strconv.ParseInt(strings.Trim(string(fileSize), "$"), 10, NAME_BUFFER_SIZE)
	c.Connection.Read(fileName)
	name := strings.Trim(string(fileName), "$")
	newFile, err := os.Create(name)
	if err != nil {
		fmt.Println(err)
	}
	defer newFile.Close()
	var received int64
	for {
		if (size - received) < BUFFER_SIZE {
			io.CopyN(newFile, c.Connection, (size - received))
			c.Connection.Read(make([]byte, (received+BUFFER_SIZE)-size))
			break
		}
		io.CopyN(newFile, c.Connection, BUFFER_SIZE)
		received += BUFFER_SIZE
	}
	filePath = name
	fmt.Println(c.Name + ": envió " + name)
	log <- c.Name + ": envió " + name
	sendFile <- c.Name + ": envió " + name + "\r\n"
	readMSg = true
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
	go servidor()
	var input string
	fmt.Scanln(&input)
}
